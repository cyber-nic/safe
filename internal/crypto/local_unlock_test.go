package crypto

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestCreateAndOpenLocalUnlockRecord(t *testing.T) {
	record, accountKey, err := CreateLocalUnlockRecord("acct-dev-001", "correct horse battery staple")
	if err != nil {
		t.Fatalf("create local unlock record: %v", err)
	}

	reopenedKey, err := OpenLocalUnlockRecord(record, "correct horse battery staple")
	if err != nil {
		t.Fatalf("open local unlock record: %v", err)
	}

	if string(reopenedKey) != string(accountKey) {
		t.Fatal("expected reopened account key to match created key")
	}
}

func TestOpenLocalUnlockRecordRejectsWrongPassword(t *testing.T) {
	record, _, err := CreateLocalUnlockRecord("acct-dev-001", "correct horse battery staple")
	if err != nil {
		t.Fatalf("create local unlock record: %v", err)
	}

	_, err = OpenLocalUnlockRecord(record, "wrong password")
	if !errors.Is(err, ErrUnlockFailed) {
		t.Fatalf("expected ErrUnlockFailed, got %v", err)
	}
}

func TestOpenLocalUnlockRecordRejectsCorruptedPayload(t *testing.T) {
	record, _, err := CreateLocalUnlockRecord("acct-dev-001", "correct horse battery staple")
	if err != nil {
		t.Fatalf("create local unlock record: %v", err)
	}

	record.WrappedKey.Ciphertext = record.WrappedKey.Ciphertext[:len(record.WrappedKey.Ciphertext)-2] + "xx"

	_, err = OpenLocalUnlockRecord(record, "correct horse battery staple")
	if !errors.Is(err, ErrUnlockFailed) {
		t.Fatalf("expected ErrUnlockFailed, got %v", err)
	}
}

func TestEncryptAndDecryptSecretMaterial(t *testing.T) {
	_, accountKey, err := CreateLocalUnlockRecord("acct-dev-001", "correct horse battery staple")
	if err != nil {
		t.Fatalf("create local unlock record: %v", err)
	}

	payload, err := EncryptSecretMaterial(accountKey, []byte("vault-secret-value"))
	if err != nil {
		t.Fatalf("encrypt secret material: %v", err)
	}

	plaintext, err := DecryptSecretMaterial(accountKey, payload)
	if err != nil {
		t.Fatalf("decrypt secret material: %v", err)
	}

	if string(plaintext) != "vault-secret-value" {
		t.Fatalf("unexpected plaintext: %s", plaintext)
	}
}

func TestDecryptSecretMaterialRejectsCorruptedPayload(t *testing.T) {
	_, accountKey, err := CreateLocalUnlockRecord("acct-dev-001", "correct horse battery staple")
	if err != nil {
		t.Fatalf("create local unlock record: %v", err)
	}

	payload, err := EncryptSecretMaterial(accountKey, []byte("vault-secret-value"))
	if err != nil {
		t.Fatalf("encrypt secret material: %v", err)
	}

	var envelope SecretMaterialEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("unmarshal secret material envelope: %v", err)
	}
	envelope.Ciphertext = envelope.Ciphertext[:len(envelope.Ciphertext)-2] + "xx"

	corruptedPayload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal corrupted secret material envelope: %v", err)
	}

	_, err = DecryptSecretMaterial(accountKey, corruptedPayload)
	if !errors.Is(err, ErrSecretMaterialDecrypt) {
		t.Fatalf("expected ErrSecretMaterialDecrypt, got %v", err)
	}
}

func TestSecretMaterialEnvelopeValidateRejectsInvalidShape(t *testing.T) {
	envelope := SecretMaterialEnvelope{
		SchemaVersion: 1,
		Algorithm:     "aes-256-gcm",
		Nonce:         "bad-base64***",
		Ciphertext:    "still-bad***",
	}

	if err := envelope.Validate(); err == nil {
		t.Fatal("expected invalid secret material envelope")
	}
}
