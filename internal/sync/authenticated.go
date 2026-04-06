package sync

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ndelorme/safe/internal/domain"
)

type SignedAccountConfig struct {
	Version   int
	Record    domain.AccountConfigRecord
	Signature string
}

type SignedCollectionHead struct {
	Record    domain.CollectionHeadRecord `json:"record"`
	Signature string                      `json:"signature"`
}

func VerifySignedAccountConfig(candidate SignedAccountConfig, trusted *SignedAccountConfig, accountPublicKey ed25519.PublicKey) error {
	candidatePayload, err := candidate.canonicalPayload()
	if err != nil {
		return err
	}

	signature, err := decodeSignature(candidate.Signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(accountPublicKey, candidatePayload, signature) {
		return ErrMutableMetadataSignature("accountConfig")
	}

	if trusted == nil {
		return nil
	}

	if trusted.Record.AccountID != candidate.Record.AccountID {
		return ErrReplayInvariant("accountConfig.accountId")
	}
	if candidate.Version < trusted.Version {
		return ErrStaleMutableMetadata("accountConfig", trusted.Version, candidate.Version)
	}

	if candidate.Version == trusted.Version {
		trustedPayload, err := trusted.canonicalPayload()
		if err != nil {
			return err
		}
		if !bytes.Equal(trustedPayload, candidatePayload) {
			return ErrDivergentMutableMetadata("accountConfig", candidate.Version)
		}
	}

	return nil
}

func SignCollectionHead(record domain.CollectionHeadRecord, privateKey ed25519.PrivateKey) (SignedCollectionHead, error) {
	payload, err := record.CanonicalJSON()
	if err != nil {
		return SignedCollectionHead{}, err
	}
	return SignedCollectionHead{
		Record:    record,
		Signature: base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload)),
	}, nil
}

func VerifySignedCollectionHead(trusted domain.CollectionHeadRecord, candidate SignedCollectionHead, latestEvent domain.VaultEventRecord, authoringDevice domain.LocalDeviceRecord) error {
	candidatePayload, err := candidate.canonicalPayload()
	if err != nil {
		return err
	}
	if err := latestEvent.Validate(); err != nil {
		return err
	}
	if err := authoringDevice.Validate(); err != nil {
		return err
	}
	if authoringDevice.Status != "active" {
		return ErrReplayInvariant("device.status")
	}
	if latestEvent.AccountID != candidate.Record.AccountID {
		return ErrReplayInvariant("event.accountId")
	}
	if latestEvent.CollectionID != candidate.Record.CollectionID {
		return ErrReplayInvariant("event.collectionId")
	}
	if latestEvent.EventID != candidate.Record.LatestEventID {
		return ErrReplayInvariant("head.latestEventId")
	}
	if latestEvent.Sequence != candidate.Record.LatestSeq {
		return ErrHeadMismatch("latestSeq", candidate.Record.LatestSeq, latestEvent.Sequence)
	}
	if authoringDevice.AccountID != candidate.Record.AccountID {
		return ErrReplayInvariant("device.accountId")
	}
	if authoringDevice.DeviceID != latestEvent.DeviceID {
		return ErrReplayInvariant("device.deviceId")
	}

	signingPublicKey, err := decodeDeviceSigningKey(authoringDevice.SigningPublicKey)
	if err != nil {
		return err
	}
	signature, err := decodeSignature(candidate.Signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(signingPublicKey, candidatePayload, signature) {
		return ErrMutableMetadataSignature("collectionHead")
	}

	if trusted.LatestEventID == "" {
		if trusted.AccountID != "" && trusted.AccountID != candidate.Record.AccountID {
			return ErrReplayInvariant("trustedHead.accountId")
		}
		if trusted.CollectionID != "" && trusted.CollectionID != candidate.Record.CollectionID {
			return ErrReplayInvariant("trustedHead.collectionId")
		}
		if candidate.Record.LatestSeq < trusted.LatestSeq {
			return ErrStaleHead(trusted, candidate.Record)
		}
		return nil
	}

	return EnsureMonotonicHead(trusted, candidate.Record)
}

func (record SignedAccountConfig) canonicalPayload() ([]byte, error) {
	if record.Version < 1 {
		return nil, ErrReplayInvariant("accountConfig.version")
	}
	accountJSON, err := record.Record.CanonicalJSON()
	if err != nil {
		return nil, err
	}

	type canonicalSignedAccountConfig struct {
		Version int             `json:"version"`
		Record  json.RawMessage `json:"record"`
	}

	return json.Marshal(canonicalSignedAccountConfig{
		Version: record.Version,
		Record:  accountJSON,
	})
}

func (record SignedCollectionHead) canonicalPayload() ([]byte, error) {
	return record.Record.CanonicalJSON()
}

func (record SignedCollectionHead) Validate() error {
	if err := record.Record.Validate(); err != nil {
		return err
	}
	_, err := decodeSignature(record.Signature)
	return err
}

func (record SignedCollectionHead) CanonicalJSON() ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, err
	}

	headJSON, err := record.Record.CanonicalJSON()
	if err != nil {
		return nil, err
	}

	type canonicalSignedCollectionHead struct {
		Record    json.RawMessage `json:"record"`
		Signature string          `json:"signature"`
	}

	return json.Marshal(canonicalSignedCollectionHead{
		Record:    headJSON,
		Signature: record.Signature,
	})
}

func ParseSignedCollectionHeadJSON(data []byte) (SignedCollectionHead, error) {
	var record SignedCollectionHead
	if err := json.Unmarshal(data, &record); err != nil {
		return SignedCollectionHead{}, err
	}
	if err := record.Validate(); err != nil {
		return SignedCollectionHead{}, err
	}
	return record, nil
}

func decodeSignature(signature string) ([]byte, error) {
	if signature == "" {
		return nil, ErrUnsignedMutableMetadata("signature")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return nil, ErrUnsignedMutableMetadata("signature")
	}
	if len(decoded) != ed25519.SignatureSize {
		return nil, ErrUnsignedMutableMetadata("signature")
	}
	return decoded, nil
}

func decodeDeviceSigningKey(value string) (ed25519.PublicKey, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, ErrReplayInvariant("device.signingPublicKey")
	}
	if len(decoded) != ed25519.PublicKeySize {
		return nil, ErrReplayInvariant("device.signingPublicKey")
	}
	return ed25519.PublicKey(decoded), nil
}

type unsignedMutableMetadataError string

func (err unsignedMutableMetadataError) Error() string {
	return fmt.Sprintf("unsigned mutable metadata rejected: %s", string(err))
}

func ErrUnsignedMutableMetadata(field string) error {
	return unsignedMutableMetadataError(field)
}

type mutableMetadataSignatureError string

func (err mutableMetadataSignatureError) Error() string {
	return fmt.Sprintf("mutable metadata signature verification failed: %s", string(err))
}

func ErrMutableMetadataSignature(family string) error {
	return mutableMetadataSignatureError(family)
}

type staleMutableMetadataError struct {
	family    string
	trusted   int
	candidate int
}

func (err staleMutableMetadataError) Error() string {
	return fmt.Sprintf("stale mutable metadata rejected: %s trusted %d candidate %d", err.family, err.trusted, err.candidate)
}

func ErrStaleMutableMetadata(family string, trusted, candidate int) error {
	return staleMutableMetadataError{family: family, trusted: trusted, candidate: candidate}
}

type divergentMutableMetadataError struct {
	family  string
	version int
}

func (err divergentMutableMetadataError) Error() string {
	return fmt.Sprintf("divergent mutable metadata rejected: %s version %d", err.family, err.version)
}

func ErrDivergentMutableMetadata(family string, version int) error {
	return divergentMutableMetadataError{family: family, version: version}
}
