package domain

import (
	"encoding/json"
	"fmt"
)

// LocalRecoveryWrappedKey holds the AES-256-GCM encrypted AMK wrapped with the recovery key.
type LocalRecoveryWrappedKey struct {
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// LocalRecoveryRecord is the persisted record for account recovery via recovery key.
// Stored at accounts/<accountID>/recovery.json.
// The recovery key is 32 random bytes (no KDF); see I6 in docs/project/INTERFACES.md.
type LocalRecoveryRecord struct {
	SchemaVersion int                     `json:"schemaVersion"`
	AccountID     string                  `json:"accountId"`
	WrappedKey    LocalRecoveryWrappedKey `json:"wrappedKey"`
}

func (r LocalRecoveryRecord) Validate() error {
	if r.SchemaVersion != 1 {
		return ErrInvalidLocalRecoveryRecord("schemaVersion")
	}
	if r.AccountID == "" {
		return ErrInvalidLocalRecoveryRecord("accountId")
	}
	if r.WrappedKey.Algorithm != "aes-256-gcm" {
		return ErrInvalidLocalRecoveryRecord("wrappedKey.algorithm")
	}
	if !isValidRawBase64(r.WrappedKey.Nonce) {
		return ErrInvalidLocalRecoveryRecord("wrappedKey.nonce")
	}
	if !isValidRawBase64(r.WrappedKey.Ciphertext) {
		return ErrInvalidLocalRecoveryRecord("wrappedKey.ciphertext")
	}
	return nil
}

func (r LocalRecoveryRecord) CanonicalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}

	type canonical struct {
		SchemaVersion int                     `json:"schemaVersion"`
		AccountID     string                  `json:"accountId"`
		WrappedKey    LocalRecoveryWrappedKey `json:"wrappedKey"`
	}

	return json.Marshal(canonical{
		SchemaVersion: r.SchemaVersion,
		AccountID:     r.AccountID,
		WrappedKey:    r.WrappedKey,
	})
}

func ParseLocalRecoveryRecordJSON(data []byte) (LocalRecoveryRecord, error) {
	var r LocalRecoveryRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return LocalRecoveryRecord{}, err
	}
	if err := r.Validate(); err != nil {
		return LocalRecoveryRecord{}, err
	}
	return r, nil
}

type invalidLocalRecoveryRecordError string

func (e invalidLocalRecoveryRecordError) Error() string {
	return fmt.Sprintf("invalid local recovery record field: %s", string(e))
}

func ErrInvalidLocalRecoveryRecord(field string) error {
	return invalidLocalRecoveryRecordError(field)
}

