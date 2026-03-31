package storage

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/ndelorme/safe/internal/domain"
)

type ObjectStore interface {
	Put(key string, value []byte) error
	Get(key string) ([]byte, error)
	List(prefix string) ([]string, error)
}

type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string][]byte
}

func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{
		objects: make(map[string][]byte),
	}
}

func (store *MemoryObjectStore) Put(key string, value []byte) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	store.objects[key] = append([]byte(nil), value...)
	return nil
}

func (store *MemoryObjectStore) Get(key string) ([]byte, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	value, ok := store.objects[key]
	if !ok {
		return nil, ErrObjectNotFound(key)
	}

	return append([]byte(nil), value...), nil
}

func (store *MemoryObjectStore) List(prefix string) ([]string, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	keys := make([]string, 0)
	for key := range store.objects {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

func StoreItemRecord(store ObjectStore, accountID, collectionID string, record domain.VaultItemRecord) (string, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return "", err
	}

	key := ItemObjectKey(accountID, collectionID, record.Item.ID)
	if err := store.Put(key, payload); err != nil {
		return "", err
	}

	return key, nil
}

func LoadItemRecord(store ObjectStore, accountID, collectionID, itemID string) (domain.VaultItemRecord, error) {
	key := ItemObjectKey(accountID, collectionID, itemID)
	payload, err := store.Get(key)
	if err != nil {
		return domain.VaultItemRecord{}, err
	}

	return domain.ParseVaultItemRecordJSON(payload)
}

func StoreEventRecord(store ObjectStore, record domain.VaultEventRecord) (string, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return "", err
	}

	key := EventObjectKey(record.AccountID, record.CollectionID, record.EventID)
	if err := store.Put(key, payload); err != nil {
		return "", err
	}

	return key, nil
}

func LoadEventRecord(store ObjectStore, accountID, collectionID, eventID string) (domain.VaultEventRecord, error) {
	key := EventObjectKey(accountID, collectionID, eventID)
	payload, err := store.Get(key)
	if err != nil {
		return domain.VaultEventRecord{}, err
	}

	return domain.ParseVaultEventRecordJSON(payload)
}

func LoadCollectionEventRecords(store ObjectStore, accountID, collectionID string) ([]domain.VaultEventRecord, error) {
	keys, err := store.List(EventPrefix(accountID, collectionID))
	if err != nil {
		return nil, err
	}

	records := make([]domain.VaultEventRecord, 0, len(keys))
	for _, key := range keys {
		payload, err := store.Get(key)
		if err != nil {
			return nil, err
		}

		record, err := domain.ParseVaultEventRecordJSON(payload)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	return records, nil
}

type objectNotFoundError string

func (key objectNotFoundError) Error() string {
	return fmt.Sprintf("object not found: %s", string(key))
}

func ErrObjectNotFound(key string) error {
	return objectNotFoundError(key)
}
