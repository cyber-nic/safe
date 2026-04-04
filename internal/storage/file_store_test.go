package storage

import (
	"testing"

	"github.com/ndelorme/safe/internal/domain"
)

// newTestFileStore creates a FileObjectStore rooted in a temp directory that is
// cleaned up automatically when the test ends.
func newTestFileStore(t *testing.T) *FileObjectStore {
	t.Helper()
	store, err := NewFileObjectStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileObjectStore: %v", err)
	}
	return store
}

// newTestFileStoreAt creates a FileObjectStore rooted at a specific directory.
// Used by restart-survival tests that need two separate store instances backed
// by the same directory.
func newTestFileStoreAt(t *testing.T, dir string) *FileObjectStore {
	t.Helper()
	store, err := NewFileObjectStore(dir)
	if err != nil {
		t.Fatalf("NewFileObjectStore(%s): %v", dir, err)
	}
	return store
}

func TestFileObjectStorePutGet(t *testing.T) {
	store := newTestFileStore(t)

	if err := store.Put("test-key", []byte("value")); err != nil {
		t.Fatalf("put: %v", err)
	}

	got, err := store.Get("test-key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if string(got) != "value" {
		t.Fatalf("unexpected value: %s", string(got))
	}
}

func TestFileObjectStoreMissingKey(t *testing.T) {
	store := newTestFileStore(t)

	if _, err := store.Get("missing-key"); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestFileObjectStoreListByPrefix(t *testing.T) {
	store := newTestFileStore(t)

	if err := store.Put("accounts/a/collections/c/events/2.json", []byte("second")); err != nil {
		t.Fatalf("put second: %v", err)
	}
	if err := store.Put("accounts/a/collections/c/events/1.json", []byte("first")); err != nil {
		t.Fatalf("put first: %v", err)
	}
	if err := store.Put("accounts/a/collections/c/items/1.json", []byte("item")); err != nil {
		t.Fatalf("put item: %v", err)
	}

	keys, err := store.List("accounts/a/collections/c/events/")
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 event keys, got %d", len(keys))
	}

	if keys[0] != "accounts/a/collections/c/events/1.json" || keys[1] != "accounts/a/collections/c/events/2.json" {
		t.Fatalf("unexpected keys: %+v", keys)
	}
}

// TestFileObjectStoreRestartSurvival is the core acceptance test for W2.
// It writes data through one FileObjectStore instance, creates a second
// instance pointing at the same directory (simulating a process restart),
// and verifies all records can still be read.
func TestFileObjectStoreRestartSurvival(t *testing.T) {
	dir := t.TempDir()

	// --- process A: write ---
	writer, err := NewFileObjectStore(dir)
	if err != nil {
		t.Fatalf("NewFileObjectStore (writer): %v", err)
	}

	if err := writer.Put("accounts/acct1/account.json", []byte(`{"hello":"world"}`)); err != nil {
		t.Fatalf("put account: %v", err)
	}
	if err := writer.Put("accounts/acct1/collections/col1/head.json", []byte(`{"head":true}`)); err != nil {
		t.Fatalf("put head: %v", err)
	}
	if err := writer.Put("accounts/acct1/collections/col1/events/evt1.json", []byte(`{"seq":1}`)); err != nil {
		t.Fatalf("put event: %v", err)
	}
	if err := writer.Put("accounts/acct1/collections/col1/secrets/ref1.txt", []byte("s3cr3t")); err != nil {
		t.Fatalf("put secret: %v", err)
	}

	// --- process B: read (new instance, same dir) ---
	reader := newTestFileStoreAt(t, dir)

	account, err := reader.Get("accounts/acct1/account.json")
	if err != nil {
		t.Fatalf("get account after restart: %v", err)
	}
	if string(account) != `{"hello":"world"}` {
		t.Fatalf("unexpected account: %s", account)
	}

	head, err := reader.Get("accounts/acct1/collections/col1/head.json")
	if err != nil {
		t.Fatalf("get head after restart: %v", err)
	}
	if string(head) != `{"head":true}` {
		t.Fatalf("unexpected head: %s", head)
	}

	secret, err := reader.Get("accounts/acct1/collections/col1/secrets/ref1.txt")
	if err != nil {
		t.Fatalf("get secret after restart: %v", err)
	}
	if string(secret) != "s3cr3t" {
		t.Fatalf("unexpected secret: %s", secret)
	}

	keys, err := reader.List("accounts/acct1/collections/col1/events/")
	if err != nil {
		t.Fatalf("list events after restart: %v", err)
	}
	if len(keys) != 1 || keys[0] != "accounts/acct1/collections/col1/events/evt1.json" {
		t.Fatalf("unexpected event keys after restart: %v", keys)
	}
}

// TestFileObjectStoreAccountConfigRestartSurvival tests the full
// StoreAccountConfigRecord → restart → LoadAccountConfigRecord path.
func TestFileObjectStoreAccountConfigRestartSurvival(t *testing.T) {
	dir := t.TempDir()
	record := domain.StarterAccountConfigRecord()

	writer := newTestFileStoreAt(t, dir)
	if _, err := StoreAccountConfigRecord(writer, record); err != nil {
		t.Fatalf("store account config: %v", err)
	}

	reader := newTestFileStoreAt(t, dir)
	loaded, err := LoadAccountConfigRecord(reader, record.AccountID)
	if err != nil {
		t.Fatalf("load account config after restart: %v", err)
	}

	if loaded.AccountID != record.AccountID {
		t.Fatalf("account ID mismatch: want %s got %s", record.AccountID, loaded.AccountID)
	}
	if loaded.DefaultCollectionID != record.DefaultCollectionID {
		t.Fatalf("default collection ID mismatch")
	}
	if len(loaded.DeviceIDs) != len(record.DeviceIDs) {
		t.Fatalf("device IDs length mismatch")
	}
}

// TestFileObjectStoreCollectionHeadRestartSurvival tests the
// StoreCollectionHeadRecord → restart → LoadCollectionHeadRecord path.
func TestFileObjectStoreCollectionHeadRestartSurvival(t *testing.T) {
	dir := t.TempDir()
	record := domain.StarterCollectionHeadRecord()

	writer := newTestFileStoreAt(t, dir)
	if _, err := StoreCollectionHeadRecord(writer, record); err != nil {
		t.Fatalf("store collection head: %v", err)
	}

	reader := newTestFileStoreAt(t, dir)
	loaded, err := LoadCollectionHeadRecord(reader, record.AccountID, record.CollectionID)
	if err != nil {
		t.Fatalf("load collection head after restart: %v", err)
	}

	if loaded.LatestEventID != record.LatestEventID {
		t.Fatalf("latest event ID mismatch: want %s got %s", record.LatestEventID, loaded.LatestEventID)
	}
	if loaded.LatestSeq != record.LatestSeq {
		t.Fatalf("latest seq mismatch")
	}
}

// TestFileObjectStoreEventRecordsRestartSurvival tests that all event records
// survive a restart and can be listed in order.
func TestFileObjectStoreEventRecordsRestartSurvival(t *testing.T) {
	dir := t.TempDir()
	records := domain.StarterVaultEventRecords()

	writer := newTestFileStoreAt(t, dir)
	for _, r := range records {
		if _, err := StoreEventRecord(writer, r); err != nil {
			t.Fatalf("store event %s: %v", r.EventID, err)
		}
	}

	reader := newTestFileStoreAt(t, dir)
	loaded, err := LoadCollectionEventRecords(reader, records[0].AccountID, records[0].CollectionID)
	if err != nil {
		t.Fatalf("load event records after restart: %v", err)
	}

	if len(loaded) != len(records) {
		t.Fatalf("expected %d event records, got %d", len(records), len(loaded))
	}

	for i, r := range records {
		if loaded[i].EventID != r.EventID {
			t.Fatalf("event[%d] ID mismatch: want %s got %s", i, r.EventID, loaded[i].EventID)
		}
		if loaded[i].Sequence != r.Sequence {
			t.Fatalf("event[%d] sequence mismatch", i)
		}
	}
}

// TestFileObjectStoreSecretMaterialRestartSurvival tests that secret material
// survives a restart.
func TestFileObjectStoreSecretMaterialRestartSurvival(t *testing.T) {
	dir := t.TempDir()

	const (
		accountID    = "acct-dev-001"
		collectionID = "vault-personal"
		secretRef    = "vault-secret://totp/gmail-primary"
		secretValue  = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	)

	writer := newTestFileStoreAt(t, dir)
	if _, err := StoreSecretMaterial(writer, accountID, collectionID, secretRef, secretValue); err != nil {
		t.Fatalf("store secret material: %v", err)
	}

	reader := newTestFileStoreAt(t, dir)
	loaded, err := LoadSecretMaterial(reader, accountID, collectionID, secretRef)
	if err != nil {
		t.Fatalf("load secret material after restart: %v", err)
	}

	if loaded != secretValue {
		t.Fatalf("unexpected secret material: %s", loaded)
	}
}

// TestFileObjectStoreItemRecordRestartSurvival tests that item records survive
// a restart.
func TestFileObjectStoreItemRecordRestartSurvival(t *testing.T) {
	dir := t.TempDir()
	record := domain.StarterVaultItemRecords()[0]

	const (
		accountID    = "acct-dev-001"
		collectionID = "vault-personal"
	)

	writer := newTestFileStoreAt(t, dir)
	if _, err := StoreItemRecord(writer, accountID, collectionID, record); err != nil {
		t.Fatalf("store item record: %v", err)
	}

	reader := newTestFileStoreAt(t, dir)
	loaded, err := LoadItemRecord(reader, accountID, collectionID, record.Item.ID)
	if err != nil {
		t.Fatalf("load item record after restart: %v", err)
	}

	if loaded.Item.ID != record.Item.ID {
		t.Fatalf("item ID mismatch: want %s got %s", record.Item.ID, loaded.Item.ID)
	}
	if loaded.Item.Kind != record.Item.Kind {
		t.Fatalf("item kind mismatch")
	}
}

// TestFileObjectStorePutOverwrite verifies that a second Put replaces the
// first value rather than appending to it.
func TestFileObjectStorePutOverwrite(t *testing.T) {
	store := newTestFileStore(t)

	if err := store.Put("k", []byte("first")); err != nil {
		t.Fatalf("first put: %v", err)
	}
	if err := store.Put("k", []byte("second")); err != nil {
		t.Fatalf("second put: %v", err)
	}

	got, err := store.Get("k")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got) != "second" {
		t.Fatalf("expected second, got %s", string(got))
	}
}
