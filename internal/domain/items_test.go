package domain

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type fixtureVaultItem struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Tags          []string `json:"tags"`
	Username      string   `json:"username,omitempty"`
	URLs          []string `json:"urls,omitempty"`
	Issuer        string   `json:"issuer,omitempty"`
	AccountName   string   `json:"accountName,omitempty"`
	Digits        int      `json:"digits,omitempty"`
	PeriodSeconds int      `json:"periodSeconds,omitempty"`
	Algorithm     string   `json:"algorithm,omitempty"`
	SecretRef     string   `json:"secretRef,omitempty"`
}

func TestVaultItemFixtureShape(t *testing.T) {
	path := filepath.Join("..", "..", "packages", "test-vectors", "src", "vault-items.json")

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var items []fixtureVaultItem
	if err := json.Unmarshal(payload, &items); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 starter items, got %d", len(items))
	}

	if items[0].Kind != string(VaultItemKindLogin) {
		t.Fatalf("expected first item kind %q, got %q", VaultItemKindLogin, items[0].Kind)
	}

	if items[1].Kind != string(VaultItemKindTOTP) {
		t.Fatalf("expected second item kind %q, got %q", VaultItemKindTOTP, items[1].Kind)
	}

	if items[1].Digits != 6 {
		t.Fatalf("expected TOTP digits 6, got %d", items[1].Digits)
	}

	if items[1].PeriodSeconds != 30 {
		t.Fatalf("expected TOTP period 30, got %d", items[1].PeriodSeconds)
	}

	if items[1].Algorithm != "SHA1" {
		t.Fatalf("expected TOTP algorithm SHA1, got %q", items[1].Algorithm)
	}
}

func TestVaultItemRecordFixtureCanonicalSerialization(t *testing.T) {
	path := filepath.Join("..", "..", "packages", "test-vectors", "src", "vault-item-records.json")

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read record fixture: %v", err)
	}

	var rawRecords []json.RawMessage
	if err := json.Unmarshal(payload, &rawRecords); err != nil {
		t.Fatalf("unmarshal record fixture: %v", err)
	}

	if len(rawRecords) != len(StarterVaultItemRecords()) {
		t.Fatalf("expected %d starter records, got %d", len(StarterVaultItemRecords()), len(rawRecords))
	}

	for index, rawRecord := range rawRecords {
		record, err := ParseVaultItemRecordJSON(rawRecord)
		if err != nil {
			t.Fatalf("parse record %d: %v", index, err)
		}

		canonical, err := record.CanonicalJSON()
		if err != nil {
			t.Fatalf("canonicalize record %d: %v", index, err)
		}

		var compact bytes.Buffer
		if err := json.Compact(&compact, rawRecord); err != nil {
			t.Fatalf("compact record %d: %v", index, err)
		}

		if compact.String() != string(canonical) {
			t.Fatalf("record %d canonical mismatch\nexpected: %s\ngot: %s", index, compact.String(), string(canonical))
		}
	}
}
