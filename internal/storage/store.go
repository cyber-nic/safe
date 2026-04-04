package storage

import (
	"errors"
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

	sort.Slice(records, func(i, j int) bool {
		if records[i].Sequence == records[j].Sequence {
			return records[i].EventID < records[j].EventID
		}

		return records[i].Sequence < records[j].Sequence
	})

	return records, nil
}

func StoreCollectionHeadRecord(store ObjectStore, record domain.CollectionHeadRecord) (string, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return "", err
	}

	key := CollectionHeadKey(record.AccountID, record.CollectionID)
	if err := store.Put(key, payload); err != nil {
		return "", err
	}

	return key, nil
}

func LoadCollectionHeadRecord(store ObjectStore, accountID, collectionID string) (domain.CollectionHeadRecord, error) {
	payload, err := store.Get(CollectionHeadKey(accountID, collectionID))
	if err != nil {
		return domain.CollectionHeadRecord{}, err
	}

	return domain.ParseCollectionHeadRecordJSON(payload)
}

func StoreAccountConfigRecord(store ObjectStore, record domain.AccountConfigRecord) (string, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return "", err
	}

	key := AccountConfigKey(record.AccountID)
	if err := store.Put(key, payload); err != nil {
		return "", err
	}

	return key, nil
}

func LoadAccountConfigRecord(store ObjectStore, accountID string) (domain.AccountConfigRecord, error) {
	payload, err := store.Get(AccountConfigKey(accountID))
	if err != nil {
		return domain.AccountConfigRecord{}, err
	}

	return domain.ParseAccountConfigRecordJSON(payload)
}

func StoreLocalUnlockRecord(store ObjectStore, record domain.LocalUnlockRecord) (string, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return "", err
	}

	key := LocalUnlockKey(record.AccountID)
	if err := store.Put(key, payload); err != nil {
		return "", err
	}

	return key, nil
}

func LoadLocalUnlockRecord(store ObjectStore, accountID string) (domain.LocalUnlockRecord, error) {
	payload, err := store.Get(LocalUnlockKey(accountID))
	if err != nil {
		return domain.LocalUnlockRecord{}, err
	}

	return domain.ParseLocalUnlockRecordJSON(payload)
}

func StoreSecretMaterialBytes(store ObjectStore, accountID, collectionID, secretRef string, secret []byte) (string, error) {
	key := SecretMaterialKey(accountID, collectionID, secretRef)
	if err := store.Put(key, secret); err != nil {
		return "", err
	}

	return key, nil
}

func LoadSecretMaterialBytes(store ObjectStore, accountID, collectionID, secretRef string) ([]byte, error) {
	return store.Get(SecretMaterialKey(accountID, collectionID, secretRef))
}

func StoreSecretMaterial(store ObjectStore, accountID, collectionID, secretRef, secret string) (string, error) {
	return StoreSecretMaterialBytes(store, accountID, collectionID, secretRef, []byte(secret))
}

func LoadSecretMaterial(store ObjectStore, accountID, collectionID, secretRef string) (string, error) {
	payload, err := LoadSecretMaterialBytes(store, accountID, collectionID, secretRef)
	if err != nil {
		return "", err
	}

	return string(payload), nil
}

// VaultMutation describes the records that must become durable together as
// one logical vault mutation. HeadRecord and EventRecord are always required.
// SecretRef and SecretMaterial are optional (both must be non-empty to write a
// secret). ItemRecord is optional (nil means no item write).
type VaultMutation struct {
	AccountID      string
	CollectionID   string
	SecretRef      string
	SecretMaterial string
	ItemRecord     *domain.VaultItemRecord
	EventRecord    domain.VaultEventRecord
	HeadRecord     domain.CollectionHeadRecord
}

// CommitVaultMutation writes all records in m in dependency order: optional
// secret material first, then optional item record, then event record, and
// finally the collection head. Writing the head last ensures that a new head
// is never observable before its supporting records are durable, even if the
// process crashes mid-commit.
func CommitVaultMutation(store ObjectStore, m VaultMutation) error {
	if m.SecretRef != "" && m.SecretMaterial != "" {
		if _, err := StoreSecretMaterial(store, m.AccountID, m.CollectionID, m.SecretRef, m.SecretMaterial); err != nil {
			return err
		}
	}
	if m.ItemRecord != nil {
		if _, err := StoreItemRecord(store, m.AccountID, m.CollectionID, *m.ItemRecord); err != nil {
			return err
		}
	}
	if _, err := StoreEventRecord(store, m.EventRecord); err != nil {
		return err
	}
	if _, err := StoreCollectionHeadRecord(store, m.HeadRecord); err != nil {
		return err
	}
	return nil
}

type objectNotFoundError string

func (key objectNotFoundError) Error() string {
	return fmt.Sprintf("object not found: %s", string(key))
}

func ErrObjectNotFound(key string) error {
	return objectNotFoundError(key)
}

// IsObjectNotFound reports whether err is an object-not-found error returned
// by Get or any Load helper in this package.
func IsObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	var notFound objectNotFoundError
	return errors.As(err, &notFound)
}
