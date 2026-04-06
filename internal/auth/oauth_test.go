package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOAuthVerifierVerifyBearerToken(t *testing.T) {
	verifier := newTestOAuthVerifier(t)
	token := newTestOAuthToken(t, OAuthIdentity{
		Subject:   "user-test-001",
		AccountID: "acct-test-001",
		Env:       "test",
		IssuedAt:  time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC).Unix(),
		ExpiresAt: time.Date(2026, 4, 6, 8, 5, 0, 0, time.UTC).Unix(),
	})

	identity, err := verifier.VerifyBearerToken("Bearer " + token)
	if err != nil {
		t.Fatalf("verify bearer token: %v", err)
	}
	if identity.AccountID != "acct-test-001" || identity.Subject != "user-test-001" || identity.Env != "test" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
}

func TestOAuthVerifierRejectsMissingBearerToken(t *testing.T) {
	verifier := newTestOAuthVerifier(t)

	_, err := verifier.VerifyBearerToken("")
	if !errors.Is(err, ErrMissingOAuthToken()) {
		t.Fatalf("expected missing oauth token error, got %v", err)
	}
}

func TestOAuthVerifierRejectsExpiredToken(t *testing.T) {
	verifier := newTestOAuthVerifier(t)
	token := newTestOAuthToken(t, OAuthIdentity{
		Subject:   "user-test-001",
		AccountID: "acct-test-001",
		Env:       "test",
		IssuedAt:  time.Date(2026, 4, 6, 7, 0, 0, 0, time.UTC).Unix(),
		ExpiresAt: time.Date(2026, 4, 6, 7, 5, 0, 0, time.UTC).Unix(),
	})

	_, err := verifier.VerifyAccessToken(token)
	if !errors.Is(err, ErrExpiredOAuthToken()) {
		t.Fatalf("expected expired oauth token error, got %v", err)
	}
}

func TestOAuthVerifierRejectsWrongAudience(t *testing.T) {
	verifier := newTestOAuthVerifier(t)
	token, err := IssueTestOAuthToken(
		"safe-test-issuer",
		"other-audience",
		[]byte("0123456789abcdef0123456789abcdef"),
		OAuthIdentity{
			Subject:   "user-test-001",
			AccountID: "acct-test-001",
			Env:       "test",
			IssuedAt:  time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC).Unix(),
			ExpiresAt: time.Date(2026, 4, 6, 8, 5, 0, 0, time.UTC).Unix(),
		},
	)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}

	_, err = verifier.VerifyAccessToken(token)
	if !errors.Is(err, ErrInvalidOAuthToken("audience")) {
		t.Fatalf("expected invalid audience error, got %v", err)
	}
}

func TestOAuthVerifierRejectsBadAuthorizationHeader(t *testing.T) {
	verifier := newTestOAuthVerifier(t)

	_, err := verifier.VerifyBearerToken("Token nope")
	if !errors.Is(err, ErrInvalidOAuthToken("authorization")) {
		t.Fatalf("expected invalid authorization error, got %v", err)
	}
}

func TestIssueTestOAuthTokenSetsClaims(t *testing.T) {
	token := newTestOAuthToken(t, OAuthIdentity{
		Subject:   "user-test-001",
		AccountID: "acct-test-001",
		Env:       "test",
		IssuedAt:  time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC).Unix(),
		ExpiresAt: time.Date(2026, 4, 6, 8, 5, 0, 0, time.UTC).Unix(),
	})

	if !strings.Contains(token, ".") {
		t.Fatalf("expected jwt-like token, got %q", token)
	}
}

func newTestOAuthVerifier(t *testing.T) *OAuthVerifier {
	t.Helper()

	verifier, err := NewOAuthVerifier(
		"safe-test-issuer",
		"safe-control-plane",
		[]byte("0123456789abcdef0123456789abcdef"),
	)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	verifier.SetNowForTest(func() time.Time {
		return time.Date(2026, 4, 6, 8, 1, 0, 0, time.UTC)
	})

	return verifier
}

func newTestOAuthToken(t *testing.T, identity OAuthIdentity) string {
	t.Helper()

	token, err := IssueTestOAuthToken(
		"safe-test-issuer",
		"safe-control-plane",
		[]byte("0123456789abcdef0123456789abcdef"),
		identity,
	)
	if err != nil {
		t.Fatalf("issue test token: %v", err)
	}

	return token
}
