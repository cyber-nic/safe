package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/argon2"

	"github.com/ndelorme/safe/internal/domain"
)

const (
	localUnlockSchemaVersion = 1
	localUnlockAADPrefix     = "safe.local-unlock.v1/"
	secretEnvelopeSchema     = 1
	secretEnvelopeAAD        = "safe.secret-material.v1"

	defaultArgonMemoryKiB   = 64 * 1024
	defaultArgonTimeCost    = 3
	defaultArgonParallelism = 4
	defaultDerivedKeyBytes  = 32
	accountMasterKeyBytes   = 32
)

var (
	ErrUnlockFailed          = errors.New("local unlock authentication failed")
	ErrSecretMaterialDecrypt = errors.New("secret material authentication failed")
)

type SecretMaterialEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Algorithm     string `json:"algorithm"`
	Nonce         string `json:"nonce"`
	Ciphertext    string `json:"ciphertext"`
}

func CreateLocalUnlockRecord(accountID, password string) (domain.LocalUnlockRecord, []byte, error) {
	return createLocalUnlockRecord(accountID, password, rand.Reader)
}

func OpenLocalUnlockRecord(record domain.LocalUnlockRecord, password string) ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}

	salt, err := decodeRaw(record.KDF.Salt)
	if err != nil {
		return nil, err
	}
	nonce, err := decodeRaw(record.WrappedKey.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := decodeRaw(record.WrappedKey.Ciphertext)
	if err != nil {
		return nil, err
	}

	kek := deriveKey(password, salt, record.KDF)
	plaintext, err := decryptBytes(kek, nonce, ciphertext, []byte(localUnlockAAD(record.AccountID)))
	if err != nil {
		return nil, ErrUnlockFailed
	}

	return plaintext, nil
}

func EncryptSecretMaterial(accountKey, plaintext []byte) ([]byte, error) {
	return encryptSecretMaterial(accountKey, plaintext, rand.Reader)
}

func DecryptSecretMaterial(accountKey, payload []byte) ([]byte, error) {
	if len(accountKey) != defaultDerivedKeyBytes {
		return nil, fmt.Errorf("invalid account key length: %d", len(accountKey))
	}

	var envelope SecretMaterialEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}
	if err := envelope.Validate(); err != nil {
		return nil, err
	}

	nonce, err := decodeRaw(envelope.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := decodeRaw(envelope.Ciphertext)
	if err != nil {
		return nil, err
	}

	plaintext, err := decryptBytes(accountKey, nonce, ciphertext, []byte(secretEnvelopeAAD))
	if err != nil {
		return nil, ErrSecretMaterialDecrypt
	}

	return plaintext, nil
}

func (envelope SecretMaterialEnvelope) Validate() error {
	if envelope.SchemaVersion != secretEnvelopeSchema {
		return fmt.Errorf("invalid secret material envelope schemaVersion: %d", envelope.SchemaVersion)
	}
	if envelope.Algorithm != "aes-256-gcm" {
		return fmt.Errorf("invalid secret material envelope algorithm: %s", envelope.Algorithm)
	}
	if _, err := decodeRaw(envelope.Nonce); err != nil {
		return fmt.Errorf("invalid secret material envelope nonce: %w", err)
	}
	if _, err := decodeRaw(envelope.Ciphertext); err != nil {
		return fmt.Errorf("invalid secret material envelope ciphertext: %w", err)
	}

	return nil
}

func createLocalUnlockRecord(accountID, password string, random io.Reader) (domain.LocalUnlockRecord, []byte, error) {
	if accountID == "" {
		return domain.LocalUnlockRecord{}, nil, fmt.Errorf("accountID is required")
	}
	if password == "" {
		return domain.LocalUnlockRecord{}, nil, fmt.Errorf("password is required")
	}

	salt, err := randomBytes(random, 16)
	if err != nil {
		return domain.LocalUnlockRecord{}, nil, err
	}
	accountKey, err := randomBytes(random, accountMasterKeyBytes)
	if err != nil {
		return domain.LocalUnlockRecord{}, nil, err
	}

	kdf := domain.LocalUnlockKDF{
		Name:        "argon2id",
		Salt:        encodeRaw(salt),
		MemoryKiB:   defaultArgonMemoryKiB,
		TimeCost:    defaultArgonTimeCost,
		Parallelism: defaultArgonParallelism,
		KeyBytes:    defaultDerivedKeyBytes,
	}

	kek := deriveKey(password, salt, kdf)
	nonce, ciphertext, err := encryptBytes(kek, accountKey, []byte(localUnlockAAD(accountID)), random)
	if err != nil {
		return domain.LocalUnlockRecord{}, nil, err
	}

	record := domain.LocalUnlockRecord{
		SchemaVersion: localUnlockSchemaVersion,
		AccountID:     accountID,
		KDF:           kdf,
		WrappedKey: domain.LocalUnlockWrappedKey{
			Algorithm:  "aes-256-gcm",
			Nonce:      encodeRaw(nonce),
			Ciphertext: encodeRaw(ciphertext),
		},
	}

	return record, accountKey, nil
}

func encryptSecretMaterial(accountKey, plaintext []byte, random io.Reader) ([]byte, error) {
	if len(accountKey) != defaultDerivedKeyBytes {
		return nil, fmt.Errorf("invalid account key length: %d", len(accountKey))
	}

	nonce, ciphertext, err := encryptBytes(accountKey, plaintext, []byte(secretEnvelopeAAD), random)
	if err != nil {
		return nil, err
	}

	envelope := SecretMaterialEnvelope{
		SchemaVersion: secretEnvelopeSchema,
		Algorithm:     "aes-256-gcm",
		Nonce:         encodeRaw(nonce),
		Ciphertext:    encodeRaw(ciphertext),
	}

	if err := envelope.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(envelope)
}

func deriveKey(password string, salt []byte, kdf domain.LocalUnlockKDF) []byte {
	return argon2.IDKey([]byte(password), salt, uint32(kdf.TimeCost), uint32(kdf.MemoryKiB), uint8(kdf.Parallelism), uint32(kdf.KeyBytes))
}

func encryptBytes(key, plaintext, aad []byte, random io.Reader) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce, err := randomBytes(random, gcm.NonceSize())
	if err != nil {
		return nil, nil, err
	}

	return nonce, gcm.Seal(nil, nonce, plaintext, aad), nil
}

func decryptBytes(key, nonce, ciphertext, aad []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, aad)
}

func randomBytes(random io.Reader, size int) ([]byte, error) {
	buf := make([]byte, size)
	if _, err := io.ReadFull(random, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func encodeRaw(value []byte) string {
	return base64.RawURLEncoding.EncodeToString(value)
}

func decodeRaw(value string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(value)
}

func localUnlockAAD(accountID string) string {
	return localUnlockAADPrefix + accountID
}
