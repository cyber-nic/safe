package crypto

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/ndelorme/safe/internal/domain"
)

func TestGenerateDeviceID(t *testing.T) {
	id, err := GenerateDeviceID()
	if err != nil {
		t.Fatalf("generate device ID: %v", err)
	}
	if len(id) != 32 {
		t.Fatalf("expected 32-character device ID, got %d characters", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Fatalf("device ID is not valid hex: %v", err)
	}
}

func TestGenerateDeviceKeyPair(t *testing.T) {
	kp, err := GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}
	if len(kp.SigningPublicKey) != 32 {
		t.Fatalf("signing public key must be 32 bytes, got %d", len(kp.SigningPublicKey))
	}
	if len(kp.SigningPrivateKey) != 64 {
		t.Fatalf("signing private key must be 64 bytes, got %d", len(kp.SigningPrivateKey))
	}
	if len(kp.EncryptionPublicKey) != 32 {
		t.Fatalf("encryption public key must be 32 bytes, got %d", len(kp.EncryptionPublicKey))
	}
	if len(kp.EncryptionPrivateKey) != 32 {
		t.Fatalf("encryption private key must be 32 bytes, got %d", len(kp.EncryptionPrivateKey))
	}
}

func TestCreateDeviceRecord(t *testing.T) {
	kp, err := GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	rec, err := CreateDeviceRecord("acct-dev-001", "aabbccdd", "My CLI", "cli", kp)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}
	if err := rec.Validate(); err != nil {
		t.Fatalf("device record invalid: %v", err)
	}
}

func TestCreateAndOpenDeviceEnrollmentBundle(t *testing.T) {
	newKP, err := GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate new device key pair: %v", err)
	}

	amk := make([]byte, 32)
	for i := range amk {
		amk[i] = byte(i + 0x80)
	}

	deviceID := "deadbeefdeadbeefdeadbeefdeadbeef"

	bundle, err := CreateDeviceEnrollmentBundle("acct-dev-001", deviceID, newKP.EncryptionPublicKey, amk)
	if err != nil {
		t.Fatalf("create enrollment bundle: %v", err)
	}
	if err := bundle.Validate(); err != nil {
		t.Fatalf("enrollment bundle invalid: %v", err)
	}

	unwrappedAMK, err := OpenDeviceEnrollmentBundle(bundle, deviceID, newKP.EncryptionPrivateKey)
	if err != nil {
		t.Fatalf("open enrollment bundle: %v", err)
	}
	if !bytes.Equal(unwrappedAMK, amk) {
		t.Fatal("unwrapped AMK does not match original")
	}
}

func TestOpenDeviceEnrollmentBundleRejectsWrongPrivateKey(t *testing.T) {
	newKP, _ := GenerateDeviceKeyPair()
	wrongKP, _ := GenerateDeviceKeyPair()

	amk := make([]byte, 32)
	deviceID := "deadbeefdeadbeefdeadbeefdeadbeef"

	bundle, err := CreateDeviceEnrollmentBundle("acct-dev-001", deviceID, newKP.EncryptionPublicKey, amk)
	if err != nil {
		t.Fatalf("create enrollment bundle: %v", err)
	}

	_, err = OpenDeviceEnrollmentBundle(bundle, deviceID, wrongKP.EncryptionPrivateKey)
	if !errors.Is(err, ErrDeviceEnrollmentFailed) {
		t.Fatalf("expected ErrDeviceEnrollmentFailed for wrong key, got %v", err)
	}
}

func TestOpenDeviceEnrollmentBundleRejectsCorruptedCiphertext(t *testing.T) {
	newKP, _ := GenerateDeviceKeyPair()

	amk := make([]byte, 32)
	deviceID := "deadbeefdeadbeefdeadbeefdeadbeef"

	bundle, err := CreateDeviceEnrollmentBundle("acct-dev-001", deviceID, newKP.EncryptionPublicKey, amk)
	if err != nil {
		t.Fatalf("create enrollment bundle: %v", err)
	}

	// Flip the last two characters of the base64url ciphertext.
	ct := bundle.WrappedKey.Ciphertext
	bundle.WrappedKey.Ciphertext = ct[:len(ct)-2] + flipBase64Pair(ct[len(ct)-2:])

	_, err = OpenDeviceEnrollmentBundle(bundle, deviceID, newKP.EncryptionPrivateKey)
	if !errors.Is(err, ErrDeviceEnrollmentFailed) {
		t.Fatalf("expected ErrDeviceEnrollmentFailed for corrupted ciphertext, got %v", err)
	}
}

func TestOpenDeviceEnrollmentBundleRejectsWrongAccountID(t *testing.T) {
	newKP, _ := GenerateDeviceKeyPair()

	amk := make([]byte, 32)
	deviceID := "deadbeefdeadbeefdeadbeefdeadbeef"

	bundle, err := CreateDeviceEnrollmentBundle("acct-dev-001", deviceID, newKP.EncryptionPublicKey, amk)
	if err != nil {
		t.Fatalf("create enrollment bundle: %v", err)
	}

	// Transplant the bundle to a different account — AAD mismatch must cause auth failure.
	bundle.AccountID = "acct-attacker-002"

	_, err = OpenDeviceEnrollmentBundle(bundle, deviceID, newKP.EncryptionPrivateKey)
	if !errors.Is(err, ErrDeviceEnrollmentFailed) {
		t.Fatalf("expected ErrDeviceEnrollmentFailed for wrong account ID, got %v", err)
	}
}

func TestOpenDeviceEnrollmentBundleRejectsWrongDeviceID(t *testing.T) {
	newKP, _ := GenerateDeviceKeyPair()

	amk := make([]byte, 32)
	deviceID := "deadbeefdeadbeefdeadbeefdeadbeef"

	bundle, err := CreateDeviceEnrollmentBundle("acct-dev-001", deviceID, newKP.EncryptionPublicKey, amk)
	if err != nil {
		t.Fatalf("create enrollment bundle: %v", err)
	}

	_, err = OpenDeviceEnrollmentBundle(bundle, "ffffffffffffffffffffffffffffffff", newKP.EncryptionPrivateKey)
	if !errors.Is(err, ErrDeviceEnrollmentFailed) {
		t.Fatalf("expected ErrDeviceEnrollmentFailed for wrong device ID, got %v", err)
	}
}

func TestOpenDeviceEnrollmentBundleFromOnDiskFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/device_enrollment_fixture.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var f struct {
		DeviceEncPrivKeyHex string                        `json:"deviceEncPrivKeyHex"`
		DeviceID            string                        `json:"deviceId"`
		AMKHex              string                        `json:"amkHex"`
		Bundle              domain.DeviceEnrollmentBundle `json:"bundle"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	deviceEncPrivKey, err := hex.DecodeString(f.DeviceEncPrivKeyHex)
	if err != nil {
		t.Fatalf("decode fixture device encryption private key: %v", err)
	}
	expectedAMK, err := hex.DecodeString(f.AMKHex)
	if err != nil {
		t.Fatalf("decode fixture AMK: %v", err)
	}

	amk, err := OpenDeviceEnrollmentBundle(f.Bundle, f.DeviceID, deviceEncPrivKey)
	if err != nil {
		t.Fatalf("open fixture enrollment bundle: %v", err)
	}
	if !bytes.Equal(amk, expectedAMK) {
		t.Fatalf("fixture AMK mismatch: got %x, want %x", amk, expectedAMK)
	}
}

// TestRecoveryKeyBootstrapPath verifies that the recovery-key bootstrap path reuses
// OpenLocalRecoveryRecord (W8) to produce the AMK on a new device. No new crypto is
// exercised here; the test proves the wiring is correct.
func TestRecoveryKeyBootstrapPath(t *testing.T) {
	recoveryKey, err := GenerateRecoveryKey()
	if err != nil {
		t.Fatalf("generate recovery key: %v", err)
	}

	// Simulate first-use: create unlock + recovery records for an account.
	_, _, amk := mustUnlockRecord(t)
	recoveryRecord, _, err := CreateLocalRecoveryRecord("acct-dev-001", recoveryKey, amk)
	if err != nil {
		t.Fatalf("create recovery record: %v", err)
	}

	// New device: recover AMK using recovery key (recovery-key bootstrap path).
	recoveredAMK, err := OpenLocalRecoveryRecord(recoveryRecord, recoveryKey)
	if err != nil {
		t.Fatalf("recovery bootstrap: %v", err)
	}
	if !bytes.Equal(recoveredAMK, amk) {
		t.Fatal("recovery bootstrap AMK does not match original")
	}

	// New device can now create its device record with a fresh key pair.
	deviceID, err := GenerateDeviceID()
	if err != nil {
		t.Fatalf("generate device ID: %v", err)
	}
	kp, err := GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate key pair: %v", err)
	}
	rec, err := CreateDeviceRecord("acct-dev-001", deviceID, "Recovered Device", "cli", kp)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}
	if err := rec.Validate(); err != nil {
		t.Fatalf("device record invalid after recovery bootstrap: %v", err)
	}
}
