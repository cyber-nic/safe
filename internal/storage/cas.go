package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ErrCASConflict is returned by PutIfMatch when the stored ETag does not match
// the expectedETag, indicating a concurrent write has advanced the object.
var ErrCASConflict = errors.New("object store CAS conflict: ETag mismatch")

// ObjectStoreWithCAS extends ObjectStore with compare-and-swap support for
// head-pointer advancement. Immutable objects (events, items) use the base Put.
// Only mutable head records require CAS.
type ObjectStoreWithCAS interface {
	ObjectStore

	// GetWithETag returns the current value and its ETag. The ETag is opaque to
	// callers; it must be passed back to PutIfMatch to prove the caller observed
	// the same version being replaced. Returns ErrObjectNotFound if absent.
	GetWithETag(key string) (value []byte, etag string, err error)

	// PutIfMatch writes value at key only if the current ETag equals expectedETag.
	// On success it returns the new ETag for the written value.
	// On conflict it returns ("", ErrCASConflict).
	// Pass expectedETag="" to signal "object must not yet exist" (create-only).
	PutIfMatch(key string, value []byte, expectedETag string) (newETag string, err error)
}

// ContentETag returns a stable ETag for value: hex-encoded SHA-256 of the bytes.
// This is used by MemoryObjectStoreWithCAS and can be used in tests to predict ETags.
func ContentETag(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

// MemoryObjectStoreWithCAS is an in-memory ObjectStoreWithCAS suitable for
// tests that simulate two runtimes sharing a single object store.
type MemoryObjectStoreWithCAS struct {
	mu      sync.Mutex
	objects map[string]casEntry
}

type casEntry struct {
	value []byte
	etag  string
}

// NewMemoryObjectStoreWithCAS returns an empty in-memory CAS-capable store.
func NewMemoryObjectStoreWithCAS() *MemoryObjectStoreWithCAS {
	return &MemoryObjectStoreWithCAS{
		objects: make(map[string]casEntry),
	}
}

func (s *MemoryObjectStoreWithCAS) Put(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = casEntry{value: append([]byte(nil), value...), etag: ContentETag(value)}
	return nil
}

func (s *MemoryObjectStoreWithCAS) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.objects[key]
	if !ok {
		return nil, ErrObjectNotFound(key)
	}
	return append([]byte(nil), entry.value...), nil
}

func (s *MemoryObjectStoreWithCAS) List(prefix string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0)
	for k := range s.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *MemoryObjectStoreWithCAS) GetWithETag(key string) ([]byte, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.objects[key]
	if !ok {
		return nil, "", ErrObjectNotFound(key)
	}
	return append([]byte(nil), entry.value...), entry.etag, nil
}

func (s *MemoryObjectStoreWithCAS) PutIfMatch(key string, value []byte, expectedETag string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.objects[key]
	currentETag := ""
	if exists {
		currentETag = entry.etag
	}

	if currentETag != expectedETag {
		return "", fmt.Errorf("%w: key %s expected %q got %q", ErrCASConflict, key, expectedETag, currentETag)
	}

	newETag := ContentETag(value)
	s.objects[key] = casEntry{value: append([]byte(nil), value...), etag: newETag}
	return newETag, nil
}
