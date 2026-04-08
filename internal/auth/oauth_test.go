package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
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

func TestOAuthVerifierVerifiesJWKSRS256Tokens(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": "rsa-test",
					"alg": "RS256",
					"use": "sig",
					"n":   base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes()),
				},
			},
		})
	}))
	defer jwksServer.Close()

	verifier, err := NewOAuthVerifier(OAuthVerifierConfig{
		Issuer:   "https://accounts.google.com",
		Audience: "client-test-123.apps.googleusercontent.com",
		JWKSURL:  jwksServer.URL,
	})
	if err != nil {
		t.Fatalf("new jwks verifier: %v", err)
	}
	verifier.SetNowForTest(func() time.Time {
		return time.Date(2026, 4, 8, 8, 1, 0, 0, time.UTC)
	})

	token := issueJWKSRS256Token(t, privateKey, map[string]any{
		"iss": "https://accounts.google.com",
		"aud": "client-test-123.apps.googleusercontent.com",
		"sub": "google-sub-001",
		"iat": time.Date(2026, 4, 8, 8, 0, 0, 0, time.UTC).Unix(),
		"exp": time.Date(2026, 4, 8, 8, 5, 0, 0, time.UTC).Unix(),
	})

	identity, err := verifier.VerifyAccessToken(token)
	if err != nil {
		t.Fatalf("verify jwks token: %v", err)
	}
	if identity.Subject != "google-sub-001" {
		t.Fatalf("unexpected subject: %q", identity.Subject)
	}
	if identity.AccountID != deriveOAuthAccountID("https://accounts.google.com", "google-sub-001") {
		t.Fatalf("unexpected derived account id: %q", identity.AccountID)
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

	verifier, err := NewOAuthVerifier(OAuthVerifierConfig{
		Issuer:    "safe-test-issuer",
		Audience:  "safe-control-plane",
		Env:       "test",
		DevMode:   true,
		SecretKey: []byte("0123456789abcdef0123456789abcdef"),
	})
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

func issueJWKSRS256Token(t *testing.T, privateKey *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()

	headerJSON, err := json.Marshal(map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": "rsa-test",
	})
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	hash := sha256.Sum256([]byte(headerPart + "." + payloadPart))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return headerPart + "." + payloadPart + "." + base64.RawURLEncoding.EncodeToString(signature)
}
