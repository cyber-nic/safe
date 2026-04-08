package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
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

type OAuthVerifierConfig struct {
	Issuer     string
	Audience   string
	Env        string
	DevMode    bool
	SecretKey  []byte
	JWKSURL    string
	HTTPClient *http.Client
}

type OAuthVerifier struct {
	issuer   string
	audience string
	env      string
	now      func() time.Time
	verify   func(parts []string, header oauthTokenHeader, signature []byte) error
}

type oauthTokenHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

func NewOAuthVerifier(config OAuthVerifierConfig) (*OAuthVerifier, error) {
	if config.Issuer == "" {
		return nil, ErrInvalidOAuthToken("issuer")
	}
	if config.Audience == "" {
		return nil, ErrInvalidOAuthToken("audience")
	}

	verifier := &OAuthVerifier{
		issuer:   config.Issuer,
		audience: config.Audience,
		env:      config.Env,
		now:      time.Now,
	}

	if config.DevMode {
		if len(config.SecretKey) < 32 {
			return nil, ErrInvalidOAuthToken("secretKey")
		}
		secretKey := append([]byte(nil), config.SecretKey...)
		verifier.verify = func(parts []string, header oauthTokenHeader, signature []byte) error {
			if header.Alg != "HS256" {
				return ErrInvalidOAuthToken("alg")
			}
			mac := hmac.New(sha256.New, secretKey)
			mac.Write([]byte(parts[0] + "." + parts[1]))
			if !hmac.Equal(signature, mac.Sum(nil)) {
				return ErrInvalidOAuthToken("signature")
			}
			return nil
		}
		return verifier, nil
	}

	if config.JWKSURL == "" {
		return nil, ErrInvalidOAuthToken("jwks")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	jwks := &jwksVerifier{
		url:    config.JWKSURL,
		client: httpClient,
	}
	verifier.verify = func(parts []string, header oauthTokenHeader, signature []byte) error {
		return jwks.verify(parts, header, signature)
	}

	return verifier, nil
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

	var header oauthTokenHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	if header.Typ != "" && header.Typ != "JWT" {
		return OAuthIdentity{}, ErrInvalidOAuthToken("token")
	}
	if err := v.verify(parts, header, signature); err != nil {
		return OAuthIdentity{}, err
	}

	identity, err := decodeOAuthIdentity(payloadJSON)
	if err != nil {
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

func decodeOAuthIdentity(payloadJSON []byte) (OAuthIdentity, error) {
	var claims struct {
		Issuer    string          `json:"iss"`
		Audience  json.RawMessage `json:"aud"`
		Subject   string          `json:"sub"`
		AccountID string          `json:"accountId"`
		Env       string          `json:"env"`
		IssuedAt  int64           `json:"iat"`
		ExpiresAt int64           `json:"exp"`
	}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return OAuthIdentity{}, err
	}

	audience, err := decodeAudience(claims.Audience)
	if err != nil {
		return OAuthIdentity{}, err
	}
	accountID := claims.AccountID
	if accountID == "" && claims.Issuer != "" && claims.Subject != "" {
		accountID = deriveOAuthAccountID(claims.Issuer, claims.Subject)
	}

	return OAuthIdentity{
		Issuer:    claims.Issuer,
		Audience:  audience,
		Subject:   claims.Subject,
		AccountID: accountID,
		Env:       claims.Env,
		IssuedAt:  claims.IssuedAt,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func decodeAudience(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", ErrInvalidOAuthToken("audience")
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return single, nil
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err != nil || len(list) == 0 {
		return "", ErrInvalidOAuthToken("audience")
	}

	return list[0], nil
}

func deriveOAuthAccountID(issuer, subject string) string {
	sum := sha256.Sum256([]byte(issuer + ":" + subject))
	return "acct-oauth-" + hex.EncodeToString(sum[:16])
}

type jwksVerifier struct {
	url    string
	client *http.Client
}

func (v *jwksVerifier) verify(parts []string, header oauthTokenHeader, signature []byte) error {
	switch header.Alg {
	case "RS256", "ES256":
	default:
		return ErrInvalidOAuthToken("alg")
	}

	keys, err := v.fetchKeys()
	if err != nil {
		return ErrInvalidOAuthToken("token")
	}

	key, err := selectJWK(keys, header)
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	switch header.Alg {
	case "RS256":
		publicKey, err := key.rsaPublicKey()
		if err != nil {
			return ErrInvalidOAuthToken("signature")
		}
		if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
			return ErrInvalidOAuthToken("signature")
		}
	case "ES256":
		publicKey, err := key.ecdsaPublicKey()
		if err != nil {
			return ErrInvalidOAuthToken("signature")
		}
		if len(signature) != 64 {
			return ErrInvalidOAuthToken("signature")
		}
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:])
		if !ecdsa.Verify(publicKey, hash[:], r, s) {
			return ErrInvalidOAuthToken("signature")
		}
	}

	return nil
}

func (v *jwksVerifier) fetchKeys() ([]jwkKey, error) {
	request, err := http.NewRequest(http.MethodGet, v.url, nil)
	if err != nil {
		return nil, err
	}
	response, err := v.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks status: %d", response.StatusCode)
	}

	var payload struct {
		Keys []jwkKey `json:"keys"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Keys) == 0 {
		return nil, ErrInvalidOAuthToken("jwks")
	}
	return payload.Keys, nil
}

type jwkKey struct {
	KeyType string `json:"kty"`
	KeyID   string `json:"kid"`
	Use     string `json:"use"`
	Alg     string `json:"alg"`
	N       string `json:"n"`
	E       string `json:"e"`
	Curve   string `json:"crv"`
	X       string `json:"x"`
	Y       string `json:"y"`
}

func selectJWK(keys []jwkKey, header oauthTokenHeader) (jwkKey, error) {
	if header.Kid != "" {
		for _, key := range keys {
			if key.KeyID == header.Kid {
				return key, nil
			}
		}
		return jwkKey{}, ErrInvalidOAuthToken("signature")
	}

	if len(keys) == 1 {
		return keys[0], nil
	}

	for _, key := range keys {
		if key.Alg == header.Alg {
			return key, nil
		}
	}
	return jwkKey{}, ErrInvalidOAuthToken("signature")
}

func (k jwkKey) rsaPublicKey() (*rsa.PublicKey, error) {
	if k.KeyType != "RSA" || k.N == "" || k.E == "" {
		return nil, ErrInvalidOAuthToken("signature")
	}
	modulus, err := decodeBase64URLBigInt(k.N)
	if err != nil {
		return nil, err
	}
	exponent, err := decodeBase64URLBigInt(k.E)
	if err != nil {
		return nil, err
	}
	return &rsa.PublicKey{
		N: modulus,
		E: int(exponent.Int64()),
	}, nil
}

func (k jwkKey) ecdsaPublicKey() (*ecdsa.PublicKey, error) {
	if k.KeyType != "EC" || k.Curve != "P-256" || k.X == "" || k.Y == "" {
		return nil, ErrInvalidOAuthToken("signature")
	}
	x, err := decodeBase64URLBigInt(k.X)
	if err != nil {
		return nil, err
	}
	y, err := decodeBase64URLBigInt(k.Y)
	if err != nil {
		return nil, err
	}
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}, nil
}

func decodeBase64URLBigInt(value string) (*big.Int, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return new(big.Int).SetBytes(raw), nil
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
