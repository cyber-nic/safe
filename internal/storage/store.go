package storage

import (
	"fmt"
	"sync"

	"github.com/ndelorme/safe/internal/domain"
)

type ObjectStore interface {
	Put(key string, value []byte) error
	Get(key string) ([]byte, error)
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

type objectNotFoundError string

func (key objectNotFoundError) Error() string {
	return fmt.Sprintf("object not found: %s", string(key))
}

func ErrObjectNotFound(key string) error {
	return objectNotFoundError(key)
}
