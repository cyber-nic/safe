package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	bip39 "github.com/tyler-smith/go-bip39"

	"github.com/ndelorme/safe/internal/domain"
)

const (
	localRecoverySchemaVersion = 1
	localRecoveryAADPrefix     = "safe.local-recovery.v1/"
	recoveryKeySize            = 32 // 256-bit random key; no KDF (see D9, I6)
)

var ErrRecoveryFailed = errors.New("local recovery authentication failed")

// GenerateRecoveryKey returns 32 cryptographically random bytes for use as a recovery key.
func GenerateRecoveryKey() ([]byte, error) {
	return randomBytes(rand.Reader, recoveryKeySize)
}

// RecoveryKeyMnemonic encodes raw recovery key bytes as a 24-word BIP-39 mnemonic.
func RecoveryKeyMnemonic(keyBytes []byte) (string, error) {
	if len(keyBytes) != recoveryKeySize {
		return "", fmt.Errorf("recovery key must be %d bytes, got %d", recoveryKeySize, len(keyBytes))
	}
	mnemonic, err := bip39.NewMnemonic(keyBytes)
	if err != nil {
		return "", fmt.Errorf("encode recovery key mnemonic: %w", err)
	}
	return mnemonic, nil
}

// CreateLocalRecoveryRecord wraps AMK with the recovery key and returns the persisted record
// plus the BIP-39 mnemonic for one-time display to the user.
// No KDF is applied to recoveryKeyBytes — 32 bytes of CSPRNG output is used directly (D9).
func CreateLocalRecoveryRecord(accountID string, recoveryKeyBytes, amk []byte) (domain.LocalRecoveryRecord, string, error) {
	return createLocalRecoveryRecord(accountID, recoveryKeyBytes, amk, rand.Reader)
}

// OpenLocalRecoveryRecord unwraps the AMK using the raw recovery key bytes.
// Returns ErrRecoveryFailed for a wrong key or corrupted ciphertext.
func OpenLocalRecoveryRecord(record domain.LocalRecoveryRecord, recoveryKeyBytes []byte) ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}
	if len(recoveryKeyBytes) != recoveryKeySize {
		return nil, ErrRecoveryFailed
	}

	nonce, err := decodeRaw(record.WrappedKey.Nonce)
	if err != nil {
		return nil, ErrRecoveryFailed
	}
	ciphertext, err := decodeRaw(record.WrappedKey.Ciphertext)
	if err != nil {
		return nil, ErrRecoveryFailed
	}

	aad := []byte(localRecoveryAAD(record.AccountID))
	plaintext, err := decryptBytes(recoveryKeyBytes, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrRecoveryFailed
	}
	return plaintext, nil
}

func createLocalRecoveryRecord(accountID string, recoveryKeyBytes, amk []byte, random io.Reader) (domain.LocalRecoveryRecord, string, error) {
	if accountID == "" {
		return domain.LocalRecoveryRecord{}, "", fmt.Errorf("accountID is required")
	}
	if len(recoveryKeyBytes) != recoveryKeySize {
		return domain.LocalRecoveryRecord{}, "", fmt.Errorf("recovery key must be %d bytes, got %d", recoveryKeySize, len(recoveryKeyBytes))
	}
	if len(amk) != accountMasterKeyBytes {
		return domain.LocalRecoveryRecord{}, "", fmt.Errorf("AMK must be %d bytes, got %d", accountMasterKeyBytes, len(amk))
	}

	aad := []byte(localRecoveryAAD(accountID))
	nonce, ciphertext, err := encryptBytes(recoveryKeyBytes, amk, aad, random)
	if err != nil {
		return domain.LocalRecoveryRecord{}, "", err
	}

	record := domain.LocalRecoveryRecord{
		SchemaVersion: localRecoverySchemaVersion,
		AccountID:     accountID,
		WrappedKey: domain.LocalRecoveryWrappedKey{
			Algorithm:  "aes-256-gcm",
			Nonce:      encodeRaw(nonce),
			Ciphertext: encodeRaw(ciphertext),
		},
	}

	mnemonic, err := RecoveryKeyMnemonic(recoveryKeyBytes)
	if err != nil {
		return domain.LocalRecoveryRecord{}, "", err
	}

	return record, mnemonic, nil
}

func localRecoveryAAD(accountID string) string {
	return localRecoveryAADPrefix + accountID
}
