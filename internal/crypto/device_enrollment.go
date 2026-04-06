package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"

	"github.com/ndelorme/safe/internal/domain"
)

const (
	deviceEnrollmentSchemaVersion = 1
	deviceEnrollmentAADPrefix     = "safe.device-enrollment.v1/"
	deviceIDBytes                 = 16
	x25519KeySize                 = 32
)

var ErrDeviceEnrollmentFailed = errors.New("device enrollment authentication failed")

// DeviceKeyPair holds the two key pairs generated for a device at enrollment time.
// Ed25519 is used for signing; X25519 is used for key agreement during enrollment.
type DeviceKeyPair struct {
	SigningPrivateKey    ed25519.PrivateKey // 64 bytes; includes the public key
	SigningPublicKey     ed25519.PublicKey  // 32 bytes
	EncryptionPrivateKey []byte            // 32 bytes, X25519 scalar
	EncryptionPublicKey  []byte            // 32 bytes, X25519 point
}

// GenerateDeviceID returns a random 16-byte device ID, hex-encoded as a 32-character
// lowercase string. See D13 in docs/project/DECISIONS.md.
func GenerateDeviceID() (string, error) {
	b, err := randomBytes(rand.Reader, deviceIDBytes)
	if err != nil {
		return "", fmt.Errorf("generate device ID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateDeviceKeyPair generates an Ed25519 signing key pair and an X25519 encryption
// key pair for a new device. Private keys must remain local to the device.
func GenerateDeviceKeyPair() (DeviceKeyPair, error) {
	return generateDeviceKeyPair(rand.Reader)
}

// CreateDeviceRecord builds a LocalDeviceRecord from the provided public keys.
// createdAt is set to the current UTC time in RFC 3339 format.
func CreateDeviceRecord(accountID, deviceID, label, deviceType string, kp DeviceKeyPair) (domain.LocalDeviceRecord, error) {
	if accountID == "" {
		return domain.LocalDeviceRecord{}, fmt.Errorf("accountID is required")
	}
	if deviceID == "" {
		return domain.LocalDeviceRecord{}, fmt.Errorf("deviceID is required")
	}
	if label == "" {
		return domain.LocalDeviceRecord{}, fmt.Errorf("label is required")
	}
	if deviceType != "cli" && deviceType != "web" {
		return domain.LocalDeviceRecord{}, fmt.Errorf("deviceType must be cli or web")
	}

	return domain.LocalDeviceRecord{
		SchemaVersion:       deviceEnrollmentSchemaVersion,
		AccountID:           accountID,
		DeviceID:            deviceID,
		Label:               label,
		DeviceType:          deviceType,
		SigningPublicKey:    encodeRaw(kp.SigningPublicKey),
		EncryptionPublicKey: encodeRaw(kp.EncryptionPublicKey),
		CreatedAt:           time.Now().UTC().Format(time.RFC3339),
		Status:              "active",
	}, nil
}

// CreateDeviceEnrollmentBundle wraps the AMK for a new device using ECIES:
// ephemeral X25519 key agreement + HKDF-SHA256 + AES-256-GCM.
// deviceEncryptionPublicKey is the new device's X25519 public key (32 bytes).
// See I8 and D12 in docs/project/INTERFACES.md and DECISIONS.md.
func CreateDeviceEnrollmentBundle(accountID, deviceID string, deviceEncryptionPublicKey, amk []byte) (domain.DeviceEnrollmentBundle, error) {
	return createDeviceEnrollmentBundle(accountID, deviceID, deviceEncryptionPublicKey, amk, rand.Reader)
}

// OpenDeviceEnrollmentBundle unwraps the AMK from a DeviceEnrollmentBundle using
// the new device's X25519 private key.
// Returns ErrDeviceEnrollmentFailed for wrong key, corrupted ciphertext, wrong account
// ID, or wrong device ID.
func OpenDeviceEnrollmentBundle(bundle domain.DeviceEnrollmentBundle, deviceID string, deviceEncryptionPrivateKey []byte) ([]byte, error) {
	if err := bundle.Validate(); err != nil {
		return nil, err
	}
	if bundle.DeviceID != deviceID {
		return nil, ErrDeviceEnrollmentFailed
	}
	if len(deviceEncryptionPrivateKey) != x25519KeySize {
		return nil, ErrDeviceEnrollmentFailed
	}

	ephemeralPub, err := decodeRaw(bundle.WrappedKey.EphemeralPublicKey)
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}
	nonce, err := decodeRaw(bundle.WrappedKey.Nonce)
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}
	ciphertext, err := decodeRaw(bundle.WrappedKey.Ciphertext)
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}

	sharedSecret, err := curve25519.X25519(deviceEncryptionPrivateKey, ephemeralPub)
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}

	encKey, err := deriveEnrollmentKey(sharedSecret, deviceEnrollmentAAD(bundle.AccountID))
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}

	aad := []byte(deviceEnrollmentAAD(bundle.AccountID))
	amk, err := decryptBytes(encKey, nonce, ciphertext, aad)
	if err != nil {
		return nil, ErrDeviceEnrollmentFailed
	}
	return amk, nil
}

func generateDeviceKeyPair(random io.Reader) (DeviceKeyPair, error) {
	// Ed25519 signing key pair
	sigPub, sigPriv, err := ed25519.GenerateKey(random)
	if err != nil {
		return DeviceKeyPair{}, fmt.Errorf("generate Ed25519 key pair: %w", err)
	}

	// X25519 encryption key pair: generate a 32-byte scalar, clamp it, derive the public point
	encPriv, err := randomBytes(random, x25519KeySize)
	if err != nil {
		return DeviceKeyPair{}, fmt.Errorf("generate X25519 private key: %w", err)
	}
	// curve25519.X25519 with the base point (9) derives the public key
	basePoint := [32]byte{9}
	encPub, err := curve25519.X25519(encPriv, basePoint[:])
	if err != nil {
		return DeviceKeyPair{}, fmt.Errorf("derive X25519 public key: %w", err)
	}

	return DeviceKeyPair{
		SigningPrivateKey:    sigPriv,
		SigningPublicKey:     sigPub,
		EncryptionPrivateKey: encPriv,
		EncryptionPublicKey:  encPub,
	}, nil
}

func createDeviceEnrollmentBundle(accountID, deviceID string, deviceEncryptionPublicKey, amk []byte, random io.Reader) (domain.DeviceEnrollmentBundle, error) {
	if accountID == "" {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("accountID is required")
	}
	if deviceID == "" {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("deviceID is required")
	}
	if len(deviceEncryptionPublicKey) != x25519KeySize {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("deviceEncryptionPublicKey must be %d bytes", x25519KeySize)
	}
	if len(amk) != accountMasterKeyBytes {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("AMK must be %d bytes, got %d", accountMasterKeyBytes, len(amk))
	}

	// Generate ephemeral X25519 key pair
	ephemeralPriv, err := randomBytes(random, x25519KeySize)
	if err != nil {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("generate ephemeral key: %w", err)
	}
	basePoint := [32]byte{9}
	ephemeralPub, err := curve25519.X25519(ephemeralPriv, basePoint[:])
	if err != nil {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("derive ephemeral public key: %w", err)
	}

	// ECDH
	sharedSecret, err := curve25519.X25519(ephemeralPriv, deviceEncryptionPublicKey)
	if err != nil {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("ECDH: %w", err)
	}

	// HKDF-SHA256 → 32-byte encryption key
	encKey, err := deriveEnrollmentKey(sharedSecret, deviceEnrollmentAAD(accountID))
	if err != nil {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("derive enrollment key: %w", err)
	}

	// AES-256-GCM encrypt AMK
	aad := []byte(deviceEnrollmentAAD(accountID))
	nonce, ciphertext, err := encryptBytes(encKey, amk, aad, random)
	if err != nil {
		return domain.DeviceEnrollmentBundle{}, fmt.Errorf("encrypt AMK: %w", err)
	}

	return domain.DeviceEnrollmentBundle{
		SchemaVersion: deviceEnrollmentSchemaVersion,
		AccountID:     accountID,
		DeviceID:      deviceID,
		WrappedKey: domain.DeviceEnrollmentWrappedKey{
			Algorithm:          "x25519-hkdf-aes-256-gcm",
			EphemeralPublicKey: encodeRaw(ephemeralPub),
			Nonce:              encodeRaw(nonce),
			Ciphertext:         encodeRaw(ciphertext),
		},
	}, nil
}

// deriveEnrollmentKey runs HKDF-SHA256 over the ECDH shared secret with the AAD as info.
func deriveEnrollmentKey(sharedSecret []byte, aad string) ([]byte, error) {
	r := hkdf.New(sha256.New, sharedSecret, nil, []byte(aad))
	key := make([]byte, x25519KeySize)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

func deviceEnrollmentAAD(accountID string) string {
	return deviceEnrollmentAADPrefix + accountID
}
