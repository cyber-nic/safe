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
