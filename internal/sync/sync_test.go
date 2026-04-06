package sync

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ndelorme/safe/internal/domain"
	"github.com/ndelorme/safe/internal/storage"
)

// --- helpers ---

const (
	testAccountID    = "acct-sync-test-001"
	testCollectionID = "coll-sync-test-001"
	testDeviceID     = "device-a"
)

func makeEvent(seq int, itemID string, action domain.VaultEventAction) domain.VaultEventRecord {
	ev := domain.VaultEventRecord{
		SchemaVersion: 1,
		EventID:       fmt.Sprintf("evt-%s-v%d", itemID, seq),
		AccountID:     testAccountID,
		DeviceID:      testDeviceID,
		CollectionID:  testCollectionID,
		Sequence:      seq,
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
		Action:        action,
	}
	if action == domain.VaultEventActionPutItem {
		ev.ItemRecord = domain.VaultItemRecord{
			SchemaVersion: 1,
			Item: domain.VaultItem{
				ID:          itemID,
				Kind:        domain.VaultItemKindNote,
				Title:       "Test " + itemID,
				Tags:        []string{},
				BodyPreview: "preview",
			},
		}
	} else {
		ev.ItemID = itemID
	}
	return ev
}

func makeMutation(seq int, itemID string, action domain.VaultEventAction) SyncMutation {
	ev := makeEvent(seq, itemID, action)
	m := SyncMutation{
		AccountID:    testAccountID,
		CollectionID: testCollectionID,
		EventRecord:  ev,
	}
	if action == domain.VaultEventActionPutItem {
		ir := ev.ItemRecord
		m.ItemRecord = &ir
	}
	return m
}

func newStore() *storage.MemoryObjectStoreWithCAS {
	return storage.NewMemoryObjectStoreWithCAS()
}

// --- tests ---

// TestSyncWriterSingleCommit verifies a single mutation is readable after commit.
func TestSyncWriterSingleCommit(t *testing.T) {
	store := newStore()
	writer := NewSyncWriter(store, nil)
	reader := NewSyncReader(store, nil)

	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit: %v", err)
	}

	proj, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, ok := proj.Items["item-a"]; !ok {
		t.Fatal("expected item-a in projection")
	}
	if proj.LatestSeq != 1 {
		t.Fatalf("expected LatestSeq=1, got %d", proj.LatestSeq)
	}
}

// TestTwoRuntimesConverge verifies that two independent SyncWriters sharing
// one object store produce an identical projection when both sync.
func TestTwoRuntimesConverge(t *testing.T) {
	store := newStore()
	writerA := NewSyncWriter(store, nil)
	writerB := NewSyncWriter(store, nil)
	reader := NewSyncReader(store, nil)

	// Runtime A writes seq=1 and seq=2.
	if err := writerA.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("A commit seq=1: %v", err)
	}
	if err := writerA.CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("A commit seq=2: %v", err)
	}

	// Runtime B syncs to confirm it sees A's writes.
	projB, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("B sync: %v", err)
	}
	if projB.LatestSeq != 2 {
		t.Fatalf("B expected LatestSeq=2, got %d", projB.LatestSeq)
	}

	// Runtime B appends seq=3.
	if err := writerB.CommitSyncMutation(makeMutation(3, "item-c", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("B commit seq=3: %v", err)
	}

	// Final sync sees all three events converged.
	finalProj, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("final sync: %v", err)
	}
	if finalProj.LatestSeq != 3 {
		t.Fatalf("expected LatestSeq=3, got %d", finalProj.LatestSeq)
	}
	for _, itemID := range []string{"item-a", "item-b", "item-c"} {
		if _, ok := finalProj.Items[itemID]; !ok {
			t.Errorf("expected %s in final projection", itemID)
		}
	}
}

// TestInterruptSafetyImmutablesWithoutHead verifies that immutable objects
// uploaded without a head advancement are ignored during replay. The head is
// the commit point; unreachable objects must not affect the projection.
func TestInterruptSafetyImmutablesWithoutHead(t *testing.T) {
	store := newStore()
	reader := NewSyncReader(store, nil)

	// Manually upload an event without advancing the head.
	// This simulates a crash after step 2 (immutable upload) but before CAS.
	ev := makeEvent(1, "orphan-item", domain.VaultEventActionPutItem)
	evBytes, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal orphan event: %v", err)
	}
	if err := store.Put(storage.EventObjectKey(testAccountID, testCollectionID, ev.EventID), evBytes); err != nil {
		t.Fatalf("upload orphan event: %v", err)
	}

	// No head was written — the projection must be empty.
	proj, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync after orphan upload: %v", err)
	}
	if len(proj.Items) != 0 {
		t.Fatalf("expected empty projection with no head, got %d items", len(proj.Items))
	}
}

// TestStaleHeadRejected verifies that IncrementalSync rejects a candidate head
// that is behind the caller's claimed cursor (since value).
func TestStaleHeadRejected(t *testing.T) {
	store := newStore()
	writer := NewSyncWriter(store, nil)

	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit seq=1: %v", err)
	}
	if err := writer.CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit seq=2: %v", err)
	}

	// Request sync with since=3 (ahead of the committed head at seq=2).
	// The head (seq=2) is stale relative to the caller's cursor (3).
	_, err := NewSyncReader(store, nil).IncrementalSync(testAccountID, testCollectionID, 3)
	if err == nil {
		t.Fatal("expected error when head is behind caller cursor, got nil")
	}
}

// TestIdempotentCommit verifies that committing the same event twice succeeds
// on the second attempt without error or duplication.
func TestIdempotentCommit(t *testing.T) {
	store := newStore()
	writer := NewSyncWriter(store, nil)
	m := makeMutation(1, "item-a", domain.VaultEventActionPutItem)

	if err := writer.CommitSyncMutation(m); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if err := writer.CommitSyncMutation(m); err != nil {
		t.Fatalf("second commit (idempotent): %v", err)
	}

	proj, err := NewSyncReader(store, nil).IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync after idempotent commits: %v", err)
	}
	if proj.LatestSeq != 1 {
		t.Fatalf("expected LatestSeq=1 after idempotent commits, got %d", proj.LatestSeq)
	}
}

// TestCASConflictIsDetected verifies ErrCASConflict is returned when a stale
// ETag is presented to PutIfMatch.
func TestCASConflictIsDetected(t *testing.T) {
	store := newStore()

	// Write seq=1 so we have an initial ETag.
	writer := NewSyncWriter(store, nil)
	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("setup commit: %v", err)
	}

	headKey := storage.CollectionHeadKey(testAccountID, testCollectionID)
	_, etag, err := store.GetWithETag(headKey)
	if err != nil {
		t.Fatalf("get head ETag: %v", err)
	}

	// A concurrent writer advances the head.
	if err := NewSyncWriter(store, nil).CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("concurrent commit: %v", err)
	}

	// The stale ETag must now cause a CAS conflict.
	_, casErr := store.PutIfMatch(headKey, []byte(`{}`), etag)
	if !errors.Is(casErr, storage.ErrCASConflict) {
		t.Fatalf("expected ErrCASConflict, got %v", casErr)
	}
}

// TestEmptyCollectionSync verifies sync on a collection with no committed
// events returns an empty projection without error.
func TestEmptyCollectionSync(t *testing.T) {
	store := newStore()
	reader := NewSyncReader(store, nil)

	proj, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync on empty collection: %v", err)
	}
	if proj.LatestSeq != 0 {
		t.Fatalf("expected LatestSeq=0 for empty collection, got %d", proj.LatestSeq)
	}
	if len(proj.Items) != 0 {
		t.Fatalf("expected 0 items for empty collection, got %d", len(proj.Items))
	}
}

// TestDeleteEventConverges verifies that a delete event removes the item from
// the projection in both runtimes.
func TestDeleteEventConverges(t *testing.T) {
	store := newStore()
	writer := NewSyncWriter(store, nil)

	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := writer.CommitSyncMutation(makeMutation(2, "item-a", domain.VaultEventActionDeleteItem)); err != nil {
		t.Fatalf("delete: %v", err)
	}

	proj, err := NewSyncReader(store, nil).IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if _, ok := proj.Items["item-a"]; ok {
		t.Fatal("deleted item-a should not appear in projection")
	}
	if proj.LatestSeq != 2 {
		t.Fatalf("expected LatestSeq=2, got %d", proj.LatestSeq)
	}
}
