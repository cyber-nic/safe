package storage

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileObjectStore is a file-backed ObjectStore. Each object key is mapped to a
// file path under rootDir by treating forward-slash separators in keys as
// directory separators. This matches the key layout produced by keys.go.
//
// Writes are performed atomically via a temp file and rename so that a process
// crash during Put cannot leave a partially written file behind.
//
// FileObjectStore satisfies the ObjectStore interface and can be used anywhere
// MemoryObjectStore is used today.
type FileObjectStore struct {
	rootDir string
}

// NewFileObjectStore returns a FileObjectStore rooted at rootDir. The directory
// is created if it does not already exist.
func NewFileObjectStore(rootDir string) (*FileObjectStore, error) {
	if err := os.MkdirAll(rootDir, 0o700); err != nil {
		return nil, err
	}
	return &FileObjectStore{rootDir: rootDir}, nil
}

func (s *FileObjectStore) filePath(key string) string {
	return filepath.Join(s.rootDir, filepath.FromSlash(key))
}

// Put writes value to the file mapped to key. Parent directories are created as
// needed. The write is atomic: a temp file is written first and then renamed
// into place.
func (s *FileObjectStore) Put(key string, value []byte) error {
	p := s.filePath(key)

	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}

	// Write to a sibling temp file then rename for atomicity.
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, value, 0o600); err != nil {
		return err
	}

	return os.Rename(tmp, p)
}

// Get reads the value stored under key. Returns ErrObjectNotFound if the key
// does not exist.
func (s *FileObjectStore) Get(key string) ([]byte, error) {
	p := s.filePath(key)

	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrObjectNotFound(key)
	}
	if err != nil {
		return nil, err
	}

	return data, nil
}

// List returns all keys whose path begins with prefix, sorted lexicographically.
// Temp files left by interrupted writes (suffix ".tmp") are excluded.
func (s *FileObjectStore) List(prefix string) ([]string, error) {
	var keys []string

	err := filepath.WalkDir(s.rootDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Exclude in-flight temp files.
		if strings.HasSuffix(p, ".tmp") {
			return nil
		}

		rel, err := filepath.Rel(s.rootDir, p)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)

		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(keys)
	return keys, nil
}
