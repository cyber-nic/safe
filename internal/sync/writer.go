package sync

import (
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
// write protocol from SYSTEM_DESIGN §11.
type SyncWriter struct {
	store      storage.ObjectStoreWithCAS
	verifyHead VerifyHeadFunc
	signHead   HeadSignerFunc
}

// NewSyncWriter returns a SyncWriter backed by store. If verifyHead is nil the
// default signed-head verifier is used.
func NewSyncWriter(store storage.ObjectStoreWithCAS, verifyHead VerifyHeadFunc, signHead HeadSignerFunc) *SyncWriter {
	if verifyHead == nil {
		verifyHead = VerifySignedCollectionHead
	}
	return &SyncWriter{store: store, verifyHead: verifyHead, signHead: signHead}
}

// CommitSyncMutation writes m to the shared object store, retrying up to
// maxCASRetries times on CAS conflict. It returns an error if the operation
// cannot succeed after the retry budget is exhausted.
func (w *SyncWriter) CommitSyncMutation(m SyncMutation) error {
	if m.AccountID == "" {
		return fmt.Errorf("sync commit: accountID is required")
	}
	if m.CollectionID == "" {
		return fmt.Errorf("sync commit: collectionID is required")
	}
	if w.signHead == nil {
		return fmt.Errorf("sync commit: head signer is required")
	}
	if err := m.EventRecord.Validate(); err != nil {
		return fmt.Errorf("sync commit: invalid event record: %w", err)
	}

	for attempt := 0; attempt < maxCASRetries; attempt++ {
		currentSignedHead, currentETag, err := loadSignedHeadWithETag(w.store, m.AccountID, m.CollectionID)
		if err != nil && !storage.IsObjectNotFound(err) {
			return fmt.Errorf("sync commit: read head: %w", err)
		}
		currentHead := currentSignedHead.Record

		if currentETag != "" {
			latestEvent, authoringDevice, err := loadHeadVerificationContext(w.store, currentSignedHead.Record)
			if err != nil {
				return fmt.Errorf("sync commit: load head verification context: %w", err)
			}
			if verifyErr := w.verifyHead(currentHead, currentSignedHead, latestEvent, authoringDevice); verifyErr != nil {
				return fmt.Errorf("sync commit: head verification: %w", verifyErr)
			}
		}

		if currentETag != "" && currentHead.LatestEventID == m.EventRecord.EventID {
			return nil
		}

		expectedSeq := 1
		if currentETag != "" {
			expectedSeq = currentHead.LatestSeq + 1
		}
		if m.EventRecord.Sequence != expectedSeq {
			return fmt.Errorf("sync commit: event sequence %d does not match expected %d (head seq %d)",
				m.EventRecord.Sequence, expectedSeq, currentHead.LatestSeq)
		}

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

		eventPayload, err := m.EventRecord.CanonicalJSON()
		if err != nil {
			return fmt.Errorf("sync commit: marshal event record: %w", err)
		}
		eventKey := storage.EventObjectKey(m.AccountID, m.CollectionID, m.EventRecord.EventID)
		if err := w.store.Put(eventKey, eventPayload); err != nil {
			return fmt.Errorf("sync commit: upload event record: %w", err)
		}

		newHead := domain.CollectionHeadRecord{
			SchemaVersion: 1,
			AccountID:     m.AccountID,
			CollectionID:  m.CollectionID,
			LatestEventID: m.EventRecord.EventID,
			LatestSeq:     m.EventRecord.Sequence,
		}
		signedHead, err := w.signHead(newHead)
		if err != nil {
			return fmt.Errorf("sync commit: sign new head: %w", err)
		}
		headPayload, err := signedHead.CanonicalJSON()
		if err != nil {
			return fmt.Errorf("sync commit: marshal signed head: %w", err)
		}
		headKey := storage.CollectionHeadKey(m.AccountID, m.CollectionID)
		_, casErr := w.store.PutIfMatch(headKey, headPayload, currentETag)
		if casErr == nil {
			return nil
		}

		if !errors.Is(casErr, storage.ErrCASConflict) {
			return fmt.Errorf("sync commit: advance head: %w", casErr)
		}
	}

	return fmt.Errorf("sync commit: exceeded %d CAS retries for %s/%s", maxCASRetries, m.AccountID, m.CollectionID)
}

// loadSignedHeadWithETag reads the signed collection head from the store and returns its ETag.
// Returns a zero-value head and "" ETag when the head does not yet exist.
func loadSignedHeadWithETag(store storage.ObjectStoreWithCAS, accountID, collectionID string) (SignedCollectionHead, string, error) {
	key := storage.CollectionHeadKey(accountID, collectionID)
	data, etag, err := store.GetWithETag(key)
	if err != nil {
		return SignedCollectionHead{}, "", err
	}
	head, err := ParseSignedCollectionHeadJSON(data)
	if err != nil {
		return SignedCollectionHead{}, "", fmt.Errorf("unmarshal signed head: %w", err)
	}
	return head, etag, nil
}

func loadHeadVerificationContext(store storage.ObjectStore, head domain.CollectionHeadRecord) (domain.VaultEventRecord, domain.LocalDeviceRecord, error) {
	latestEvent, err := storage.LoadEventRecord(store, head.AccountID, head.CollectionID, head.LatestEventID)
	if err != nil {
		return domain.VaultEventRecord{}, domain.LocalDeviceRecord{}, err
	}

	devicePayload, err := store.Get(deviceRecordKey(head.AccountID, latestEvent.DeviceID))
	if err != nil {
		return domain.VaultEventRecord{}, domain.LocalDeviceRecord{}, err
	}
	deviceRecord, err := domain.ParseLocalDeviceRecordJSON(devicePayload)
	if err != nil {
		return domain.VaultEventRecord{}, domain.LocalDeviceRecord{}, err
	}

	return latestEvent, deviceRecord, nil
}

func deviceRecordKey(accountID, deviceID string) string {
	return fmt.Sprintf("accounts/%s/devices/%s.json", accountID, deviceID)
}
