package domain

import (
	"encoding/json"
	"fmt"
)

// LocalDeviceRecord is the persisted record for an enrolled device.
// Stored at accounts/<accountID>/devices/<deviceID>.json.
// See I8 in docs/project/INTERFACES.md.
type LocalDeviceRecord struct {
	SchemaVersion       int    `json:"schemaVersion"`
	AccountID           string `json:"accountId"`
	DeviceID            string `json:"deviceId"`
	Label               string `json:"label"`
	DeviceType          string `json:"deviceType"`
	SigningPublicKey    string `json:"signingPublicKey"`
	EncryptionPublicKey string `json:"encryptionPublicKey"`
	CreatedAt           string `json:"createdAt"`
	Status              string `json:"status"`
}

// DeviceEnrollmentWrappedKey holds the ECIES-wrapped AMK for existing-device approval.
// Algorithm is always "x25519-hkdf-aes-256-gcm" (D12).
type DeviceEnrollmentWrappedKey struct {
	Algorithm          string `json:"algorithm"`
	EphemeralPublicKey string `json:"ephemeralPublicKey"`
	Nonce              string `json:"nonce"`
	Ciphertext         string `json:"ciphertext"`
}

// DeviceEnrollmentBundle is created by an existing trusted device to transfer the AMK
// to a newly enrolling device. See I8 in docs/project/INTERFACES.md.
type DeviceEnrollmentBundle struct {
	SchemaVersion int                        `json:"schemaVersion"`
	AccountID     string                     `json:"accountId"`
	DeviceID      string                     `json:"deviceId"`
	WrappedKey    DeviceEnrollmentWrappedKey `json:"wrappedKey"`
}

func (r LocalDeviceRecord) Validate() error {
	if r.SchemaVersion != 1 {
		return ErrInvalidLocalDeviceRecord("schemaVersion")
	}
	if r.AccountID == "" {
		return ErrInvalidLocalDeviceRecord("accountId")
	}
	if r.DeviceID == "" {
		return ErrInvalidLocalDeviceRecord("deviceId")
	}
	if r.Label == "" {
		return ErrInvalidLocalDeviceRecord("label")
	}
	if r.DeviceType != "cli" && r.DeviceType != "web" {
		return ErrInvalidLocalDeviceRecord("deviceType")
	}
	if !isValidRawBase64(r.SigningPublicKey) {
		return ErrInvalidLocalDeviceRecord("signingPublicKey")
	}
	if !isValidRawBase64(r.EncryptionPublicKey) {
		return ErrInvalidLocalDeviceRecord("encryptionPublicKey")
	}
	if r.CreatedAt == "" {
		return ErrInvalidLocalDeviceRecord("createdAt")
	}
	if r.Status != "active" && r.Status != "revoked" {
		return ErrInvalidLocalDeviceRecord("status")
	}
	return nil
}

func (r LocalDeviceRecord) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	type canonical struct {
		SchemaVersion       int    `json:"schemaVersion"`
		AccountID           string `json:"accountId"`
		DeviceID            string `json:"deviceId"`
		Label               string `json:"label"`
		DeviceType          string `json:"deviceType"`
		SigningPublicKey    string `json:"signingPublicKey"`
		EncryptionPublicKey string `json:"encryptionPublicKey"`
		CreatedAt           string `json:"createdAt"`
		Status              string `json:"status"`
	}

	return json.Marshal(canonical{
		SchemaVersion:       r.SchemaVersion,
		AccountID:           r.AccountID,
		DeviceID:            r.DeviceID,
		Label:               r.Label,
		DeviceType:          r.DeviceType,
		SigningPublicKey:    r.SigningPublicKey,
		EncryptionPublicKey: r.EncryptionPublicKey,
		CreatedAt:           r.CreatedAt,
		Status:              r.Status,
	})
}

func ParseLocalDeviceRecordJSON(data []byte) (LocalDeviceRecord, error) {
	var r LocalDeviceRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return LocalDeviceRecord{}, err
	}
	if err := r.Validate(); err != nil {
		return LocalDeviceRecord{}, err
	}
	return r, nil
}

func (b DeviceEnrollmentBundle) Validate() error {
	if b.SchemaVersion != 1 {
		return ErrInvalidDeviceEnrollmentBundle("schemaVersion")
	}
	if b.AccountID == "" {
		return ErrInvalidDeviceEnrollmentBundle("accountId")
	}
	if b.DeviceID == "" {
		return ErrInvalidDeviceEnrollmentBundle("deviceId")
	}
	if b.WrappedKey.Algorithm != "x25519-hkdf-aes-256-gcm" {
		return ErrInvalidDeviceEnrollmentBundle("wrappedKey.algorithm")
	}
	if !isValidRawBase64(b.WrappedKey.EphemeralPublicKey) {
		return ErrInvalidDeviceEnrollmentBundle("wrappedKey.ephemeralPublicKey")
	}
	if !isValidRawBase64(b.WrappedKey.Nonce) {
		return ErrInvalidDeviceEnrollmentBundle("wrappedKey.nonce")
	}
	if !isValidRawBase64(b.WrappedKey.Ciphertext) {
		return ErrInvalidDeviceEnrollmentBundle("wrappedKey.ciphertext")
	}
	return nil
}

func ParseDeviceEnrollmentBundleJSON(data []byte) (DeviceEnrollmentBundle, error) {
	var b DeviceEnrollmentBundle
	if err := json.Unmarshal(data, &b); err != nil {
		return DeviceEnrollmentBundle{}, err
	}
	if err := b.Validate(); err != nil {
		return DeviceEnrollmentBundle{}, err
	}
	return b, nil
}

// DeviceEnrollmentRequest is published by a new device to request enrollment.
// An existing trusted device reads this record, creates a DeviceEnrollmentBundle,
// and writes the bundle at the corresponding enrollment bundle path.
// Stored at accounts/<accountID>/enrollments/<deviceID>/request.json.
type DeviceEnrollmentRequest struct {
	SchemaVersion       int    `json:"schemaVersion"`
	AccountID           string `json:"accountId"`
	DeviceID            string `json:"deviceId"`
	Label               string `json:"label"`
	DeviceType          string `json:"deviceType"`
	EncryptionPublicKey string `json:"encryptionPublicKey"`
	RequestedAt         string `json:"requestedAt"`
}

func (r DeviceEnrollmentRequest) Validate() error {
	if r.SchemaVersion != 1 {
		return ErrInvalidDeviceEnrollmentRequest("schemaVersion")
	}
	if r.AccountID == "" {
		return ErrInvalidDeviceEnrollmentRequest("accountId")
	}
	if r.DeviceID == "" {
		return ErrInvalidDeviceEnrollmentRequest("deviceId")
	}
	if r.Label == "" {
		return ErrInvalidDeviceEnrollmentRequest("label")
	}
	if r.DeviceType != "cli" && r.DeviceType != "web" {
		return ErrInvalidDeviceEnrollmentRequest("deviceType")
	}
	if !isValidRawBase64(r.EncryptionPublicKey) {
		return ErrInvalidDeviceEnrollmentRequest("encryptionPublicKey")
	}
	if r.RequestedAt == "" {
		return ErrInvalidDeviceEnrollmentRequest("requestedAt")
	}
	return nil
}

func (r DeviceEnrollmentRequest) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	type canonical struct {
		SchemaVersion       int    `json:"schemaVersion"`
		AccountID           string `json:"accountId"`
		DeviceID            string `json:"deviceId"`
		Label               string `json:"label"`
		DeviceType          string `json:"deviceType"`
		EncryptionPublicKey string `json:"encryptionPublicKey"`
		RequestedAt         string `json:"requestedAt"`
	}

	return json.Marshal(canonical{
		SchemaVersion:       r.SchemaVersion,
		AccountID:           r.AccountID,
		DeviceID:            r.DeviceID,
		Label:               r.Label,
		DeviceType:          r.DeviceType,
		EncryptionPublicKey: r.EncryptionPublicKey,
		RequestedAt:         r.RequestedAt,
	})
}

func ParseDeviceEnrollmentRequestJSON(data []byte) (DeviceEnrollmentRequest, error) {
	var r DeviceEnrollmentRequest
	if err := json.Unmarshal(data, &r); err != nil {
		return DeviceEnrollmentRequest{}, err
	}
	if err := r.Validate(); err != nil {
		return DeviceEnrollmentRequest{}, err
	}
	return r, nil
}

type invalidDeviceEnrollmentRequestError string

func (e invalidDeviceEnrollmentRequestError) Error() string {
	return fmt.Sprintf("invalid device enrollment request field: %s", string(e))
}

func ErrInvalidDeviceEnrollmentRequest(field string) error {
	return invalidDeviceEnrollmentRequestError(field)
}

type invalidLocalDeviceRecordError string

func (e invalidLocalDeviceRecordError) Error() string {
	return fmt.Sprintf("invalid local device record field: %s", string(e))
}

func ErrInvalidLocalDeviceRecord(field string) error {
	return invalidLocalDeviceRecordError(field)
}

type invalidDeviceEnrollmentBundleError string

func (e invalidDeviceEnrollmentBundleError) Error() string {
	return fmt.Sprintf("invalid device enrollment bundle field: %s", string(e))
}

func ErrInvalidDeviceEnrollmentBundle(field string) error {
	return invalidDeviceEnrollmentBundleError(field)
}
