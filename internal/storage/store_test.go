package storage

import (
	"testing"

	"github.com/ndelorme/safe/internal/domain"
)

func TestMemoryObjectStorePutGet(t *testing.T) {
	store := NewMemoryObjectStore()

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

func TestMemoryObjectStoreMissingKey(t *testing.T) {
	store := NewMemoryObjectStore()

	if _, err := store.Get("missing-key"); err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestMemoryObjectStoreListByPrefix(t *testing.T) {
	store := NewMemoryObjectStore()

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

func TestStoreItemRecord(t *testing.T) {
	store := NewMemoryObjectStore()
	record := domain.StarterVaultItemRecords()[1]

	key, err := StoreItemRecord(store, "acct-dev-001", "vault-personal", record)
	if err != nil {
		t.Fatalf("store item record: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("get stored item record: %v", err)
	}

	want, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical item record: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("unexpected stored item record\nwant: %s\ngot:  %s", string(want), string(got))
	}

	loaded, err := LoadItemRecord(store, "acct-dev-001", "vault-personal", record.Item.ID)
	if err != nil {
		t.Fatalf("load item record: %v", err)
	}

	if loaded.Item.ID != record.Item.ID {
		t.Fatalf("unexpected loaded item record: %+v", loaded)
	}
}

func TestStoreEventRecord(t *testing.T) {
	store := NewMemoryObjectStore()
	record := domain.StarterVaultEventRecords()[0]

	key, err := StoreEventRecord(store, record)
	if err != nil {
		t.Fatalf("store event record: %v", err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("get stored event record: %v", err)
	}

	want, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonical event record: %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("unexpected stored event record\nwant: %s\ngot:  %s", string(want), string(got))
	}

	loaded, err := LoadEventRecord(store, record.AccountID, record.CollectionID, record.EventID)
	if err != nil {
		t.Fatalf("load event record: %v", err)
	}

	if loaded.EventID != record.EventID {
		t.Fatalf("unexpected loaded event record: %+v", loaded)
	}
}

func TestLoadCollectionEventRecords(t *testing.T) {
	store := NewMemoryObjectStore()
	records := domain.StarterVaultEventRecords()

	for _, record := range records {
		if _, err := StoreEventRecord(store, record); err != nil {
			t.Fatalf("store event record: %v", err)
		}
	}

	loaded, err := LoadCollectionEventRecords(store, "acct-dev-001", "vault-personal")
	if err != nil {
		t.Fatalf("load collection event records: %v", err)
	}

	if len(loaded) != len(records) {
		t.Fatalf("expected %d records, got %d", len(records), len(loaded))
	}

	if loaded[0].EventID != records[0].EventID || loaded[1].EventID != records[1].EventID {
		t.Fatalf("unexpected loaded records: %+v", loaded)
	}
}

func TestStoreAndLoadCollectionHeadRecord(t *testing.T) {
	store := NewMemoryObjectStore()
	record := domain.StarterCollectionHeadRecord()

	key, err := StoreCollectionHeadRecord(store, record)
	if err != nil {
		t.Fatalf("store collection head record: %v", err)
	}

	if key != "accounts/acct-dev-001/collections/vault-personal/head.json" {
		t.Fatalf("unexpected collection head key: %s", key)
	}

	loaded, err := LoadCollectionHeadRecord(store, "acct-dev-001", "vault-personal")
	if err != nil {
		t.Fatalf("load collection head record: %v", err)
	}

	if loaded.LatestEventID != record.LatestEventID || loaded.LatestSeq != record.LatestSeq {
		t.Fatalf("unexpected loaded collection head: %+v", loaded)
	}
}
