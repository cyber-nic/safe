package storage

import "testing"

func TestEventObjectKey(t *testing.T) {
	got := EventObjectKey("acct-dev-001", "vault-personal", "evt-login-gmail-primary-v1")
	want := "accounts/acct-dev-001/collections/vault-personal/events/evt-login-gmail-primary-v1.json"

	if got != want {
		t.Fatalf("unexpected event object key\nwant: %s\ngot:  %s", want, got)
	}
}

func TestItemObjectKey(t *testing.T) {
	got := ItemObjectKey("acct-dev-001", "vault-personal", "totp-gmail-primary")
	want := "accounts/acct-dev-001/collections/vault-personal/items/totp-gmail-primary.json"

	if got != want {
		t.Fatalf("unexpected item object key\nwant: %s\ngot:  %s", want, got)
	}
}
