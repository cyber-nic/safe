package storage_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/ndelorme/safe/internal/storage"
)

// localStackConfig returns an S3Config pointed at a LocalStack instance.
// Tests that call this skip automatically when SAFE_S3_ENDPOINT is not set.
func localStackConfig(t *testing.T) storage.S3Config {
	t.Helper()
	endpoint := os.Getenv("SAFE_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("SAFE_S3_ENDPOINT not set; skipping LocalStack integration test")
	}
	bucket := os.Getenv("SAFE_S3_BUCKET")
	if bucket == "" {
		bucket = "safe-dev"
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return storage.S3Config{
		Bucket:          bucket,
		Region:          region,
		Endpoint:        endpoint,
		AccessKeyID:     getEnvOrDefault("AWS_ACCESS_KEY_ID", "test"),
		SecretAccessKey: getEnvOrDefault("AWS_SECRET_ACCESS_KEY", "test"),
	}
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func newS3Store(t *testing.T) *storage.S3ObjectStoreWithCAS {
	t.Helper()
	cfg := localStackConfig(t)
	store, err := storage.NewS3ObjectStoreWithCAS(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewS3ObjectStoreWithCAS: %v", err)
	}
	return store
}

// uniqueKey returns a test-scoped key prefix that avoids cross-test collisions.
func uniqueKey(t *testing.T, suffix string) string {
	t.Helper()
	return fmt.Sprintf("test/%s/%s", t.Name(), suffix)
}

func TestS3ObjectStore_PutAndGet(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "obj")
	want := []byte("hello s3")

	if err := store.Put(key, want); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestS3ObjectStore_GetNotFound(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "missing")

	_, err := store.Get(key)
	if !storage.IsObjectNotFound(err) {
		t.Errorf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestS3ObjectStore_List(t *testing.T) {
	store := newS3Store(t)
	prefix := uniqueKey(t, "")

	keys := []string{prefix + "a", prefix + "b", prefix + "c"}
	for _, k := range keys {
		if err := store.Put(k, []byte(k)); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	got, err := store.List(prefix)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != len(keys) {
		t.Fatalf("List returned %d keys, want %d: %v", len(got), len(keys), got)
	}
	for i, k := range keys {
		if got[i] != k {
			t.Errorf("List[%d] = %q, want %q", i, got[i], k)
		}
	}
}

func TestS3ObjectStore_GetWithETag(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "etag-obj")
	value := []byte("etag content")

	if err := store.Put(key, value); err != nil {
		t.Fatalf("Put: %v", err)
	}

	data, etag, err := store.GetWithETag(key)
	if err != nil {
		t.Fatalf("GetWithETag: %v", err)
	}
	if string(data) != string(value) {
		t.Errorf("data = %q, want %q", data, value)
	}
	if etag == "" {
		t.Error("expected non-empty ETag")
	}
	if etag[0] == '"' || etag[len(etag)-1] == '"' {
		t.Errorf("ETag should not have surrounding quotes, got %q", etag)
	}
}

func TestS3ObjectStore_GetWithETag_NotFound(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "missing-etag")

	_, _, err := store.GetWithETag(key)
	if !storage.IsObjectNotFound(err) {
		t.Errorf("expected ErrObjectNotFound, got %v", err)
	}
}

func TestS3ObjectStore_PutIfMatch_CreateOnly(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "create-only")
	value := []byte("initial")

	etag, err := store.PutIfMatch(key, value, "")
	if err != nil {
		t.Fatalf("PutIfMatch create-only: %v", err)
	}
	if etag == "" {
		t.Error("expected non-empty ETag after create")
	}

	// Second create-only attempt must fail.
	_, err = store.PutIfMatch(key, []byte("duplicate"), "")
	if !errors.Is(err, storage.ErrCASConflict) {
		t.Errorf("expected ErrCASConflict on second create-only, got %v", err)
	}
}

func TestS3ObjectStore_PutIfMatch_UpdateWithCorrectETag(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "update-correct")

	// Seed the object.
	etag1, err := store.PutIfMatch(key, []byte("v1"), "")
	if err != nil {
		t.Fatalf("PutIfMatch seed: %v", err)
	}

	// Update using the correct ETag.
	etag2, err := store.PutIfMatch(key, []byte("v2"), etag1)
	if err != nil {
		t.Fatalf("PutIfMatch update: %v", err)
	}
	if etag2 == etag1 {
		t.Error("expected ETag to change after update")
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("got %q, want %q", got, "v2")
	}
}

func TestS3ObjectStore_PutIfMatch_UpdateWithStaleETag(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "stale-etag")

	etag1, err := store.PutIfMatch(key, []byte("v1"), "")
	if err != nil {
		t.Fatalf("PutIfMatch seed: %v", err)
	}

	// Advance to v2.
	_, err = store.PutIfMatch(key, []byte("v2"), etag1)
	if err != nil {
		t.Fatalf("PutIfMatch advance: %v", err)
	}

	// Attempt to overwrite with stale etag1 must fail.
	_, err = store.PutIfMatch(key, []byte("v3"), etag1)
	if !errors.Is(err, storage.ErrCASConflict) {
		t.Errorf("expected ErrCASConflict with stale ETag, got %v", err)
	}
}

func TestS3ObjectStore_PutIfMatch_ETagRoundTrip(t *testing.T) {
	store := newS3Store(t)
	key := uniqueKey(t, "etag-roundtrip")
	value := []byte("roundtrip")

	_, err := store.PutIfMatch(key, value, "")
	if err != nil {
		t.Fatalf("PutIfMatch seed: %v", err)
	}

	// ETag returned by GetWithETag must be accepted by PutIfMatch.
	_, etag, err := store.GetWithETag(key)
	if err != nil {
		t.Fatalf("GetWithETag: %v", err)
	}

	_, err = store.PutIfMatch(key, []byte("updated"), etag)
	if err != nil {
		t.Fatalf("PutIfMatch with GetWithETag ETag: %v", err)
	}
}
