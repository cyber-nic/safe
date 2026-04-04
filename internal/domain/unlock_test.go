package domain

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestLocalUnlockRecordCanonicalSerialization(t *testing.T) {
	record := LocalUnlockRecord{
		SchemaVersion: 1,
		AccountID:     "acct-dev-001",
		KDF: LocalUnlockKDF{
			Name:        "argon2id",
			Salt:        base64.RawURLEncoding.EncodeToString([]byte("0123456789abcdef")),
			MemoryKiB:   65536,
			TimeCost:    3,
			Parallelism: 4,
			KeyBytes:    32,
		},
		WrappedKey: LocalUnlockWrappedKey{
			Algorithm:  "aes-256-gcm",
			Nonce:      base64.RawURLEncoding.EncodeToString([]byte("0123456789ab")),
			Ciphertext: base64.RawURLEncoding.EncodeToString([]byte("ciphertext")),
		},
	}

	canonical, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonicalize local unlock record: %v", err)
	}

	expected, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal expected local unlock record: %v", err)
	}

	if string(canonical) != string(expected) {
		t.Fatalf("local unlock canonical mismatch\nexpected: %s\ngot: %s", string(expected), string(canonical))
	}

	parsed, err := ParseLocalUnlockRecordJSON(canonical)
	if err != nil {
		t.Fatalf("parse local unlock record: %v", err)
	}

	if parsed.AccountID != record.AccountID || parsed.KDF.Name != "argon2id" {
		t.Fatalf("unexpected parsed local unlock record: %+v", parsed)
	}
}

func TestLocalUnlockRecordValidateRejectsInvalidFields(t *testing.T) {
	record := LocalUnlockRecord{
		SchemaVersion: 1,
		AccountID:     "acct-dev-001",
		KDF: LocalUnlockKDF{
			Name:        "argon2id",
			Salt:        "not-base64***",
			MemoryKiB:   65536,
			TimeCost:    3,
			Parallelism: 4,
			KeyBytes:    32,
		},
		WrappedKey: LocalUnlockWrappedKey{
			Algorithm:  "aes-256-gcm",
			Nonce:      base64.RawURLEncoding.EncodeToString([]byte("0123456789ab")),
			Ciphertext: base64.RawURLEncoding.EncodeToString([]byte("ciphertext")),
		},
	}

	if err := record.Validate(); err == nil {
		t.Fatal("expected invalid local unlock record error")
	}
}
