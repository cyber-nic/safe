package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type OAuthIdentity struct {
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	Subject   string `json:"sub"`
	AccountID string `json:"accountId"`
	Env       string `json:"env"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

type OAuthVerifier struct {
	issuer    string
	audience  string
	secretKey []byte
	now       func() time.Time
}

func NewOAuthVerifier(issuer, audience string, secretKey []byte) (*OAuthVerifier, error) {
	if issuer == "" {
		return nil, ErrInvalidOAuthToken("issuer")
	}
	if audience == "" {
		return nil, ErrInvalidOAuthToken("audience")
	}
	if len(secretKey) < 32 {
		return nil, ErrInvalidOAuthToken("secretKey")
	}

	return &OAuthVerifier{
		issuer:    issuer,
		audience:  audience,
		secretKey: append([]byte(nil), secretKey...),
		now:       time.Now,
	}, nil
}

func (v *OAuthVerifier) SetNowForTest(now func() time.Time) {
	if now != nil {
		v.now = now
	}
}

func (v *OAuthVerifier) VerifyBearerToken(header string) (OAuthIdentity, error) {
	token, err := bearerToken(header)
	if err != nil {
		return OAuthIdentity{}, err
	}

	return v.VerifyAccessToken(token)
}

func (v *OAuthVerifier) VerifyAccessToken(token string) (OAuthIdentity, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}

	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	if header.Alg != "HS256" || header.Typ != "JWT" {
		return OAuthIdentity{}, ErrInvalidOAuthToken("alg")
	}

	mac := hmac.New(sha256.New, v.secretKey)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return OAuthIdentity{}, ErrInvalidOAuthToken("signature")
	}

	var identity OAuthIdentity
	if err := json.Unmarshal(payloadJSON, &identity); err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	if err := v.validate(identity); err != nil {
		return OAuthIdentity{}, err
	}

	return identity, nil
}

func (v *OAuthVerifier) validate(identity OAuthIdentity) error {
	if identity.Issuer != v.issuer {
		return ErrInvalidOAuthToken("issuer")
	}
	if identity.Audience != v.audience {
		return ErrInvalidOAuthToken("audience")
	}
	if identity.Subject == "" {
		return ErrInvalidOAuthToken("subject")
	}
	if !isValidIdentitySegment(identity.AccountID) {
		return ErrInvalidOAuthToken("accountId")
	}
	if identity.Env == "" {
		return ErrInvalidOAuthToken("env")
	}
	if identity.IssuedAt <= 0 {
		return ErrInvalidOAuthToken("issuedAt")
	}
	if identity.ExpiresAt <= identity.IssuedAt {
		return ErrInvalidOAuthToken("expiresAt")
	}
	if !v.now().UTC().Before(time.Unix(identity.ExpiresAt, 0).UTC()) {
		return ErrExpiredOAuthToken()
	}

	return nil
}

func IssueTestOAuthToken(issuer, audience string, secretKey []byte, identity OAuthIdentity) (string, error) {
	if len(secretKey) < 32 {
		return "", ErrInvalidOAuthToken("secretKey")
	}

	headerJSON, err := json.Marshal(struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}{
		Alg: "HS256",
		Typ: "JWT",
	})
	if err != nil {
		return "", err
	}
	payload := identity
	payload.Issuer = issuer
	payload.Audience = audience
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerPart := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadPart := base64.RawURLEncoding.EncodeToString(payloadJSON)
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(headerPart + "." + payloadPart))
	signaturePart := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return headerPart + "." + payloadPart + "." + signaturePart, nil
}

func bearerToken(header string) (string, error) {
	if header == "" {
		return "", ErrMissingOAuthToken()
	}
	if !strings.HasPrefix(header, "Bearer ") {
		return "", ErrInvalidOAuthToken("authorization")
	}

	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return "", ErrMissingOAuthToken()
	}

	return token, nil
}

type invalidOAuthTokenError string

func (e invalidOAuthTokenError) Error() string {
	return fmt.Sprintf("invalid oauth token field: %s", string(e))
}

func ErrInvalidOAuthToken(field string) error {
	return invalidOAuthTokenError(field)
}

type missingOAuthTokenError struct{}

func (missingOAuthTokenError) Error() string {
	return "missing oauth bearer token"
}

func ErrMissingOAuthToken() error {
	return missingOAuthTokenError{}
}

type expiredOAuthTokenError struct{}

func (expiredOAuthTokenError) Error() string {
	return "expired oauth token"
}

func ErrExpiredOAuthToken() error {
	return expiredOAuthTokenError{}
}
