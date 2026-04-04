package domain

import (
	"encoding/base64"
	"encoding/json"
)

type LocalUnlockKDF struct {
	Name        string `json:"name"`
	Salt        string `json:"salt"`
	MemoryKiB   int    `json:"memoryKiB"`
	TimeCost    int    `json:"timeCost"`
	Parallelism int    `json:"parallelism"`
	KeyBytes    int    `json:"keyBytes"`
}

type LocalUnlockWrappedKey struct {
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type LocalUnlockRecord struct {
	SchemaVersion int                   `json:"schemaVersion"`
	AccountID     string                `json:"accountId"`
	KDF           LocalUnlockKDF        `json:"kdf"`
	WrappedKey    LocalUnlockWrappedKey `json:"wrappedKey"`
}

func (record LocalUnlockRecord) Validate() error {
	if record.SchemaVersion != 1 {
		return ErrInvalidLocalUnlockRecord("schemaVersion")
	}
	if record.AccountID == "" {
		return ErrInvalidLocalUnlockRecord("accountId")
	}
	if record.KDF.Name != "argon2id" {
		return ErrInvalidLocalUnlockRecord("kdf.name")
	}
	if !isValidRawBase64(record.KDF.Salt) {
		return ErrInvalidLocalUnlockRecord("kdf.salt")
	}
	if record.KDF.MemoryKiB < 1 {
		return ErrInvalidLocalUnlockRecord("kdf.memoryKiB")
	}
	if record.KDF.TimeCost < 1 {
		return ErrInvalidLocalUnlockRecord("kdf.timeCost")
	}
	if record.KDF.Parallelism < 1 || record.KDF.Parallelism > 255 {
		return ErrInvalidLocalUnlockRecord("kdf.parallelism")
	}
	if record.KDF.KeyBytes != 32 {
		return ErrInvalidLocalUnlockRecord("kdf.keyBytes")
	}
	if record.WrappedKey.Algorithm != "aes-256-gcm" {
		return ErrInvalidLocalUnlockRecord("wrappedKey.algorithm")
	}
	if !isValidRawBase64(record.WrappedKey.Nonce) {
		return ErrInvalidLocalUnlockRecord("wrappedKey.nonce")
	}
	if !isValidRawBase64(record.WrappedKey.Ciphertext) {
		return ErrInvalidLocalUnlockRecord("wrappedKey.ciphertext")
	}

	return nil
}

func (record LocalUnlockRecord) CanonicalJSON() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}

	type canonicalLocalUnlockRecord struct {
		SchemaVersion int                   `json:"schemaVersion"`
		AccountID     string                `json:"accountId"`
		KDF           LocalUnlockKDF        `json:"kdf"`
		WrappedKey    LocalUnlockWrappedKey `json:"wrappedKey"`
	}

	return json.Marshal(canonicalLocalUnlockRecord{
		SchemaVersion: record.SchemaVersion,
		AccountID:     record.AccountID,
		KDF:           record.KDF,
		WrappedKey:    record.WrappedKey,
	})
}

func ParseLocalUnlockRecordJSON(data []byte) (LocalUnlockRecord, error) {
	var record LocalUnlockRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return LocalUnlockRecord{}, err
	}

	if err := record.Validate(); err != nil {
		return LocalUnlockRecord{}, err
	}

	return record, nil
}

type invalidLocalUnlockRecordError string

func (field invalidLocalUnlockRecordError) Error() string {
	return "invalid local unlock record field: " + string(field)
}

func ErrInvalidLocalUnlockRecord(field string) error {
	return invalidLocalUnlockRecordError(field)
}

func isValidRawBase64(value string) bool {
	if value == "" {
		return false
	}

	_, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil
}
