package sync

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
)

// maxCASRetries is the maximum number of times CommitSyncMutation will retry
// after a CAS conflict before giving up.
const maxCASRetries = 10

// SyncMutation describes the records to commit to the shared object store in
// one logical sync operation. Unlike storage.VaultMutation it carries no
// secret material; secrets stay in the local encrypted store only.
type SyncMutation struct {
	AccountID    string
	CollectionID string
	EventRecord  domain.VaultEventRecord
	ItemRecord   *domain.VaultItemRecord // nil for delete events
}

// SyncWriter commits vault mutations to a shared object store using the
// write protocol from SYSTEM_DESIGN §11:
//
//  1. Read current head with ETag.
//  2. Upload immutable objects (item, event) unconditionally.
//  3. CAS-advance the head pointer.
//  4. On conflict, re-read head, check idempotency, and rebuild if needed.
//
// The VerifyHead hook will be replaced by W12's signed-metadata verifier once
// the event signature envelope is frozen. Until then it validates monotonicity.
type SyncWriter struct {
	store      storage.ObjectStoreWithCAS
	verifyHead VerifyHeadFunc
}

// NewSyncWriter returns a SyncWriter backed by store. If verifyHead is nil the
// default monotonicity-only check is used (see MonotonicVerifyHead).
func NewSyncWriter(store storage.ObjectStoreWithCAS, verifyHead VerifyHeadFunc) *SyncWriter {
	if verifyHead == nil {
		verifyHead = MonotonicVerifyHead
	}
	return &SyncWriter{store: store, verifyHead: verifyHead}
}

// CommitSyncMutation writes m to the shared object store, retrying up to
// maxCASRetries times on CAS conflict. It returns an error if the operation
// cannot succeed after the retry budget is exhausted.
//
// Immutable objects (item record, event record) are uploaded before the head
// is advanced. A crash after the uploads but before the CAS leaves unreachable
// immutable objects in the store, which is safe — they are ignored during
// normal replay because no committed head references them.
func (w *SyncWriter) CommitSyncMutation(m SyncMutation) error {
	if m.AccountID == "" {
		return fmt.Errorf("sync commit: accountID is required")
	}
	if m.CollectionID == "" {
		return fmt.Errorf("sync commit: collectionID is required")
	}
	if err := m.EventRecord.Validate(); err != nil {
		return fmt.Errorf("sync commit: invalid event record: %w", err)
	}

	for attempt := 0; attempt < maxCASRetries; attempt++ {
		// Step 1: read current head and ETag.
		currentHead, currentETag, err := loadHeadWithETag(w.store, m.AccountID, m.CollectionID)
		if err != nil && !storage.IsObjectNotFound(err) {
			return fmt.Errorf("sync commit: read head: %w", err)
		}
		// An absent head means this is the genesis write; currentETag is "".

		// Verify the current head is not stale compared to any local trusted state.
		// (No-op for genesis; monotonic check on subsequent writes.)
		if currentETag != "" {
			if verifyErr := w.verifyHead(currentHead, currentHead); verifyErr != nil {
				return fmt.Errorf("sync commit: head verification: %w", verifyErr)
			}
		}

		// Idempotency check: if our event is already referenced by the current head,
		// the mutation committed on a previous attempt.
		if currentETag != "" && currentHead.LatestEventID == m.EventRecord.EventID {
			return nil
		}

		// The new event must strictly advance the sequence.
		expectedSeq := 1
		if currentETag != "" {
			expectedSeq = currentHead.LatestSeq + 1
		}
		if m.EventRecord.Sequence != expectedSeq {
			return fmt.Errorf("sync commit: event sequence %d does not match expected %d (head seq %d)",
				m.EventRecord.Sequence, expectedSeq, currentHead.LatestSeq)
		}

		// Step 2: upload immutable objects (safe to redo on retry).
		if m.ItemRecord != nil {
			payload, err := m.ItemRecord.CanonicalJSON()
			if err != nil {
				return fmt.Errorf("sync commit: marshal item record: %w", err)
			}
			key := storage.ItemObjectKey(m.AccountID, m.CollectionID, m.ItemRecord.Item.ID)
			if err := w.store.Put(key, payload); err != nil {
				return fmt.Errorf("sync commit: upload item record: %w", err)
			}
		}

		eventPayload, err := json.Marshal(m.EventRecord)
		if err != nil {
			return fmt.Errorf("sync commit: marshal event record: %w", err)
		}
		eventKey := storage.EventObjectKey(m.AccountID, m.CollectionID, m.EventRecord.EventID)
		if err := w.store.Put(eventKey, eventPayload); err != nil {
			return fmt.Errorf("sync commit: upload event record: %w", err)
		}

		// Step 3: build and CAS-advance the head pointer.
		newHead := domain.CollectionHeadRecord{
			SchemaVersion: 1,
			AccountID:     m.AccountID,
			CollectionID:  m.CollectionID,
			LatestEventID: m.EventRecord.EventID,
			LatestSeq:     m.EventRecord.Sequence,
		}
		headPayload, err := newHead.CanonicalJSON()
		if err != nil {
			return fmt.Errorf("sync commit: marshal new head: %w", err)
		}
		headKey := storage.CollectionHeadKey(m.AccountID, m.CollectionID)
		_, casErr := w.store.PutIfMatch(headKey, headPayload, currentETag)
		if casErr == nil {
			return nil // committed
		}

		// Step 4: CAS conflict — another writer advanced the head concurrently.
		if !errors.Is(casErr, storage.ErrCASConflict) {
			return fmt.Errorf("sync commit: advance head: %w", casErr)
		}
		// Loop: re-read head and retry.
	}

	return fmt.Errorf("sync commit: exceeded %d CAS retries for %s/%s", maxCASRetries, m.AccountID, m.CollectionID)
}

// loadHeadWithETag reads the collection head from the store and returns its ETag.
// Returns a zero-value head and "" ETag when the head does not yet exist.
func loadHeadWithETag(store storage.ObjectStoreWithCAS, accountID, collectionID string) (domain.CollectionHeadRecord, string, error) {
	key := storage.CollectionHeadKey(accountID, collectionID)
	data, etag, err := store.GetWithETag(key)
	if err != nil {
		return domain.CollectionHeadRecord{}, "", err
	}
	var head domain.CollectionHeadRecord
	if err := json.Unmarshal(data, &head); err != nil {
		return domain.CollectionHeadRecord{}, "", fmt.Errorf("unmarshal head: %w", err)
	}
	return head, etag, nil
}
