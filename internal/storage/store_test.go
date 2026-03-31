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
}
