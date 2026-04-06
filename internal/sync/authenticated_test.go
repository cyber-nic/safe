package sync

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	internalcrypto "github.com/ndelorme/safe/internal/crypto"
	"github.com/ndelorme/safe/internal/domain"
)

func TestVerifySignedAccountConfigAcceptsValidSignature(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fatalf(t, "generate key: %v", err)
	}

	candidate := mustSignAccountConfig(t, privateKey, domain.StarterAccountConfigRecord(), 2)

	if err := VerifySignedAccountConfig(candidate, nil, publicKey); err != nil {
		t.Fatalf("verify signed account config: %v", err)
	}
}

func TestVerifySignedAccountConfigRejectsUnsignedMetadata(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	candidate := SignedAccountConfig{Version: 1, Record: domain.StarterAccountConfigRecord()}
	if err := VerifySignedAccountConfig(candidate, nil, publicKey); err == nil {
		t.Fatal("expected unsigned metadata rejection")
	}
}

func TestVerifySignedAccountConfigRejectsRollback(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	trusted := mustSignAccountConfig(t, privateKey, domain.StarterAccountConfigRecord(), 3)
	candidate := mustSignAccountConfig(t, privateKey, domain.StarterAccountConfigRecord(), 2)

	if err := VerifySignedAccountConfig(candidate, &trusted, publicKey); err == nil {
		t.Fatal("expected stale metadata rejection")
	}
}

func TestVerifySignedAccountConfigRejectsDivergenceAtSameVersion(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	trusted := mustSignAccountConfig(t, privateKey, domain.StarterAccountConfigRecord(), 4)
	candidateRecord := domain.StarterAccountConfigRecord()
	candidateRecord.DefaultCollectionID = "vault-rotated"
	candidateRecord.CollectionIDs = []string{"vault-personal", "vault-rotated"}
	candidate := mustSignAccountConfig(t, privateKey, candidateRecord, 4)

	if err := VerifySignedAccountConfig(candidate, &trusted, publicKey); err == nil {
		t.Fatal("expected divergence rejection")
	}
}

func TestVerifySignedCollectionHeadAcceptsValidSignature(t *testing.T) {
	keyPair, err := internalcrypto.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}

	event := domain.StarterVaultEventRecords()[1]
	head := domain.StarterCollectionHeadRecord()
	deviceRecord, err := internalcrypto.CreateDeviceRecord(event.AccountID, event.DeviceID, "Primary laptop", "cli", keyPair)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}

	candidate := mustSignCollectionHead(t, keyPair.SigningPrivateKey, head)
	if err := VerifySignedCollectionHead(domain.CollectionHeadRecord{}, candidate, event, deviceRecord); err != nil {
		t.Fatalf("verify signed collection head: %v", err)
	}
}

func TestVerifySignedCollectionHeadRejectsWrongAuthoringDevice(t *testing.T) {
	keyPair, err := internalcrypto.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}

	event := domain.StarterVaultEventRecords()[1]
	head := domain.StarterCollectionHeadRecord()
	deviceRecord, err := internalcrypto.CreateDeviceRecord(event.AccountID, "dev-other-001", "Secondary laptop", "cli", keyPair)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}

	candidate := mustSignCollectionHead(t, keyPair.SigningPrivateKey, head)
	if err := VerifySignedCollectionHead(domain.CollectionHeadRecord{}, candidate, event, deviceRecord); err == nil {
		t.Fatal("expected authoring-device binding rejection")
	}
}

func TestVerifySignedCollectionHeadRejectsBadSignature(t *testing.T) {
	keyPair, err := internalcrypto.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}

	otherPublic, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate unrelated key: %v", err)
	}

	event := domain.StarterVaultEventRecords()[1]
	head := domain.StarterCollectionHeadRecord()
	deviceRecord, err := internalcrypto.CreateDeviceRecord(event.AccountID, event.DeviceID, "Primary laptop", "cli", keyPair)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}
	deviceRecord.SigningPublicKey = base64.RawURLEncoding.EncodeToString(otherPublic)

	candidate := mustSignCollectionHead(t, keyPair.SigningPrivateKey, head)
	if err := VerifySignedCollectionHead(domain.CollectionHeadRecord{}, candidate, event, deviceRecord); err == nil {
		t.Fatal("expected signature failure")
	}
}

func TestVerifySignedCollectionHeadRejectsRollback(t *testing.T) {
	keyPair, err := internalcrypto.GenerateDeviceKeyPair()
	if err != nil {
		t.Fatalf("generate device key pair: %v", err)
	}

	events := domain.StarterVaultEventRecords()
	event := events[1]
	trustedHead := domain.CollectionHeadRecord{
		SchemaVersion: 1,
		AccountID:     event.AccountID,
		CollectionID:  event.CollectionID,
		LatestEventID: "evt-login-github-primary-v3",
		LatestSeq:     3,
	}
	deviceRecord, err := internalcrypto.CreateDeviceRecord(event.AccountID, event.DeviceID, "Primary laptop", "cli", keyPair)
	if err != nil {
		t.Fatalf("create device record: %v", err)
	}

	candidate := mustSignCollectionHead(t, keyPair.SigningPrivateKey, domain.StarterCollectionHeadRecord())
	if err := VerifySignedCollectionHead(trustedHead, candidate, event, deviceRecord); err == nil {
		t.Fatal("expected stale head rejection")
	}
}

func mustSignAccountConfig(t *testing.T, privateKey ed25519.PrivateKey, record domain.AccountConfigRecord, version int) SignedAccountConfig {
	t.Helper()

	signed := SignedAccountConfig{Version: version, Record: record}
	payload, err := signed.canonicalPayload()
	if err != nil {
		t.Fatalf("canonical signed account config: %v", err)
	}
	signed.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	return signed
}

func mustSignCollectionHead(t *testing.T, privateKey ed25519.PrivateKey, record domain.CollectionHeadRecord) SignedCollectionHead {
	t.Helper()

	signed, err := SignCollectionHead(record, privateKey)
	if err != nil {
		t.Fatalf("sign collection head: %v", err)
	}
	return signed
}

func fatalf(t *testing.T, format string, args ...any) {
	t.Helper()
	t.Fatalf(format, args...)
}
