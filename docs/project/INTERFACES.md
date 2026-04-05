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

- W5 moved the web runtime toward the same account-local concepts as the CLI path
- browser-specific storage adapter details and unlock UX remain thinner than the CLI path
- `apps/web` is still a runtime helper package rather than a navigable authenticated client surface

Target:

- web surface consumes the same local-runtime concepts as the CLI, with a browser-specific storage adapter if needed

Rules:

- do not invent a separate unlock model in `apps/web`
- persisted workspace snapshots are transitional and should not become the final runtime API
- runtime persistence should prefer account config, collection head, replayable events, and opaque secret-material boundaries over derived UI state
- M1 is not complete until those runtime helpers are exposed through a real client surface instead of test-only package entry points

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

## I5 - Handoff Protocol

Status:

- accepted

Rules:

- implementation owner updates `docs/project/HANDOFFS.md` when a task is ready for another engineer
- receiver acknowledges by updating the relevant task status in `docs/project/WORKBOARD.md`
- if a task blocks on an interface question, stop and record the exact blocker instead of guessing
