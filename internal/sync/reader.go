package sync

import (
	"encoding/json"
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
)

// SyncReader loads incremental state from a shared object store and produces a
// local Projection. It enforces the monotonicity and (eventually, via W12)
// signature rules from I7 before applying any events.
type SyncReader struct {
	store      storage.ObjectStoreWithCAS
	verifyHead VerifyHeadFunc
}

// NewSyncReader returns a SyncReader backed by store. If verifyHead is nil
// MonotonicVerifyHead is used.
func NewSyncReader(store storage.ObjectStoreWithCAS, verifyHead VerifyHeadFunc) *SyncReader {
	if verifyHead == nil {
		verifyHead = MonotonicVerifyHead
	}
	return &SyncReader{store: store, verifyHead: verifyHead}
}

// IncrementalSync fetches all events for (accountID, collectionID) after since
// (exclusive), verifies the current head, loads the new events, and replays
// them. It returns an error if the head fails monotonicity/signature checks,
// if event continuity is broken, or if replay invariants are violated.
//
// A caller that has no local state passes since=0 to load the full history.
// The returned Projection reflects state up to the current committed head.
func (r *SyncReader) IncrementalSync(accountID, collectionID string, since int) (Projection, error) {
	if accountID == "" {
		return Projection{}, fmt.Errorf("incremental sync: accountID is required")
	}
	if collectionID == "" {
		return Projection{}, fmt.Errorf("incremental sync: collectionID is required")
	}

	// Fetch and verify the current head.
	currentHead, _, err := loadHeadWithETag(r.store, accountID, collectionID)
	if err != nil {
		if storage.IsObjectNotFound(err) {
			// No head yet — collection is empty.
			return Projection{AccountID: accountID, CollectionID: collectionID, Items: make(map[string]domain.VaultItemRecord)}, nil
		}
		return Projection{}, fmt.Errorf("incremental sync: read head: %w", err)
	}

	// Build a synthetic "trusted" head representing the caller's last known
	// position so we can check the fetched head is a valid advancement.
	trustedHead := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     accountID,
		CollectionID:  collectionID,
		LatestSeq:     since,
	}
	if err := r.verifyHead(trustedHead, currentHead); err != nil {
		return Projection{}, fmt.Errorf("incremental sync: head verification: %w", err)
	}

	if currentHead.LatestSeq == since {
		// Already up to date.
		return Projection{AccountID: accountID, CollectionID: collectionID, Items: make(map[string]domain.VaultItemRecord), LatestSeq: since}, nil
	}

	// Load all event keys under the collection prefix.
	// Replay always starts from seq=1 (no snapshot support in W15); all events
	// are loaded and replayed from scratch. The since parameter is only used for
	// the "already up to date" fast-path above.
	eventPrefix := storage.EventPrefix(accountID, collectionID)
	keys, err := r.store.List(eventPrefix)
	if err != nil {
		return Projection{}, fmt.Errorf("incremental sync: list events: %w", err)
	}

	events := make([]domain.VaultEventRecord, 0, len(keys))
	for _, key := range keys {
		data, err := r.store.Get(key)
		if err != nil {
			return Projection{}, fmt.Errorf("incremental sync: load event %s: %w", key, err)
		}
		var ev domain.VaultEventRecord
		if err := json.Unmarshal(data, &ev); err != nil {
			return Projection{}, fmt.Errorf("incremental sync: unmarshal event %s: %w", key, err)
		}
		events = append(events, ev)
	}

	// ReplayCollectionAgainstHead verifies sequence continuity, account/collection
	// binding, and that the final cursor matches the committed head.
	projection, err := ReplayCollectionAgainstHead(events, currentHead)
	if err != nil {
		return Projection{}, fmt.Errorf("incremental sync: replay: %w", err)
	}

	return projection, nil
}
