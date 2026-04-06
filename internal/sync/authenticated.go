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
	Record    domain.CollectionHeadRecord
	Signature string
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

func VerifySignedCollectionHead(candidate SignedCollectionHead, trusted *SignedCollectionHead, latestEvent domain.VaultEventRecord, authoringDevice domain.LocalDeviceRecord) error {
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

	if trusted == nil {
		return nil
	}
	return EnsureMonotonicHead(trusted.Record, candidate.Record)
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
