package sync

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	internalcrypto "github.com/ndelorme/safe/internal/crypto"
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

func mustStoreActiveDevice(t *testing.T, store storage.ObjectStore, accountID, deviceID string) ed25519.PrivateKey {
	t.Helper()

	keyPair, err := internalcrypto.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}
	deviceRecord, err := internalcrypto.CreateDeviceRecord(accountID, deviceID, "Sync test device", "cli", keyPair)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}
	payload, err := deviceRecord.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical device record: %v", err)
	}
	if err := store.Put(deviceRecordKey(accountID, deviceID), payload); err != nil {
		t.Fatalf("store device record: %v", err)
	}

	return keyPair.SigningPrivateKey
}

func newSignedWriter(store *storage.MemoryObjectStoreWithCAS, signingKey ed25519.PrivateKey) *SyncWriter {
	return NewSyncWriter(store, nil, func(head domain.CollectionHeadRecord) (SignedCollectionHead, error) {
		return SignCollectionHead(head, signingKey)
	})
}

// --- tests ---

func TestSyncWriterSingleCommit(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writer := newSignedWriter(store, signingKey)
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

func TestTwoRuntimesConverge(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writerA := newSignedWriter(store, signingKey)
	writerB := newSignedWriter(store, signingKey)
	reader := NewSyncReader(store, nil)

	if err := writerA.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("A commit seq=1: %v", err)
	}
	if err := writerA.CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("A commit seq=2: %v", err)
	}

	projB, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("B sync: %v", err)
	}
	if projB.LatestSeq != 2 {
		t.Fatalf("B expected LatestSeq=2, got %d", projB.LatestSeq)
	}

	if err := writerB.CommitSyncMutation(makeMutation(3, "item-c", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("B commit seq=3: %v", err)
	}

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

func TestInterruptSafetyImmutablesWithoutHead(t *testing.T) {
	store := newStore()
	reader := NewSyncReader(store, nil)

	ev := makeEvent(1, "orphan-item", domain.VaultEventActionPutItem)
	evBytes, err := ev.CanonicalJSON()
	if err != nil {
		t.Fatalf("marshal orphan event: %v", err)
	}
	if err := store.Put(storage.EventObjectKey(testAccountID, testCollectionID, ev.EventID), evBytes); err != nil {
		t.Fatalf("upload orphan event: %v", err)
	}

	proj, err := reader.IncrementalSync(testAccountID, testCollectionID, 0)
	if err != nil {
		t.Fatalf("sync after orphan upload: %v", err)
	}
	if len(proj.Items) != 0 {
		t.Fatalf("expected empty projection with no head, got %d items", len(proj.Items))
	}
}

func TestStaleHeadRejected(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writer := newSignedWriter(store, signingKey)

	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit seq=1: %v", err)
	}
	if err := writer.CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit seq=2: %v", err)
	}

	_, err := NewSyncReader(store, nil).IncrementalSync(testAccountID, testCollectionID, 3)
	if err == nil {
		t.Fatal("expected error when head is behind caller cursor, got nil")
	}
}

func TestIdempotentCommit(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writer := newSignedWriter(store, signingKey)
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

func TestCASConflictIsDetected(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)

	writer := newSignedWriter(store, signingKey)
	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("setup commit: %v", err)
	}

	headKey := storage.CollectionHeadKey(testAccountID, testCollectionID)
	_, etag, err := store.GetWithETag(headKey)
	if err != nil {
		t.Fatalf("get head ETag: %v", err)
	}

	if err := newSignedWriter(store, signingKey).CommitSyncMutation(makeMutation(2, "item-b", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("concurrent commit: %v", err)
	}

	_, casErr := store.PutIfMatch(headKey, []byte(`{}`), etag)
	if !errors.Is(casErr, storage.ErrCASConflict) {
		t.Fatalf("expected ErrCASConflict, got %v", casErr)
	}
}

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

func TestDeleteEventConverges(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writer := newSignedWriter(store, signingKey)

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

func TestReaderRejectsUnsignedHead(t *testing.T) {
	store := newStore()
	signingKey := mustStoreActiveDevice(t, store, testAccountID, testDeviceID)
	writer := newSignedWriter(store, signingKey)

	if err := writer.CommitSyncMutation(makeMutation(1, "item-a", domain.VaultEventActionPutItem)); err != nil {
		t.Fatalf("commit: %v", err)
	}

	headKey := storage.CollectionHeadKey(testAccountID, testCollectionID)
	payload, _, err := store.GetWithETag(headKey)
	if err != nil {
		t.Fatalf("get signed head: %v", err)
	}
	var signedHead SignedCollectionHead
	if err := json.Unmarshal(payload, &signedHead); err != nil {
		t.Fatalf("unmarshal signed head: %v", err)
	}
	signedHead.Signature = ""
	unsignedPayload, err := json.Marshal(signedHead)
	if err != nil {
		t.Fatalf("marshal unsigned head: %v", err)
	}
	if err := store.Put(headKey, unsignedPayload); err != nil {
		t.Fatalf("overwrite unsigned head: %v", err)
	}

	_, err = NewSyncReader(store, nil).IncrementalSync(testAccountID, testCollectionID, 0)
	if err == nil {
		t.Fatal("expected unsigned head rejection")
	}
}
