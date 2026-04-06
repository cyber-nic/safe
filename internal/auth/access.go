package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	ActionGet  = "get"
	ActionPut  = "put"
	ActionList = "list"
)

var validActions = []string{ActionGet, ActionPut, ActionList}

type DeviceStatusLookup func(accountID, deviceID string) (bool, error)

type CapabilitySigner struct {
	keyID     string
	secretKey []byte
	now       func() time.Time
}

type AccountAccessRequest struct {
	AccountID      string
	DeviceID       string
	Bucket         string
	AllowedActions []string
	TTL            time.Duration
}

type AccessCapability struct {
	Version        int      `json:"version"`
	AccountID      string   `json:"accountId"`
	DeviceID       string   `json:"deviceId"`
	Bucket         string   `json:"bucket"`
	Prefix         string   `json:"prefix"`
	AllowedActions []string `json:"allowedActions"`
	IssuedAt       string   `json:"issuedAt"`
	ExpiresAt      string   `json:"expiresAt"`
}

type SignedCapability struct {
	KeyID      string `json:"keyId"`
	Token      string `json:"token"`
	Capability AccessCapability
}

type AccessCheck struct {
	Bucket string
	Action string
	Key    string
	Now    time.Time
}

func NewCapabilitySigner(keyID string, secretKey []byte) (*CapabilitySigner, error) {
	if keyID == "" {
		return nil, ErrInvalidCapability("keyId")
	}
	if len(secretKey) < 32 {
		return nil, ErrInvalidCapability("secretKey")
	}

	return &CapabilitySigner{
		keyID:     keyID,
		secretKey: append([]byte(nil), secretKey...),
		now:       time.Now,
	}, nil
}

func (s *CapabilitySigner) SetNowForTest(now func() time.Time) {
	if now != nil {
		s.now = now
	}
}

func (s *CapabilitySigner) IssueAccountCapability(request AccountAccessRequest) (SignedCapability, error) {
	if err := validateAccountAccessRequest(request); err != nil {
		return SignedCapability{}, err
	}

	issuedAt := s.now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(request.TTL)
	capability := AccessCapability{
		Version:        1,
		AccountID:      request.AccountID,
		DeviceID:       request.DeviceID,
		Bucket:         request.Bucket,
		Prefix:         accountPrefix(request.AccountID),
		AllowedActions: normalizeActions(request.AllowedActions),
		IssuedAt:       issuedAt.Format(time.RFC3339),
		ExpiresAt:      expiresAt.Format(time.RFC3339),
	}

	token, err := s.sign(capability)
	if err != nil {
		return SignedCapability{}, err
	}

	return SignedCapability{
		KeyID:      s.keyID,
		Token:      token,
		Capability: capability,
	}, nil
}

func (s *CapabilitySigner) VerifyAccountAccess(token string, check AccessCheck) (AccessCapability, error) {
	capability, err := s.parseAndVerify(token)
	if err != nil {
		return AccessCapability{}, err
	}
	if err := capability.Validate(); err != nil {
		return AccessCapability{}, err
	}
	if capability.Bucket != check.Bucket {
		return AccessCapability{}, ErrForbiddenAccess("bucket")
	}
	if !containsAction(capability.AllowedActions, check.Action) {
		return AccessCapability{}, ErrForbiddenAccess("action")
	}
	if !strings.HasPrefix(check.Key, capability.Prefix) {
		return AccessCapability{}, ErrForbiddenAccess("prefix")
	}
	if !isNormalizedObjectKey(check.Key) {
		return AccessCapability{}, ErrForbiddenAccess("key")
	}

	now := check.Now
	if now.IsZero() {
		now = s.now()
	}
	expiresAt, _ := time.Parse(time.RFC3339, capability.ExpiresAt)
	if !now.UTC().Before(expiresAt) {
		return AccessCapability{}, ErrExpiredCapability()
	}

	return capability, nil
}

func (c AccessCapability) Validate() error {
	if c.Version != 1 {
		return ErrInvalidCapability("version")
	}
	if !isValidIdentitySegment(c.AccountID) {
		return ErrInvalidCapability("accountId")
	}
	if !isValidIdentitySegment(c.DeviceID) {
		return ErrInvalidCapability("deviceId")
	}
	if c.Bucket == "" {
		return ErrInvalidCapability("bucket")
	}
	if c.Prefix != accountPrefix(c.AccountID) {
		return ErrInvalidCapability("prefix")
	}
	if len(c.AllowedActions) == 0 {
		return ErrInvalidCapability("allowedActions")
	}
	if !isSortedUniqueActions(c.AllowedActions) {
		return ErrInvalidCapability("allowedActions")
	}
	for _, action := range c.AllowedActions {
		if !containsAction(validActions, action) {
			return ErrInvalidCapability("allowedActions")
		}
	}
	issuedAt, err := time.Parse(time.RFC3339, c.IssuedAt)
	if err != nil {
		return ErrInvalidCapability("issuedAt")
	}
	expiresAt, err := time.Parse(time.RFC3339, c.ExpiresAt)
	if err != nil {
		return ErrInvalidCapability("expiresAt")
	}
	if !issuedAt.Before(expiresAt) {
		return ErrInvalidCapability("expiresAt")
	}

	return nil
}

func ValidateActiveDevice(accountID, deviceID string, lookup DeviceStatusLookup) error {
	if !isValidIdentitySegment(accountID) {
		return ErrInvalidCapability("accountId")
	}
	if !isValidIdentitySegment(deviceID) {
		return ErrInvalidCapability("deviceId")
	}
	if lookup == nil {
		return nil
	}

	active, err := lookup(accountID, deviceID)
	if err != nil {
		return err
	}
	if !active {
		return ErrForbiddenAccess("device")
	}

	return nil
}

func (s *CapabilitySigner) sign(capability AccessCapability) (string, error) {
	payload, err := canonicalCapabilityPayload(capability)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write(payload)
	signature := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (s *CapabilitySigner) parseAndVerify(token string) (AccessCapability, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return AccessCapability{}, ErrInvalidCapability("token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return AccessCapability{}, ErrInvalidCapability("token")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return AccessCapability{}, ErrInvalidCapability("token")
	}

	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return AccessCapability{}, ErrInvalidCapability("signature")
	}

	var capability AccessCapability
	if err := json.Unmarshal(payload, &capability); err != nil {
		return AccessCapability{}, ErrInvalidCapability("token")
	}

	return capability, nil
}

func canonicalCapabilityPayload(capability AccessCapability) ([]byte, error) {
	if err := capability.Validate(); err != nil {
		return nil, err
	}

	type canonical struct {
		Version        int      `json:"version"`
		AccountID      string   `json:"accountId"`
		DeviceID       string   `json:"deviceId"`
		Bucket         string   `json:"bucket"`
		Prefix         string   `json:"prefix"`
		AllowedActions []string `json:"allowedActions"`
		IssuedAt       string   `json:"issuedAt"`
		ExpiresAt      string   `json:"expiresAt"`
	}

	return json.Marshal(canonical{
		Version:        capability.Version,
		AccountID:      capability.AccountID,
		DeviceID:       capability.DeviceID,
		Bucket:         capability.Bucket,
		Prefix:         capability.Prefix,
		AllowedActions: capability.AllowedActions,
		IssuedAt:       capability.IssuedAt,
		ExpiresAt:      capability.ExpiresAt,
	})
}

func validateAccountAccessRequest(request AccountAccessRequest) error {
	if !isValidIdentitySegment(request.AccountID) {
		return ErrInvalidCapability("accountId")
	}
	if !isValidIdentitySegment(request.DeviceID) {
		return ErrInvalidCapability("deviceId")
	}
	if request.Bucket == "" {
		return ErrInvalidCapability("bucket")
	}
	if request.TTL <= 0 || request.TTL > 15*time.Minute {
		return ErrInvalidCapability("ttl")
	}
	actions := normalizeActions(request.AllowedActions)
	if len(actions) == 0 {
		return ErrInvalidCapability("allowedActions")
	}

	request.AllowedActions = actions
	return nil
}

func normalizeActions(actions []string) []string {
	if len(actions) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(actions))
	for _, action := range actions {
		normalized = append(normalized, strings.ToLower(strings.TrimSpace(action)))
	}
	slices.Sort(normalized)

	unique := normalized[:0]
	for _, action := range normalized {
		if action == "" {
			continue
		}
		if len(unique) == 0 || unique[len(unique)-1] != action {
			unique = append(unique, action)
		}
	}

	return unique
}

func accountPrefix(accountID string) string {
	return fmt.Sprintf("accounts/%s/", accountID)
}

func isValidIdentitySegment(value string) bool {
	return value != "" && !strings.Contains(value, "/") && !strings.Contains(value, "..")
}

func isSortedUniqueActions(actions []string) bool {
	if len(actions) == 0 {
		return false
	}
	for i := 1; i < len(actions); i++ {
		if actions[i-1] >= actions[i] {
			return false
		}
	}
	return true
}

func containsAction(actions []string, want string) bool {
	return slices.Contains(actions, strings.ToLower(want))
}

func isNormalizedObjectKey(key string) bool {
	if key == "" || strings.HasPrefix(key, "/") || strings.HasSuffix(key, "/") {
		return false
	}
	parts := strings.Split(key, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

type invalidCapabilityError string

func (e invalidCapabilityError) Error() string {
	return fmt.Sprintf("invalid access capability field: %s", string(e))
}

func ErrInvalidCapability(field string) error {
	return invalidCapabilityError(field)
}

type forbiddenAccessError string

func (e forbiddenAccessError) Error() string {
	return fmt.Sprintf("forbidden access: %s", string(e))
}

func ErrForbiddenAccess(reason string) error {
	return forbiddenAccessError(reason)
}

type expiredCapabilityError struct{}

func (expiredCapabilityError) Error() string {
	return "expired access capability"
}

func ErrExpiredCapability() error {
	return expiredCapabilityError{}
}
