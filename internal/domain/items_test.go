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
	BodyPreview   string   `json:"bodyPreview,omitempty"`
	Service       string   `json:"service,omitempty"`
	Host          string   `json:"host,omitempty"`
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

func TestVaultEventRecordFixtureCanonicalSerialization(t *testing.T) {
	path := filepath.Join("..", "..", "packages", "test-vectors", "src", "event-records.json")

	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read event fixture: %v", err)
	}

	var rawRecords []json.RawMessage
	if err := json.Unmarshal(payload, &rawRecords); err != nil {
		t.Fatalf("unmarshal event fixture: %v", err)
	}

	if len(rawRecords) != len(StarterVaultEventRecords()) {
		t.Fatalf("expected %d starter events, got %d", len(StarterVaultEventRecords()), len(rawRecords))
	}

	for index, rawRecord := range rawRecords {
		record, err := ParseVaultEventRecordJSON(rawRecord)
		if err != nil {
			t.Fatalf("parse event %d: %v", index, err)
		}

		canonical, err := record.CanonicalJSON()
		if err != nil {
			t.Fatalf("canonicalize event %d: %v", index, err)
		}

		var compact bytes.Buffer
		if err := json.Compact(&compact, rawRecord); err != nil {
			t.Fatalf("compact event %d: %v", index, err)
		}

		if compact.String() != string(canonical) {
			t.Fatalf("event %d canonical mismatch\nexpected: %s\ngot: %s", index, compact.String(), string(canonical))
		}
	}
}

func TestDeleteEventRecordCanonicalSerialization(t *testing.T) {
	path := filepath.Join("..", "..", "packages", "test-vectors", "src", "delete-event-record.json")

	rawRecord, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read delete event fixture: %v", err)
	}

	record := VaultEventRecord{
		SchemaVersion: 1,
		EventID:       "evt-login-gmail-primary-delete-v3",
		AccountID:     "acct-dev-001",
		DeviceID:      "dev-web-001",
		CollectionID:  "vault-personal",
		Sequence:      3,
		OccurredAt:    "2026-03-31T10:04:00Z",
		Action:        VaultEventActionDeleteItem,
		ItemID:        "login-gmail-primary",
	}

	canonical, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonicalize delete event: %v", err)
	}

	var compact bytes.Buffer
	if err := json.Compact(&compact, rawRecord); err != nil {
		t.Fatalf("compact delete event fixture: %v", err)
	}

	if string(canonical) != compact.String() {
		t.Fatalf("delete event canonical mismatch\nexpected: %s\ngot: %s", compact.String(), string(canonical))
	}

	parsed, err := ParseVaultEventRecordJSON(canonical)
	if err != nil {
		t.Fatalf("parse delete event: %v", err)
	}

	if parsed.Action != VaultEventActionDeleteItem || parsed.ItemID != "login-gmail-primary" {
		t.Fatalf("unexpected parsed delete event: %+v", parsed)
	}
}

func TestCollectionHeadRecordCanonicalSerialization(t *testing.T) {
	record := StarterCollectionHeadRecord()

	canonical, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonicalize collection head: %v", err)
	}

	expected := `{"schemaVersion":1,"accountId":"acct-dev-001","collectionId":"vault-personal","latestEventId":"evt-totp-gmail-primary-v1","latestSeq":2}`
	if string(canonical) != expected {
		t.Fatalf("collection head canonical mismatch\nexpected: %s\ngot: %s", expected, string(canonical))
	}

	parsed, err := ParseCollectionHeadRecordJSON(canonical)
	if err != nil {
		t.Fatalf("parse collection head: %v", err)
	}

	if parsed.LatestSeq != 2 || parsed.LatestEventID != "evt-totp-gmail-primary-v1" {
		t.Fatalf("unexpected parsed collection head: %+v", parsed)
	}
}

func TestAccountConfigRecordCanonicalSerialization(t *testing.T) {
	record := StarterAccountConfigRecord()

	canonical, err := record.CanonicalJSON()
	if err != nil {
		t.Fatalf("canonicalize account config: %v", err)
	}

	expected := `{"schemaVersion":1,"accountId":"acct-dev-001","defaultCollectionId":"vault-personal","collectionIds":["vault-personal"],"deviceIds":["dev-web-001"]}`
	if string(canonical) != expected {
		t.Fatalf("account config canonical mismatch\nexpected: %s\ngot: %s", expected, string(canonical))
	}

	parsed, err := ParseAccountConfigRecordJSON(canonical)
	if err != nil {
		t.Fatalf("parse account config: %v", err)
	}

	if parsed.DefaultCollectionID != "vault-personal" || len(parsed.CollectionIDs) != 1 {
		t.Fatalf("unexpected parsed account config: %+v", parsed)
	}
}

func TestVaultItemRecordValidateSupportedKinds(t *testing.T) {
	tests := []struct {
		name   string
		record VaultItemRecord
	}{
		{
			name: "note",
			record: VaultItemRecord{
				SchemaVersion: 1,
				Item: VaultItem{
					ID:          "note-server-bootstrap",
					Kind:        VaultItemKindNote,
					Title:       "Server Bootstrap",
					Tags:        []string{"infra"},
					BodyPreview: "Bootstrap checklist",
				},
			},
		},
		{
			name: "apiKey",
			record: VaultItemRecord{
				SchemaVersion: 1,
				Item: VaultItem{
					ID:      "apikey-stripe-primary",
					Kind:    VaultItemKindAPIKey,
					Title:   "Stripe Primary",
					Tags:    []string{"payments"},
					Service: "Stripe",
				},
			},
		},
		{
			name: "sshKey",
			record: VaultItemRecord{
				SchemaVersion: 1,
				Item: VaultItem{
					ID:       "ssh-prod-root",
					Kind:     VaultItemKindSSHKey,
					Title:    "Prod Root",
					Tags:     []string{"infra"},
					Username: "root",
					Host:     "prod-01.example.com",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.record.Validate(); err != nil {
				t.Fatalf("validate %s: %v", test.name, err)
			}

			canonical, err := test.record.CanonicalJSON()
			if err != nil {
				t.Fatalf("canonical %s: %v", test.name, err)
			}

			parsed, err := ParseVaultItemRecordJSON(canonical)
			if err != nil {
				t.Fatalf("parse %s: %v", test.name, err)
			}

			if parsed.Item.Kind != test.record.Item.Kind || parsed.Item.ID != test.record.Item.ID {
				t.Fatalf("unexpected parsed %s record: %+v", test.name, parsed)
			}
		})
	}
}

func TestVaultItemSummarySupportedKinds(t *testing.T) {
	tests := []struct {
		item        VaultItem
		description string
	}{
		{
			item: VaultItem{
				ID:          "note-server-bootstrap",
				Kind:        VaultItemKindNote,
				Title:       "Server Bootstrap",
				Tags:        []string{"infra"},
				BodyPreview: "Bootstrap checklist",
			},
			description: "Secure note",
		},
		{
			item: VaultItem{
				ID:      "apikey-stripe-primary",
				Kind:    VaultItemKindAPIKey,
				Title:   "Stripe Primary",
				Tags:    []string{"payments"},
				Service: "Stripe",
			},
			description: "API key for Stripe",
		},
		{
			item: VaultItem{
				ID:       "ssh-prod-root",
				Kind:     VaultItemKindSSHKey,
				Title:    "Prod Root",
				Tags:     []string{"infra"},
				Username: "root",
				Host:     "prod-01.example.com",
			},
			description: "SSH key for root@prod-01.example.com",
		},
	}

	for _, test := range tests {
		summary := test.item.Summary()
		if summary.Description != test.description {
			t.Fatalf("unexpected summary for %s: %+v", test.item.Kind, summary)
		}
	}
}
