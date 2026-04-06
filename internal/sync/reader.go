package sync

import (
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
)

// SyncReader loads incremental state from a shared object store and produces a
// local Projection. It enforces the W12 signed-head trust boundary from I7
// before applying any events.
type SyncReader struct {
	store      storage.ObjectStoreWithCAS
	verifyHead VerifyHeadFunc
}

// NewSyncReader returns a SyncReader backed by store. If verifyHead is nil
// VerifySignedCollectionHead is used.
func NewSyncReader(store storage.ObjectStoreWithCAS, verifyHead VerifyHeadFunc) *SyncReader {
	if verifyHead == nil {
		verifyHead = VerifySignedCollectionHead
	}
	return &SyncReader{store: store, verifyHead: verifyHead}
}

// IncrementalSync fetches all events for (accountID, collectionID) after since
// (exclusive), verifies the current head, loads the new events, and replays
// them. It returns an error if the head fails signature or freshness checks,
// if event continuity is broken, or if replay invariants are violated.
func (r *SyncReader) IncrementalSync(accountID, collectionID string, since int) (Projection, error) {
	if accountID == "" {
		return Projection{}, fmt.Errorf("incremental sync: accountID is required")
	}
	if collectionID == "" {
		return Projection{}, fmt.Errorf("incremental sync: collectionID is required")
	}

	currentSignedHead, _, err := loadSignedHeadWithETag(r.store, accountID, collectionID)
	if err != nil {
		if storage.IsObjectNotFound(err) {
			return Projection{AccountID: accountID, CollectionID: collectionID, Items: map[string]domain.VaultItemRecord{}}, nil
		}
		return Projection{}, fmt.Errorf("incremental sync: read head: %w", err)
	}
	currentHead := currentSignedHead.Record
	latestEvent, authoringDevice, err := loadHeadVerificationContext(r.store, currentHead)
	if err != nil {
		return Projection{}, fmt.Errorf("incremental sync: load head verification context: %w", err)
	}

	trustedHead := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     accountID,
		CollectionID:  collectionID,
		LatestSeq:     since,
	}
	if err := r.verifyHead(trustedHead, currentSignedHead, latestEvent, authoringDevice); err != nil {
		return Projection{}, fmt.Errorf("incremental sync: head verification: %w", err)
	}

	if currentHead.LatestSeq == since {
		return Projection{AccountID: accountID, CollectionID: collectionID, Items: map[string]domain.VaultItemRecord{}, LatestSeq: since}, nil
	}

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
		ev, err := domain.ParseVaultEventRecordJSON(data)
		if err != nil {
			return Projection{}, fmt.Errorf("incremental sync: parse event %s: %w", key, err)
		}
		events = append(events, ev)
	}

	projection, err := ReplayCollectionAgainstHead(events, currentHead)
	if err != nil {
		return Projection{}, fmt.Errorf("incremental sync: replay: %w", err)
	}

	return projection, nil
}
