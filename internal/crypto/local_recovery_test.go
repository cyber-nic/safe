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

func TestCreateAndOpenLocalRecoveryRecord(t *testing.T) {
	recoveryKey, err := GenerateRecoveryKey()
	if err != nil {
		t.Fatalf("generate recovery key: %v", err)
	}

	_, _, amk := mustUnlockRecord(t)

	record, mnemonic, err := CreateLocalRecoveryRecord("acct-dev-001", recoveryKey, amk)
	if err != nil {
		t.Fatalf("create recovery record: %v", err)
	}
	if mnemonic == "" {
		t.Fatal("expected non-empty mnemonic")
	}

	reopenedAMK, err := OpenLocalRecoveryRecord(record, recoveryKey)
	if err != nil {
		t.Fatalf("open recovery record: %v", err)
	}

	if !bytes.Equal(reopenedAMK, amk) {
		t.Fatal("reopened AMK does not match original")
	}
}

func TestOpenLocalRecoveryRecordRejectsWrongKey(t *testing.T) {
	recoveryKey, _ := GenerateRecoveryKey()
	_, _, amk := mustUnlockRecord(t)

	record, _, err := CreateLocalRecoveryRecord("acct-dev-001", recoveryKey, amk)
	if err != nil {
		t.Fatalf("create recovery record: %v", err)
	}

	wrongKey := make([]byte, 32)
	copy(wrongKey, recoveryKey)
	wrongKey[0] ^= 0xff

	_, err = OpenLocalRecoveryRecord(record, wrongKey)
	if !errors.Is(err, ErrRecoveryFailed) {
		t.Fatalf("expected ErrRecoveryFailed, got %v", err)
	}
}

func TestOpenLocalRecoveryRecordRejectsCorruptedCiphertext(t *testing.T) {
	recoveryKey, _ := GenerateRecoveryKey()
	_, _, amk := mustUnlockRecord(t)

	record, _, err := CreateLocalRecoveryRecord("acct-dev-001", recoveryKey, amk)
	if err != nil {
		t.Fatalf("create recovery record: %v", err)
	}

	// Flip the last two characters of the base64url ciphertext.
	ct := record.WrappedKey.Ciphertext
	record.WrappedKey.Ciphertext = ct[:len(ct)-2] + flipBase64Pair(ct[len(ct)-2:])

	_, err = OpenLocalRecoveryRecord(record, recoveryKey)
	if !errors.Is(err, ErrRecoveryFailed) {
		t.Fatalf("expected ErrRecoveryFailed, got %v", err)
	}
}

func TestOpenLocalRecoveryRecordRejectsWrongAccountID(t *testing.T) {
	recoveryKey, _ := GenerateRecoveryKey()
	_, _, amk := mustUnlockRecord(t)

	record, _, err := CreateLocalRecoveryRecord("acct-dev-001", recoveryKey, amk)
	if err != nil {
		t.Fatalf("create recovery record: %v", err)
	}

	// Transplant the record to a different account ID.
	// The AAD includes accountId so decryption must fail before any plaintext is returned.
	record.AccountID = "acct-attacker-002"

	_, err = OpenLocalRecoveryRecord(record, recoveryKey)
	if !errors.Is(err, ErrRecoveryFailed) {
		t.Fatalf("expected ErrRecoveryFailed for wrong account ID, got %v", err)
	}
}

func TestOpenLocalRecoveryRecordFromOnDiskFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/recovery_fixture.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var f struct {
		RecoveryKeyHex string                    `json:"recoveryKeyHex"`
		AMKHex         string                    `json:"amkHex"`
		Mnemonic       string                    `json:"mnemonic"`
		Record         domain.LocalRecoveryRecord `json:"record"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	recoveryKey, err := hex.DecodeString(f.RecoveryKeyHex)
	if err != nil {
		t.Fatalf("decode fixture recovery key: %v", err)
	}
	expectedAMK, err := hex.DecodeString(f.AMKHex)
	if err != nil {
		t.Fatalf("decode fixture AMK: %v", err)
	}

	amk, err := OpenLocalRecoveryRecord(f.Record, recoveryKey)
	if err != nil {
		t.Fatalf("open fixture recovery record: %v", err)
	}

	if !bytes.Equal(amk, expectedAMK) {
		t.Fatalf("fixture AMK mismatch: got %x, want %x", amk, expectedAMK)
	}

	// Verify the mnemonic encodes the same recovery key bytes.
	mnemonic, err := RecoveryKeyMnemonic(recoveryKey)
	if err != nil {
		t.Fatalf("encode mnemonic: %v", err)
	}
	if mnemonic != f.Mnemonic {
		t.Fatalf("fixture mnemonic mismatch:\n  got  %s\n  want %s", mnemonic, f.Mnemonic)
	}
}

// mustUnlockRecord is a test helper that creates an unlock record and returns
// the accountID, record, and account master key.
func mustUnlockRecord(t *testing.T) (string, domain.LocalUnlockRecord, []byte) {
	t.Helper()
	accountID := "acct-dev-001"
	record, amk, err := CreateLocalUnlockRecord(accountID, "correct horse battery staple")
	if err != nil {
		t.Fatalf("create unlock record: %v", err)
	}
	return accountID, record, amk
}

// flipBase64Pair returns a two-character base64url string that differs from the input.
func flipBase64Pair(s string) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	b := []byte(s)
	for i := range b {
		for _, c := range []byte(alphabet) {
			if c != b[i] {
				b[i] = c
				break
			}
		}
	}
	return string(b)
}
