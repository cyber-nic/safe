package auth

import (
	"errors"
	"testing"
	"time"
)

func TestIssueAndVerifyAccountCapability(t *testing.T) {
	signer := newTestSigner(t, time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC))

	signed, err := signer.IssueAccountCapability(AccountAccessRequest{
		AccountID:      "acct-dev-001",
		DeviceID:       "dev-web-001",
		Bucket:         "safe-dev",
		AllowedActions: []string{ActionPut, ActionGet},
		TTL:            5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("issue capability: %v", err)
	}

	if signed.Capability.Prefix != "accounts/acct-dev-001/" {
		t.Fatalf("unexpected prefix: %s", signed.Capability.Prefix)
	}
	if len(signed.Capability.AllowedActions) != 2 || signed.Capability.AllowedActions[0] != ActionGet || signed.Capability.AllowedActions[1] != ActionPut {
		t.Fatalf("unexpected actions: %#v", signed.Capability.AllowedActions)
	}

	capability, err := signer.VerifyAccountAccess(signed.Token, AccessCheck{
		Bucket: "safe-dev",
		Action: ActionPut,
		Key:    "accounts/acct-dev-001/collections/vault/items/item-1.json",
		Now:    time.Date(2026, 4, 6, 8, 2, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("verify capability: %v", err)
	}
	if capability.DeviceID != "dev-web-001" {
		t.Fatalf("unexpected device: %s", capability.DeviceID)
	}
}

func TestVerifyAccountAccessRejectsPrefixOverreach(t *testing.T) {
	signer := newTestSigner(t, time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC))
	signed := mustIssueCapability(t, signer)

	_, err := signer.VerifyAccountAccess(signed.Token, AccessCheck{
		Bucket: "safe-dev",
		Action: ActionGet,
		Key:    "accounts/acct-other-001/collections/vault/items/item-1.json",
		Now:    time.Date(2026, 4, 6, 8, 1, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected prefix overreach rejection")
	}
}

func TestVerifyAccountAccessRejectsMethodEscalation(t *testing.T) {
	signer := newTestSigner(t, time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC))
	signed, err := signer.IssueAccountCapability(AccountAccessRequest{
		AccountID:      "acct-dev-001",
		DeviceID:       "dev-web-001",
		Bucket:         "safe-dev",
		AllowedActions: []string{ActionGet},
		TTL:            5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("issue capability: %v", err)
	}

	_, err = signer.VerifyAccountAccess(signed.Token, AccessCheck{
		Bucket: "safe-dev",
		Action: ActionPut,
		Key:    "accounts/acct-dev-001/collections/vault/items/item-1.json",
		Now:    time.Date(2026, 4, 6, 8, 1, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected method escalation rejection")
	}
}

func TestVerifyAccountAccessRejectsExpiredCapability(t *testing.T) {
	signer := newTestSigner(t, time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC))
	signed := mustIssueCapability(t, signer)

	_, err := signer.VerifyAccountAccess(signed.Token, AccessCheck{
		Bucket: "safe-dev",
		Action: ActionGet,
		Key:    "accounts/acct-dev-001/collections/vault/items/item-1.json",
		Now:    time.Date(2026, 4, 6, 8, 6, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrExpiredCapability()) {
		t.Fatalf("expected expired capability, got %v", err)
	}
}

func TestVerifyAccountAccessRejectsListByDefault(t *testing.T) {
	signer := newTestSigner(t, time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC))
	signed := mustIssueCapability(t, signer)

	_, err := signer.VerifyAccountAccess(signed.Token, AccessCheck{
		Bucket: "safe-dev",
		Action: ActionList,
		Key:    "accounts/acct-dev-001/collections/vault/events",
		Now:    time.Date(2026, 4, 6, 8, 1, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected list rejection")
	}
}

func TestValidateActiveDevice(t *testing.T) {
	err := ValidateActiveDevice("acct-dev-001", "dev-web-001", func(accountID, deviceID string) (bool, error) {
		if accountID != "acct-dev-001" || deviceID != "dev-web-001" {
			t.Fatalf("unexpected lookup args: %s %s", accountID, deviceID)
		}
		return true, nil
	})
	if err != nil {
		t.Fatalf("validate active device: %v", err)
	}

	err = ValidateActiveDevice("acct-dev-001", "dev-revoked-001", func(accountID, deviceID string) (bool, error) {
		return false, nil
	})
	if err == nil {
		t.Fatal("expected revoked device rejection")
	}
}

func newTestSigner(t *testing.T, now time.Time) *CapabilitySigner {
	t.Helper()

	signer, err := NewCapabilitySigner("test-key", []byte("0123456789abcdef0123456789abcdef"))
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	signer.now = func() time.Time { return now }
	return signer
}

func mustIssueCapability(t *testing.T, signer *CapabilitySigner) SignedCapability {
	t.Helper()

	signed, err := signer.IssueAccountCapability(AccountAccessRequest{
		AccountID:      "acct-dev-001",
		DeviceID:       "dev-web-001",
		Bucket:         "safe-dev",
		AllowedActions: []string{ActionGet, ActionPut},
		TTL:            5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("issue capability: %v", err)
	}

	return signed
}
