# Protocol Decisions

## Purpose

This document freezes the protocol-level decisions that must be agreed before implementation starts.

## Mandatory Rules

- All security-relevant records use canonical serialization before hashing or signing.
- All mutable security-relevant metadata must be authenticated.
- Clients must reject stale, unsigned, or invalid mutable metadata.
- Write operations must be idempotent.
- Object-store authorization must be narrowly scoped and short-lived.

## Canonical Records

The following record types require canonical serialization and authenticated verification:

- account config
- head pointers
- device records
- membership records
- invite state
- event records
- snapshot pointers

## Signers

- Device keys sign event records.
- Account-level trust material authenticates account config, snapshot pointers, device metadata, membership state, and invite state.
- Head pointers are authenticated by the device key that authored the committed event they reference.
- Membership and invite transitions must be authenticated by an authorized writer and validated by recipients.

Clients must reject unsigned or unverifiable mutable metadata. This is a blocking protocol rule, not an optional diagnostic.

## Rollback Rules

- Every client stores the highest trusted version, cursor, or sequence it has accepted for each rollback-sensitive mutable record family.
- Clients reject older authenticated mutable state unless explicitly recovering from a known-safe local reset flow.
- If two authenticated mutable records claim the same trusted counter or cursor but differ in authenticated contents, clients reject the candidate as divergence.
- Snapshots are valid only if their base cursor and authenticated metadata match the trusted event history.

## Capability Rules

- Signed URLs or scoped credentials must be short-lived.
- Capabilities must be bound to account ID, device ID, operation, and storage path scope.
- Shared collection access must never require broad owner-prefix access.
- Listing privileges should be avoided unless strictly required.

## Device Enrollment Rules

- Existing-device approval must bind a specific new-device public key, nonce, and human-verifiable confirmation code.
- Recovery-key bootstrap must require OAuth plus recovery key and must not weaken the trust model.
- Revoked devices must lose future capability issuance immediately.

## Invite Rules

- Invites must be one-time-use or replay-safe.
- Invite acceptance must bind recipient account and recipient device key explicitly.
- Invite expiry and revocation must be enforced as authenticated state transitions.
