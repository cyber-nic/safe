package sync

import (
	"testing"

	"github.com/ndelorme/safe/internal/domain"
)

func TestReplayCollectionBuildsLatestState(t *testing.T) {
	events := domain.StarterVaultEventRecords()

	projection, err := ReplayCollection(events)
	if err != nil {
		t.Fatalf("replay collection: %v", err)
	}

	if projection.AccountID != "acct-dev-001" {
		t.Fatalf("unexpected account ID: %s", projection.AccountID)
	}

	if projection.CollectionID != "vault-personal" {
		t.Fatalf("unexpected collection ID: %s", projection.CollectionID)
	}

	if projection.LatestSeq != 2 {
		t.Fatalf("unexpected latest sequence: %d", projection.LatestSeq)
	}

	if len(projection.Items) != 2 {
		t.Fatalf("expected 2 projected items, got %d", len(projection.Items))
	}
}

func TestReplayCollectionSortsInput(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	events[0], events[1] = events[1], events[0]

	projection, err := ReplayCollection(events)
	if err != nil {
		t.Fatalf("replay collection: %v", err)
	}

	if projection.LatestSeq != 2 {
		t.Fatalf("unexpected latest sequence: %d", projection.LatestSeq)
	}

	if _, ok := projection.Items["login-gmail-primary"]; !ok {
		t.Fatal("expected login item in projection")
	}
}

func TestReplayCollectionRejectsSequenceGap(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	events[1].Sequence = 3

	if _, err := ReplayCollection(events); err == nil {
		t.Fatal("expected sequence gap error")
	}
}

func TestReplayCollectionRejectsMixedCollection(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	events[1].CollectionID = "vault-shared"

	if _, err := ReplayCollection(events); err == nil {
		t.Fatal("expected mixed collection error")
	}
}

func TestReplayCollectionDeletesItems(t *testing.T) {
	events := append([]domain.VaultEventRecord(nil), domain.StarterVaultEventRecords()...)
	events = append(events, domain.VaultEventRecord{
		SchemaVersion: 1,
		EventID:       "evt-login-gmail-primary-delete-v3",
		AccountID:     "acct-dev-001",
		DeviceID:      "dev-web-001",
		CollectionID:  "vault-personal",
		Sequence:      3,
		OccurredAt:    "2026-03-31T10:04:00Z",
		Action:        domain.VaultEventActionDeleteItem,
		ItemID:        "login-gmail-primary",
	})

	projection, err := ReplayCollection(events)
	if err != nil {
		t.Fatalf("replay collection: %v", err)
	}

	if projection.LatestSeq != 3 {
		t.Fatalf("unexpected latest sequence: %d", projection.LatestSeq)
	}

	if _, ok := projection.Items["login-gmail-primary"]; ok {
		t.Fatal("expected login item to be deleted from projection")
	}
}

func TestReplayCollectionAgainstHead(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	head := domain.StarterCollectionHeadRecord()

	projection, err := ReplayCollectionAgainstHead(events, head)
	if err != nil {
		t.Fatalf("replay collection against head: %v", err)
	}

	if projection.LatestSeq != head.LatestSeq || projection.LatestEventID != head.LatestEventID {
		t.Fatalf("unexpected projection/head alignment: %+v %+v", projection, head)
	}
}

func TestReplayCollectionAgainstHeadRejectsLatestSeqMismatch(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	head := domain.StarterCollectionHeadRecord()
	head.LatestSeq = 3

	if _, err := ReplayCollectionAgainstHead(events, head); err == nil {
		t.Fatal("expected latest sequence mismatch error")
	}
}

func TestReplayCollectionAgainstHeadRejectsLatestEventMismatch(t *testing.T) {
	events := domain.StarterVaultEventRecords()
	head := domain.StarterCollectionHeadRecord()
	head.LatestEventID = "evt-mismatch"

	if _, err := ReplayCollectionAgainstHead(events, head); err == nil {
		t.Fatal("expected latest event mismatch error")
	}
}

func TestEnsureMonotonicHeadRejectsRollback(t *testing.T) {
	trusted := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     "acct-dev-001",
		CollectionID:  "vault-personal",
		LatestEventID: "evt-login-github-primary-v3",
		LatestSeq:     3,
	}
	candidate := domain.StarterCollectionHeadRecord()

	if err := EnsureMonotonicHead(trusted, candidate); err == nil {
		t.Fatal("expected stale head rejection")
	}
}

func TestEnsureMonotonicHeadRejectsEqualSeqDifferentEvent(t *testing.T) {
	trusted := domain.StarterCollectionHeadRecord()
	candidate := trusted
	candidate.LatestEventID = "evt-different"

	if err := EnsureMonotonicHead(trusted, candidate); err == nil {
		t.Fatal("expected equal-sequence different-event rejection")
	}
}

func TestEnsureMonotonicHeadAcceptsAdvance(t *testing.T) {
	trusted := domain.StarterCollectionHeadRecord()
	candidate := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     trusted.AccountID,
		CollectionID:  trusted.CollectionID,
		LatestEventID: "evt-login-github-primary-v3",
		LatestSeq:     3,
	}

	if err := EnsureMonotonicHead(trusted, candidate); err != nil {
		t.Fatalf("expected monotonic head acceptance, got %v", err)
	}
}

func TestBuildPutItemMutation(t *testing.T) {
	head := domain.StarterCollectionHeadRecord()
	itemRecord := domain.VaultItemRecord{
		SchemaVersion: 1,
		Item: domain.VaultItem{
			ID:       "login-github-primary",
			Kind:     domain.VaultItemKindLogin,
			Title:    "GitHub",
			Tags:     []string{"dev"},
			Username: "alice",
			URLs:     []string{"https://github.com/login"},
		},
	}

	event, newHead, err := BuildPutItemMutation(head, "dev-web-001", itemRecord, "2026-03-31T10:02:00Z")
	if err != nil {
		t.Fatalf("build mutation: %v", err)
	}

	if event.Sequence != 3 {
		t.Fatalf("expected sequence 3, got %d", event.Sequence)
	}

	if event.EventID != "evt-login-github-primary-v3" {
		t.Fatalf("unexpected event ID: %s", event.EventID)
	}

	if newHead.LatestSeq != 3 || newHead.LatestEventID != event.EventID {
		t.Fatalf("unexpected head: %+v", newHead)
	}
}

func TestBuildDeleteItemMutation(t *testing.T) {
	head := domain.StarterCollectionHeadRecord()

	event, newHead, err := BuildDeleteItemMutation(head, "dev-web-001", "login-gmail-primary", "2026-03-31T10:04:00Z")
	if err != nil {
		t.Fatalf("build delete mutation: %v", err)
	}

	if event.Sequence != 3 {
		t.Fatalf("expected sequence 3, got %d", event.Sequence)
	}

	if event.EventID != "evt-login-gmail-primary-delete-v3" {
		t.Fatalf("unexpected event ID: %s", event.EventID)
	}

	if event.Action != domain.VaultEventActionDeleteItem || event.ItemID != "login-gmail-primary" {
		t.Fatalf("unexpected event: %+v", event)
	}

	if newHead.LatestSeq != 3 || newHead.LatestEventID != event.EventID {
		t.Fatalf("unexpected head: %+v", newHead)
	}
}
