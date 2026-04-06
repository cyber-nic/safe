# Safe Interfaces

## Purpose

This file defines the current shared contracts between workstreams.

Related planning artifacts:

- `docs/project/WORKBOARD.md`
- `docs/project/HANDOFFS.md`
- `docs/project/GITHUB_PROJECTS.md`

If you need to change one of these contracts:

1. update this file first
2. note the change in `docs/project/HANDOFFS.md`
3. comment on the matching GitHub issue
4. then update consumers

## I1 - Durable Local Persistence Contract

Status:

- accepted

Owner:

- Engineer1

Consumers:

- Engineer2 implementation
- Engineer1 CLI integration
- Engineer3 web integration later

Goals:

- persist local vault state across process restart
- support an encrypted local runtime later without rewriting all persistence semantics
- keep the first implementation small
- preserve the current record model already used by `internal/storage` and `cmd/safe`

Required persisted units:

- account config record
- collection head record
- vault item records
- vault event records
- secret material records

Current account-local key layout:

- `accounts/<accountID>/account.json` for the account config record
- `accounts/<accountID>/collections/<collectionID>/head.json` for the collection head record
- `accounts/<accountID>/collections/<collectionID>/items/<itemID>.json` for item records
- `accounts/<accountID>/collections/<collectionID>/events/<eventID>.json` for event records
- `accounts/<accountID>/collections/<collectionID>/secrets/<base64url(secretRef)>.txt` for secret material

Contract rules:

- the first durable adapter is account-local and backend-specific, but it must preserve the logical key layout above
- higher-level consumers must not depend on the physical backend being memory, files, or SQLite
- the adapter must not synthesize starter fixtures; initialization creates empty durable state only
- the adapter must preserve canonical record bytes for account config, collection head, item records, and event records
- secret material is stored as opaque bytes from the persistence layer perspective; higher-level code must not assume plaintext storage forever
- additional account-scoped records needed for unlock metadata may be added under `accounts/<accountID>/` without changing collection paths

Required behaviors:

- initialize an empty account-local store without starter data
- write one or more records durably
- load all records needed to rebuild current state
- return event records in deterministic ascending `sequence` order, with a stable `eventId` tie-break if needed
- support a single logical mutation commit for the save or update path:
  optional secret material write, optional item record write, event record write, and collection head update must become durable together or fail without exposing a partial new head
- survive process recreation in tests

Constraints:

- no network dependency
- no control-plane dependency
- no implicit starter fixtures
- no plaintext format assumptions in higher-level consumers
- no consumer dependence on backend filename ordering

Notes:

- the first persistence adapter does not need to solve final encryption by itself
- it must not make future encryption layering awkward
- `cmd/safe` currently violates this contract by recreating a fresh `MemoryObjectStore` plus starter records on each run; W4 is the consumer task that will switch to this contract

## I2 - Local Unlock Contract

Status:

- accepted

Owner:

- Engineer1

Goals:

- derive a local key from user password
- open encrypted local runtime state
- fail safely on wrong password or corrupted payload

Contract boundary:

- password-derived unlock metadata is account-scoped, not collection-scoped
- the unlock flow owns crypto envelope parsing and key derivation; the persistence adapter only stores and loads the required durable bytes
- the unlock flow must be able to open the same durable account-local store after process restart without replaying starter fixtures
- the unlock flow is the only supported path to decrypted secret material once W3 and W4 land

Required behaviors:

- create new local unlock state for a first-use account
- reopen existing local state with the same password
- reject wrong password
- reject corrupted ciphertext or metadata

Frozen for W3 (`refs #5`):

- unlock metadata is stored at `accounts/<accountID>/unlock.json`
- unlock record schema is `LocalUnlockRecord` with:
  `schemaVersion`, `accountId`, `kdf`, and `wrappedKey`
- `kdf` fields are:
  `name=argon2id`, `salt`, `memoryKiB`, `timeCost`, `parallelism`, and `keyBytes`
- `wrappedKey` fields are:
  `algorithm=aes-256-gcm`, `nonce`, and `ciphertext`
- the unlock flow derives a 32-byte KEK from the user password with Argon2id and unwraps a random 32-byte account master key via AES-256-GCM
- local secret material is stored as a versioned JSON envelope containing:
  `schemaVersion`, `algorithm`, `nonce`, and `ciphertext`
- both the unlock record and the secret-material envelope use base64url encoding without padding for binary fields
- wrong-password and corrupted-payload failures may share the same authentication-failure result; callers must reject both without exposing plaintext

Still not defined:

- password rotation flow

Frozen for W7 (`refs #20`):

- see I6 below for the recovery-key contract

## I3 - CLI Integration Contract

Status:

- accepted

Owner:

- Engineer1

Current gap:

- `cmd/safe` currently constructs a fresh `MemoryObjectStore` and starter data on each run

Target:

- main save/read path uses durable local persistence plus real unlock flow

Rules:

- CLI wiring should consume the local runtime contract, not bypass it
- legacy starter bootstrap may remain only behind clearly marked dev-only paths while migration is in progress
- normal CLI bootstrap must hydrate account config, head state, items, events, and secret material from the durable adapter
- the first-use CLI path may create empty durable state plus unlock metadata, but it must not silently repopulate the store with sample records

## I4 - Web Integration Contract

Status:

- draft

Owner:

- Engineer1

Current gap:

- W10 exposed the web runtime through a local server-rendered client surface in `apps/web`
- browser-native storage adapter details still remain thinner than the CLI path
- future browser-native work should refine that adapter without reopening the M1 surface decision

Target:

- web surface consumes the same local-runtime concepts as the CLI
- M1 is satisfied by the shipped local web client surface; a browser-specific adapter remains follow-up work if the web target moves fully in-browser

Rules:

- do not invent a separate unlock model in `apps/web`
- persisted workspace snapshots are transitional and should not become the final runtime API
- runtime persistence should prefer account config, collection head, replayable events, and opaque secret-material boundaries over derived UI state
- do not treat browser-native adapter polish as grounds to reopen M1 once the real client surface is shipped

## I6 - Recovery Key Contract

Status:

- accepted

Owner:

- Engineer2 (Claude)

Frozen for W7 (`refs #20`):

Goals:

- allow an account to be unlocked on a new device without the master password
- derive no key material from the recovery key via a KDF; the recovery key itself is the KEK for the AMK wrap
- keep the recovery record schema aligned with the unlock record schema from I2

### Recovery Key Format

- a recovery key is 32 random bytes generated on the client at first-use time
- it is encoded for human display and backup as a 24-word BIP-39 mnemonic (256-bit entropy → 24 words with checksum)
- the raw 32 bytes are the input key material; no KDF is applied because the entropy is already sufficient
- callers must display the mnemonic exactly once during account creation and require explicit acknowledgement before proceeding

### Storage Path

- recovery metadata is stored at `accounts/<accountID>/recovery.json`
- this is an account-scoped path, parallel to `accounts/<accountID>/unlock.json`
- the persistence adapter stores and loads the recovery record as opaque bytes; it does not parse the crypto payload

### Recovery Record Schema

The record is named `LocalRecoveryRecord` and contains:

```json
{
  "schemaVersion": 1,
  "accountId": "<accountID>",
  "wrappedKey": {
    "algorithm": "aes-256-gcm",
    "nonce": "<base64url-no-pad>",
    "ciphertext": "<base64url-no-pad>"
  }
}
```

Fields:

- `schemaVersion`: integer, must be `1`
- `accountId`: the account ID this record belongs to
- `wrappedKey.algorithm`: must be `"aes-256-gcm"`
- `wrappedKey.nonce`: base64url without padding, 12 random bytes
- `wrappedKey.ciphertext`: base64url without padding, the AES-256-GCM encryption of the 32-byte AMK using the raw recovery key bytes as the key

### AAD

- the authenticated additional data for the wrappedKey envelope is the fixed ASCII string `"safe.local-recovery.v1/"` concatenated with the `accountId`
- this binds the ciphertext to its account and prevents cross-account record transplant

### Wrap and Unwrap Rules

- wrap: `AES-256-GCM.Seal(key=recoveryKeyBytes, nonce=random12, plaintext=AMK, aad=AAD)`
- unwrap: `AES-256-GCM.Open(key=recoveryKeyBytes, nonce=record.wrappedKey.nonce, ciphertext=record.wrappedKey.ciphertext, aad=AAD)`
- a wrong recovery key or corrupted ciphertext must return an authentication-failure error and must never expose partial plaintext
- wrong-key and corrupted-ciphertext failures may share the same error value; callers must not distinguish them

### First-Use Lifecycle

- `CreateLocalRecoveryRecord(accountID string, recoveryKeyBytes []byte, AMK []byte)` creates the record and returns the mnemonic for display
- the record is persisted to the durable store via `StoreLocalRecoveryRecord` immediately after creation
- `OpenLocalRecoveryRecord(record LocalRecoveryRecord, recoveryKeyBytes []byte)` returns the unwrapped AMK
- both functions must be tested against serialized on-disk fixtures, not only in-memory state

### Relationship to the Password Unlock Path (I2)

- both I2 and I6 wrap the same AMK; they are parallel unlock paths for the same account key
- the recovery record does not replace the unlock record; both must be present after account creation
- callers that have the AMK from either path proceed identically; no caller should branch on which path produced the AMK

### Negative Tests Required by W8

- wrong recovery key → authentication-failure error, no plaintext
- corrupted ciphertext (bit-flipped) → authentication-failure error, no plaintext
- recovery record for wrong account ID → validation error before any decryption attempt
- recovery succeeds against a serialized on-disk fixture (not only in-memory helpers)

### Not in Scope for W7 or W8

- mnemonic validation or checksum checking during input (UX concern for a later slice)
- recovery key rotation
- server-assisted recovery escrow
- device enrollment via recovery key (separate protocol slice)

## I7 - Signed Mutable Metadata and Rollback Contract

Status:

- accepted

Owner:

- Engineer1

Frozen for W9 (`refs #18`):

Goals:

- freeze which mutable records are security-relevant trust anchors
- define who authenticates each mutable record family
- require freshness and rollback checks before a client trusts mutable state from storage or the control plane

Mutable record families in scope:

- account config
- collection head pointers
- snapshot pointers
- device metadata
- membership state
- invite state

Out of scope for I7:

- exact signature envelopes for mutable metadata families other than the persisted collection-head path frozen below
- control-plane storage schema
- share-protocol implementation details beyond the trust checks consumers must enforce

Frozen for W12 (`refs #36`):

- sync stores the mutable collection head at `accounts/<accountID>/collections/<collectionID>/head.json` as `{"record": <CollectionHeadRecord>, "signature": <base64url-no-pad Ed25519 signature>}`
- the signature covers the canonical JSON bytes of `record` only
- sync readers and writers must load the signed head, the referenced latest event, and the active authoring device record before trusting or advancing the head

### Canonicalization and Authentication

- every record family in scope must have canonical bytes before hashing or signing
- clients must authenticate canonical bytes, not transport formatting, pretty-printed JSON, or field order as received
- unsigned mutable metadata is invalid; clients must reject it rather than downgrade to best-effort replay
- signature or MAC verification failures are hard failures, not telemetry-only warnings

### Signer Ownership

- account config is authenticated by account-level trust material
- collection head pointers are authenticated by the device signing key that authored the committed event referenced by the head
- snapshot pointers are authenticated by account-level trust material and must bind the referenced snapshot object plus the base event cursor
- device metadata is authenticated by account-level trust material; device self-assertion alone is insufficient
- membership state is authenticated by account-level trust material for collection-owner decisions
- invite state is authenticated by account-level trust material for lifecycle transitions and must bind the intended recipient account and recipient device key when accepted

### Required Binding Fields

- every mutable record must bind `accountId`
- collection-scoped mutable records must bind `collectionId`
- device-scoped mutable records must bind `deviceId`
- head pointers must bind `latestEventId` and the monotonic event cursor or sequence they advance to
- snapshot pointers must bind the snapshot object identifier plus the event cursor the snapshot represents
- membership and invite records must bind the member or recipient account identity and any recipient device key material they authorize

### Freshness and Rollback Rules

- clients must persist the highest trusted version of each mutable record family they have accepted
- a candidate mutable record with a lower trusted counter, cursor, or version than the local trusted value is stale and must be rejected
- if a candidate record presents the same trusted counter, cursor, or version but different authenticated contents, clients must reject it as a divergence attempt
- clients may only clear trusted rollback state through an explicit local reset or account-recovery flow; ordinary restart, logout, or cache eviction must not erase trusted high-water marks
- compare-and-swap success at the storage layer is necessary for writes but not sufficient for reads; readers must still verify signatures and monotonicity locally

### Record-Specific Freshness Requirements

- account config must advance monotonically whenever device membership, default collection wiring, or other security-relevant account metadata changes
- collection head pointers must advance monotonically by event sequence or cursor; equal-sequence candidates are valid only when the authenticated `latestEventId` also matches
- snapshot pointers must not move to an older base cursor than the highest trusted snapshot or committed head for the same scope
- device metadata must reject removed or downgraded device state unless the change is authenticated by current account-level trust material
- membership state must reject stale active or stale revoked states; revocation is forward-looking but rollback to pre-revocation metadata is invalid
- invite state must reject replay to an earlier lifecycle state once acceptance, expiry, or revocation has been trusted

### Consumer Behavior

- clients must fetch mutable metadata before trusting referenced immutable objects
- clients must verify the signer, scope bindings, and freshness of mutable metadata before applying events, snapshots, memberships, or invites
- unreachable immutable objects that are not referenced by the latest trusted mutable metadata must not become authoritative
- if mutable metadata verification fails, clients must stop the affected sync or access flow and surface an integrity failure instead of silently repairing state

## I8 - Device Enrollment Contract

Status:

- accepted

Owner:

- Engineer2 (Claude), cross-boundary per W13 (`refs #35`); Engineer1 review required

Frozen for W13 (`refs #35`):

Goals:

- allow a new device to obtain the account master key (AMK) via an existing trusted device or via the recovery key
- keep the enrollment model explicit: no silent enrollment, no server-mediated key escrow
- preserve the zero-knowledge property across both enrollment paths

### Device Key Pairs

Each device generates two key pairs locally at enrollment time:

- Ed25519 key pair for signing events and mutable metadata (signing key)
- X25519 key pair for key agreement during device enrollment and invite flows (encryption key)

Private keys remain local on the device and must not leave the device.
Public keys are stored in the device record at `accounts/<accountID>/devices/<deviceID>.json`.

### Device Record Schema

The record is named `LocalDeviceRecord` and contains:

```json
{
  "schemaVersion": 1,
  "accountId": "<accountID>",
  "deviceId": "<deviceID>",
  "label": "<human-friendly label>",
  "deviceType": "cli|web",
  "signingPublicKey": "<base64url-no-pad, Ed25519 32-byte public key>",
  "encryptionPublicKey": "<base64url-no-pad, X25519 32-byte public key>",
  "createdAt": "<RFC 3339 timestamp>",
  "status": "active|revoked"
}
```

Fields:

- `schemaVersion`: integer, must be `1`
- `accountId`: the account this device belongs to
- `deviceId`: random 16-byte value, hex-encoded (see D14)
- `label`: non-empty human label for display; does not affect crypto
- `deviceType`: must be one of `"cli"` or `"web"`
- `signingPublicKey`: base64url without padding, 32-byte Ed25519 public key
- `encryptionPublicKey`: base64url without padding, 32-byte X25519 public key
- `createdAt`: RFC 3339 timestamp recorded at record creation time
- `status`: `"active"` for a live device; `"revoked"` for a removed device

### Storage Path

- device records are stored at `accounts/<accountID>/devices/<deviceID>.json`
- the persistence adapter stores and loads the device record as opaque bytes; it does not parse the crypto payload

### Device Enrollment Bundle (Existing-Device Approval)

When an existing trusted device approves a new device, it creates a `DeviceEnrollmentBundle`:

```json
{
  "schemaVersion": 1,
  "accountId": "<accountID>",
  "deviceId": "<new device ID>",
  "wrappedKey": {
    "algorithm": "x25519-hkdf-aes-256-gcm",
    "ephemeralPublicKey": "<base64url-no-pad>",
    "nonce": "<base64url-no-pad>",
    "ciphertext": "<base64url-no-pad>"
  }
}
```

Fields:

- `schemaVersion`: integer, must be `1`
- `accountId`: must match the account the new device is enrolling into
- `deviceId`: must match the new device's device ID
- `wrappedKey.algorithm`: must be `"x25519-hkdf-aes-256-gcm"`
- `wrappedKey.ephemeralPublicKey`: base64url without padding, 32-byte ephemeral X25519 public key generated by the approving device
- `wrappedKey.nonce`: base64url without padding, 12 random bytes for AES-256-GCM
- `wrappedKey.ciphertext`: base64url without padding, AES-256-GCM encryption of the 32-byte AMK

### Wrap and Unwrap Rules (Existing-Device Approval)

Key derivation uses ECDH + HKDF-SHA256 (see D13, D14):

- wrap:
  1. Approving device generates ephemeral X25519 key pair
  2. ECDH(ephemeral_private, new_device_X25519_public) → 32-byte shared secret
  3. HKDF-SHA256(ikm=shared_secret, salt=none, info=AAD) → 32-byte encryption key
  4. AES-256-GCM.Seal(key=encryption_key, nonce=random12, plaintext=AMK, aad=AAD)
- unwrap:
  1. ECDH(device_X25519_private, ephemeral_public) → 32-byte shared secret
  2. HKDF-SHA256 same params → 32-byte encryption key
  3. AES-256-GCM.Open → AMK
- AAD is the fixed ASCII string `"safe.device-enrollment.v1/"` concatenated with the `accountId`
- wrong key or corrupted ciphertext must return an authentication-failure error and must never expose partial plaintext

### Recovery-Key Bootstrap Path

When no trusted device is available:

- new device presents its X25519 public key and device ID during OAuth authentication
- user inputs the 24-word BIP-39 mnemonic → `recoveryKeyBytes` (via the BIP-39 library from W8)
- `OpenLocalRecoveryRecord(record, recoveryKeyBytes)` → AMK (reuses W8 implementation, no new crypto)
- device then creates its local device record with the freshly generated key pair
- no `DeviceEnrollmentBundle` is produced; the recovery path produces the AMK directly

### Negative Tests Required by W14

- wrong new-device private key during bundle unwrap → authentication-failure error, no plaintext
- corrupted ciphertext (bit-flipped) in enrollment bundle → authentication-failure error, no plaintext
- bundle with wrong account ID → validation error before any decryption attempt
- bundle with wrong device ID → validation error before any decryption attempt
- enrollment succeeds against a serialized on-disk fixture (not only in-memory helpers)

### Not in Scope for W13 or W14

- control-plane device registration or policy checks
- human-verifiable confirmation code (UX slice, see PROTOCOL.md)
- recovery-key input validation or checksum checking during mnemonic entry
- device revocation flows
- mutable metadata signing (separate slice)

## I9 - Account-Scoped Object Access Capability Contract

Status:

- accepted

Owner:

- Engineer1 (`refs #32`)

Frozen for W16 (`refs #32`):

Goals:

- define the minimal control-plane-issued access contract needed for single-account sync
- keep object-store access short-lived, path-scoped, and operation-bound
- leave sharing and broader collection-scoped authorization for later milestones

Capability payload:

```json
{
  "version": 1,
  "accountId": "<accountID>",
  "deviceId": "<deviceID>",
  "bucket": "<bucket name>",
  "prefix": "accounts/<accountID>/",
  "allowedActions": ["get", "put"],
  "issuedAt": "<RFC 3339 timestamp>",
  "expiresAt": "<RFC 3339 timestamp>"
}
```

Fields:

- `version`: integer, must be `1`
- `accountId`: the authenticated account receiving access
- `deviceId`: the authenticated device receiving access; must identify an active device record for the same account
- `bucket`: the object-store bucket the capability is valid against
- `prefix`: the only authorized object prefix for this capability; for W16 it is always `accounts/<accountID>/`
- `allowedActions`: non-empty set drawn from `"get"`, `"put"`, and `"list"`; `"list"` is optional and must never be implied by `"get"` or `"put"`
- `issuedAt`: RFC 3339 issuance time recorded by the control plane
- `expiresAt`: RFC 3339 expiry time recorded by the control plane

Capability rules:

- capabilities are authenticated by the control plane as signed opaque tokens whose payload matches the fields above
- W16 capabilities are account-scoped only; they must not grant collection-specific sharing access or any prefix outside `accounts/<accountID>/`
- the control plane must bind the capability to both `accountId` and `deviceId`; cross-account or cross-device reuse is invalid
- default issuance for single-account sync is `"get"` plus `"put"` only; `"list"` requires explicit request and must remain off by default
- clients and storage-facing adapters must treat `expiresAt` as a hard limit; expired capabilities are invalid even if the signature is otherwise correct
- verification must reject method escalation, prefix escape, empty prefixes, bucket mismatch, and non-account-local object paths

Out of scope for W16:

- durable control-plane storage for session state
- one-time-use capability tracking
- shared-collection authorization
- provider-specific IAM policies or pre-signed URL formats

## I5 - Handoff Protocol

Status:

- accepted

Rules:

- implementation owner updates `docs/project/HANDOFFS.md` when a task is ready for another engineer
- receiver acknowledges by updating the relevant task status in `docs/project/WORKBOARD.md`
- if a task blocks on an interface question, stop and record the exact blocker instead of guessing
