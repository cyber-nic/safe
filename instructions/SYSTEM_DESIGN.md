# Secret Manager System Design (v1)

## 1. Overview

This document defines the v1 architecture for a zero-knowledge secret manager for personal and family use.

The product goal is:

- Store secrets securely with client-side encryption only
- Work across desktop, mobile, browser extension, and CLI
- Support simple encrypted sharing between a small number of trusted users
- Remain fast, offline-capable, and operational on unreliable networks
- Use S3-compatible object storage as the durable data plane
- Avoid a traditional application database in v1

The intended outcome is a system that is cheap to operate, portable across cloud providers, and resistant to control-plane compromise because the server never has the material required to decrypt vault data.

## 2. Non-Goals

Out of scope for v1:

- Enterprise RBAC, SCIM, or SSO administration
- Real-time collaborative editing
- Server-side search or analytics over secret contents
- Fine-grained sharing below the collection level
- Password reset flows that can recover data without a recovery secret
- HSM- or KMS-dependent decryption paths

## 3. Design Principles

### Security

- All secret material is encrypted on the client before upload
- The control plane can authenticate users and authorize object access, but cannot decrypt vault data
- Key compromise should be scoped as narrowly as practical
- Device addition and sharing flows must be explicit and auditable

### Simplicity

- Object storage is the system of record
- Writes are immutable wherever possible
- A small number of durable object types should model the whole system
- v1 prefers correctness over aggressive optimization

### Performance

- Clients sync incrementally from an append-only log
- Clients cache ciphertext and maintain a local decrypted search index
- Decryption is on demand for full records, not bulk by default

### Resilience

- Clients are usable offline against local state
- Recovery does not depend on vendor-managed infrastructure
- Storage versioning protects against accidental overwrite and partial corruption

## 4. High-Level Architecture

The system has three parts.

### Client Applications

Clients include desktop, mobile, browser extension, and CLI. They are responsible for:

- User unlock with master password
- Key derivation and cryptographic operations
- Local encrypted storage
- Local indexing and search
- Sync, conflict handling, and snapshot generation
- Invite acceptance and key unwrap flows

### Control Plane

The control plane is intentionally thin. It is responsible for:

- OAuth-based identity
- Device registration and policy checks
- Invite coordination
- Authorization decisions
- Issuing short-lived scoped upload/download credentials or signed URLs
- Rate limiting and abuse protection

The control plane is not responsible for:

- Storing plaintext secrets
- Performing decryption
- Maintaining the primary vault state in a database
- Indexing secret content

### Object Storage

S3-compatible object storage holds:

- Immutable encrypted objects
- Append-only event logs
- Periodic encrypted snapshots
- Small mutable head pointers
- Invite payloads and device records

The object store is the durable source of truth.

## 5. Trust Model and Threat Model

### Trust Assumptions

Trusted:

- The client binary and runtime on a healthy user device
- The cryptographic primitives and their implementations
- The user to protect their master password and recovery secret

Not trusted:

- The control plane with respect to vault confidentiality
- The object storage provider with respect to vault confidentiality
- The network between client and backend

### Threats Addressed

- Control-plane compromise resulting in access to ciphertext only
- Object store breach resulting in access to ciphertext only
- Passive network monitoring
- Replay of stale signed requests
- Loss or theft of a locked device
- Malicious web pages attempting to abuse the browser extension
- Unauthorized continued access after a member is removed from a shared collection

### Threats Not Fully Addressed

- A fully compromised, unlocked endpoint
- Malware with access to the user session
- Phishing of the user’s OAuth account and master password
- Side-channel leakage from the host OS or browser

These limits should be stated plainly. "Zero knowledge" here means the service cannot decrypt user vault contents under the normal design; it does not mean compromise of an unlocked client is harmless.

## 6. Identity, Authentication, and Unlock

Identity and cryptographic authority are intentionally separate.

- OAuth establishes who the user is for account access and device registration
- The master password establishes access to vault keys

This split gives acceptable UX while preserving the property that the service cannot derive vault keys from OAuth alone.

### Login Flow

1. User authenticates to the control plane with OAuth.
2. Control plane returns account metadata and device enrollment state.
3. Client prompts for master password.
4. Client derives the root key locally and attempts to unwrap the account key material.
5. On success, the client unlocks the local vault and starts sync.

If OAuth succeeds but the master password is wrong, the user is authenticated to the service but cannot decrypt data.

## 7. Cryptographic Design

### Cryptographic Goals

- Strong resistance to offline password guessing
- Isolation between collections and items
- Efficient item updates without broad re-encryption
- A practical device enrollment and sharing model

### Key Hierarchy

The v1 hierarchy should be:

1. Master password
2. Password-derived root key via Argon2id with per-account salt
3. Account key-encryption key (KEK) derived from the root key
4. Random account master key (AMK), wrapped by the KEK
5. Random collection keys, wrapped by the AMK or by recipient-specific share records
6. Random item data keys, wrapped by the collection key

This is preferable to deriving every level from the password directly. The password-derived key should protect a random account master key, not serve as the vault data key itself. That enables password rotation without re-encrypting all vault contents.

### Recommended Primitive Set

- KDF: Argon2id
- Symmetric encryption: AES-256-GCM or XChaCha20-Poly1305
- Key wrapping: AEAD-based wrapping using random nonces
- Hashing: SHA-256 or BLAKE2/BLAKE3 for integrity metadata
- Public-key crypto for invites and device bootstrap: X25519 for key agreement, Ed25519 for signatures if signatures are needed

The implementation should standardize on one primitive set across all clients. Mixing algorithms by platform creates migration and audit risk.

For v1, the primitive set should be frozen in Phase 0 and used consistently across all clients and services.

### Password Rotation

Password rotation should only re-wrap the AMK under a newly derived KEK. It should not trigger full vault re-encryption.

### Device Keys

Each device should have a device key pair generated locally during enrollment. Device public keys are stored in account metadata. Device private keys remain local in OS-protected secure storage where available.

Device keys support:

- Secure device provisioning
- Invite acceptance flows
- Optional future peer-assisted recovery or approval workflows

### Device Enrollment Model

v1 should make device addition explicit rather than implicit.

Supported enrollment paths:

- Existing-device approval: a currently unlocked device approves a new device and transfers the AMK or a device-unlock bundle encrypted to the new device public key
- Recovery-key bootstrap: if no trusted device is available, the user authenticates with OAuth and uses the recovery key to unwrap the AMK locally on the new device

Not supported in v1:

- Silent device enrollment based on OAuth alone
- Server-mediated escrow of device private keys or plaintext account keys

This keeps device addition consistent with the zero-knowledge model and avoids inventing a weaker fallback later.

## 8. Metadata Exposure

The current design must be explicit about what leaks and why.

### Encrypted Metadata

Encrypt:

- Item title
- Username
- URLs
- Tags
- Notes
- Secret payloads
- Collection names
- Membership notes or labels

### Cleartext Metadata

Keep cleartext limited to what is operationally required:

- Stable object IDs
- Object type
- Schema version
- Creation/update timestamps
- Parent references needed for sync and storage layout
- Content hash or checksum of ciphertext blobs
- Event sequence identifiers

### Important Caveat

Timestamps, object counts, object sizes, collection membership cardinality, and traffic patterns are metadata. They may leak useful information even when content is encrypted. The system should minimize but not pretend to eliminate this leakage.

## 9. Durable Object Model

v1 should define a small set of object types with explicit semantics.

### Account Config

Mutable, small, and frequently read.

Contains:

- Account ID
- Schema version
- KDF parameters and salt
- Wrapped AMK
- Current snapshot pointer
- Current event-log head pointer
- Region and storage configuration

### Device Record

One record per enrolled device.

Contains:

- Device ID
- Public keys
- Creation and last-seen timestamps
- Device type and friendly label
- Status: active, revoked

### Collection Record

Immutable logical record describing a collection.

Contains:

- Collection ID
- Encrypted metadata blob
- Owner account ID
- Current collection key version

### Collection Membership Record

Represents access to a collection for a user or device.

Contains:

- Membership ID
- Collection ID
- Recipient account or device ID
- Role
- Wrapped collection key for that recipient and key version
- Status: active, revoked

### Item Record

Immutable encrypted item object.

Contains:

- Item ID
- Collection ID
- Item version
- Encrypted metadata blob
- Encrypted content blob
- Wrapped item data key
- Author device ID
- Logical timestamp

### Event Record

Append-only sync log entry.

Contains:

- Event ID
- Monotonic sequence number or sortable logical cursor
- Event type
- References to affected objects
- Integrity hash
- Author device ID
- Event creation timestamp
- Idempotency key
- Base head or base cursor observed by the writer
- Mandatory author signature over canonical event bytes

### Snapshot Record

Periodic materialized state for fast sync bootstrap.

Contains:

- Snapshot ID
- Base event cursor
- Encrypted state bundle or manifest of current object heads
- Snapshot creation timestamp

### Invite Record

Used for out-of-band collection sharing and onboarding.

Contains:

- Invite ID
- Issuer account/device ID
- Target collection ID
- Expiry time
- Encrypted payload for recipient acceptance
- Status: pending, accepted, revoked, expired

## 10. Storage Layout

One acceptable layout is:

```text
/accounts/{account_id}/config.json
/accounts/{account_id}/heads/latest.json
/accounts/{account_id}/devices/{device_id}.json
/accounts/{account_id}/events/{sequence}.json
/accounts/{account_id}/events/by-device/{device_id}/{device_sequence}.json
/accounts/{account_id}/snapshots/{snapshot_id}.json
/accounts/{account_id}/collections/{collection_id}/meta.json
/accounts/{account_id}/collections/{collection_id}/memberships/{membership_id}.json
/accounts/{account_id}/collections/{collection_id}/items/{item_id}/{version}.json
/accounts/{account_id}/invites/{invite_id}.json
/shared/{collection_id}/manifest.json
/shared/{collection_id}/items/{item_id}/{version}.json
```

Guidelines:

- Event keys should be lexicographically sortable without listing deep directory trees
- Immutable objects should never be updated in place
- Only small pointer objects such as `heads/latest.json` and `config.json` should be mutable
- Bucket versioning should be enabled

Shared collections should not rely on broad cross-account bucket access. Shared objects should either live in an explicit shared namespace or be reachable through control-plane-issued capabilities scoped to specific collection paths.

## 11. Sync Model

The sync design needs stronger invariants than "events + snapshots".

### Source of Truth

- Immutable item, membership, and invite objects are the authoritative records
- The event log is the authoritative ordering mechanism for state transitions
- Snapshots are accelerators, not the primary source of truth

### Initial Sync

1. Fetch `config.json`
2. Read the latest snapshot pointer
3. Download and decrypt the snapshot
4. Replay all events after the snapshot cursor
5. Materialize local state
6. Persist the new local sync cursor

### Incremental Sync

1. Fetch the latest head pointer
2. If the local cursor is behind, download events after the current cursor
3. Validate event continuity, signatures, and rollback markers
4. Apply events deterministically
5. Fetch any referenced immutable objects not already cached
6. Advance the local cursor only after durable local commit

### Write Protocol

The write path must be specified tightly because object storage is not a database.

Recommended v1 protocol:

1. Writer reads the current head pointer and records its ETag and cursor.
2. Writer creates immutable objects for any new item, membership, or invite versions.
3. Writer creates a canonical event object containing an idempotency key, the base cursor, references to immutable objects, and the author device ID.
4. Writer signs the canonical event bytes with the device signing key.
5. Writer attempts a compare-and-swap update of `heads/latest.json` using the last observed ETag. The new head points to the new event cursor and event object and includes an authenticated rollback marker.
6. If the compare-and-swap succeeds, the write is committed.
7. If the compare-and-swap fails, the writer re-reads head state, checks whether its idempotency key already committed, and otherwise rebuilds against the new base state.

This means the mutable head pointer is the commit point. Immutable objects may exist without being committed; clients must ignore unreachable objects during normal replay and garbage collection can clean them up later.

### Sync Invariants

- Events must be idempotent
- Applying the same event twice must be safe
- A client must not advance its cursor before all referenced objects are locally committed
- Snapshots must be reproducible from prior state plus events
- A committed event must reference only immutable objects that already exist
- Writers must be able to detect whether a retried mutation already committed by matching on idempotency key
- Clients must reject unsigned or stale mutable metadata
- Head pointers, account config, membership state, and snapshot pointers must be rollback-detectable

### Snapshot Policy

Create a new snapshot:

- After N events
- After a configurable size threshold
- After expensive operations such as key rotation

Snapshots prevent unbounded replay cost but must be disposable and regenerable.

## 12. Concurrency and Conflict Resolution

Object stores do not provide database-style transactions, so v1 must keep the concurrency model conservative.

### Write Model

- New item versions are append-only
- Mutations emit a new immutable object and a corresponding event
- Head updates use optimistic concurrency based on the last observed head version or ETag

### Conflict Strategy

For v1:

- Treat item edits as last-writer-wins at the record level
- Surface conflicts when two writers update from the same base version
- Preserve prior immutable versions for manual recovery

This is intentionally simple. Rich field-level merges can be added later if they are justified by real usage.

The implementation should also preserve an explicit "base version" on mutation requests so conflicts can be surfaced deterministically instead of inferred from timestamps alone.

## 13. Local Client Storage

Each client maintains:

- Encrypted local object cache
- Local materialized state database
- Local search index over decrypted metadata
- Sync cursor and integrity markers
- Device key material in platform-secure storage when available

Recommended local stores:

- Desktop and CLI: SQLite with page-level encryption or encrypted blobs
- Browser extension: IndexedDB for cached ciphertext and a minimal decrypted index
- Mobile: SQLite or platform-native secure storage plus database

The local database should be treated as a cache plus usability layer, not the primary durable source of truth.

## 14. Search

Search is fully local in v1.

Indexed fields:

- Title
- Username
- URL
- Tags
- Notes

Design notes:

- Search index contents are sensitive because they are decrypted derivatives
- The index must be deleted on sign-out and protected at rest
- Browser extension search should avoid preloading full secret values unless the user explicitly opens the item

## 15. Sharing Model

v1 sharing is collection-based.

### Roles

- Owner: can rotate keys, invite members, revoke members
- Member: can read and write items within the collection

If a read-only role is needed, add it explicitly. Do not imply it exists.

### Share Flow

1. Owner creates an invite for a target user identity.
2. Recipient authenticates and presents a device public key.
3. Owner or issuing client wraps the current collection key for the recipient.
4. Recipient accepts the invite and stores the wrapped collection key in a membership record.
5. Recipient syncs collection metadata and items normally.

### Shared-Data Authorization

The authorization model must match the crypto model.

For v1:

- Shared collection objects should live in a collection-scoped namespace rather than only under the owner account path
- The control plane should authorize a member to fetch only the collection paths for collections where an active membership exists
- A recipient should not receive broad access to the owner's account prefix in order to read a shared collection

This keeps the access boundary narrow and makes revocation enforceable at the storage-authorization layer.

### Revocation

Revocation has two separate concerns:

- Stop future authorization to fetch new objects
- Prevent access to future data using old keys

For v1, revocation should mean:

1. Mark membership revoked in control-plane-visible metadata
2. Generate a new collection key version
3. Re-wrap that new key for remaining active members
4. Re-encrypt item keys or new item versions going forward under the new collection key version

Important limitation:

Revocation cannot make a previously authorized client forget plaintext or delete already-downloaded ciphertext. It only prevents access to future protected state. The document should be explicit about this.

The system should distinguish clearly between:

- Authorization revocation: stop fetching future protected objects
- Cryptographic forward revocation: rotate to a new collection key version for future writes

It does not provide retroactive deletion of prior plaintext exposure.

## 16. Browser Extension Security

The extension is one of the highest-risk surfaces.

Requirements:

- Cryptographic operations occur in extension-controlled context, never in page JS
- Content scripts should request secrets from a background/service worker process through a narrow API
- Autofill must be origin-bound and require strict URL matching rules
- Plaintext should only enter DOM fields at fill time
- Clipboard use should be explicit and time-limited
- Sensitive logs must be disabled in production builds

The extension should default to conservative behavior. A false negative on autofill is preferable to a credential leak.

## 17. Implementation Alignment

The implementation stack for v1 should be:

- Go for the control plane, storage-facing backend components, and CLI
- TypeScript for desktop UI, browser extension, and web-facing client code
- Material UI for the primary application UI layer
- Containers for local development and deployment consistency
- AWS as the primary cloud target, with S3 as the initial object store

Shared protocol schemas, canonical serialization rules, and test vectors must be defined independent of implementation language so Go and TypeScript implementations remain interoperable.
## 18. Recovery Model

Recovery needs more precision than "master password + recovery key".

### v1 Recovery Artifacts

- Master password
- Recovery key generated at account creation

The recovery key should be a high-entropy random secret encoded for human backup, not a user-chosen secondary password.

### Recommended Recovery Design

- The AMK is wrapped once by the password-derived KEK
- The same AMK is wrapped again by a recovery KEK derived from the recovery key

Recovery flow:

1. User proves identity with OAuth
2. User enters recovery key
3. Client derives recovery KEK locally
4. Client unwraps the AMK
5. Client prompts the user to set a new master password
6. Client re-wraps the AMK with a new password-derived KEK

If both the master password and recovery key are lost, data is unrecoverable by design.

Recovery should be validated as part of the initial account model, not deferred until release hardening. The first implementation milestone should prove that account creation, recovery-key storage guidance, AMK recovery, and password reset all work against real serialized account data.

## 19. Backend Responsibilities

The backend must remain narrow even as features are added.

Required responsibilities:

- OAuth handling and session validation
- Issuing short-lived object-store access scoped to account paths
- Issuing short-lived object-store access scoped to shared collection paths for active members
- Device registration and revocation
- Invite lookup and acceptance coordination
- Basic rate limiting, auditing, and abuse prevention
- Region routing

Explicitly excluded responsibilities:

- Decryption
- Search indexing
- Secret content processing
- Recovery escrow

## 20. Infrastructure

### Cloud Providers

Primary supported backends:

- AWS S3
- Cloudflare R2
- Google Cloud Storage

### Infrastructure as Code

Provision with Terraform:

- Buckets
- IAM or equivalent access policies
- Control-plane compute
- CDN or DNS as needed
- Logging and monitoring

### Bucket and Object Settings

- Versioning enabled
- Lifecycle rules for abandoned multipart uploads and stale superseded objects where safe
- Server-side encryption at rest enabled for baseline cloud hygiene

Server-side encryption at rest is not a substitute for client-side encryption. It is still worth enabling for operational defense in depth.

## 21. Regionality and Privacy

The cleanest v1 rule is:

- One account belongs to one region

Each region should have:

- Its own control-plane deployment
- Its own object storage bucket or namespace
- Region-local logs and monitoring where practical

Avoid global mutable state where possible. If a global identity service exists, it should store only the minimum needed to route users to the correct region.

## 22. Cost Model

Primary cost drivers:

- Stored ciphertext volume
- PUT and GET request volume
- Cross-region or internet egress
- Control-plane authentication and logging

Known risks:

- Too many small objects
- Expensive list operations
- Excessive snapshot churn
- Unbounded object version sprawl

Mitigations:

- Use predictable key layouts
- Prefer direct GET by known object key over list-heavy discovery
- Snapshot on thresholds, not on every write
- Compress large encrypted payloads before encryption where appropriate

## 23. Main Risks

### Security Risks

- Weak browser-extension boundaries
- Accidental metadata leakage through logs, metrics, or object naming
- Incorrect key rotation or invite acceptance logic

### Correctness Risks

- Sync divergence between clients
- Lost updates due to optimistic concurrency bugs
- Corrupt or incomplete snapshots

### Product Risks

- Recovery UX that users do not complete
- Sharing semantics that users misunderstand
- Excessive complexity in first-time device enrollment

These should drive test planning and staged rollout.

## 24. Testing Strategy

The original document omitted validation strategy. v1 should include it.

### Crypto Tests

- Test vectors for all key derivation and wrapping paths
- Cross-platform interoperability tests
- Negative tests for wrong password, wrong recovery key, corrupted ciphertext
- Device-enrollment tests for both existing-device approval and recovery-key bootstrap

### Sync Tests

- Replay correctness from empty state and from snapshot
- Idempotent event application
- Concurrent writers on multiple devices
- Partial upload and partial download failure recovery
- Compare-and-swap head-update contention and retry behavior
- Rollback detection for stale mutable heads or snapshots

### Sharing Tests

- Invite accept and reject flows
- Membership revocation and collection key rotation
- Access denial after revocation for future objects
- Authorization scoping for shared collection paths without owner-prefix overreach

### Client Security Tests

- Extension origin-binding tests
- Local cache lock/unlock behavior
- Secret redaction in logs and crash reports
- Local rollback-detection behavior when older signed heads or snapshots are replayed

## 25. Implementation Plan

### Phase 1

- Single-user vault
- Account unlock and local cache
- Basic event log and snapshot sync
- Desktop app and CLI

### Phase 2

- Browser extension
- Origin-bound lookup and autofill

### Phase 3

- Collection-based sharing
- Invite acceptance
- Member revocation and collection key rotation

### Phase 4

- Recovery key flow
- Export and import
- Mobile clients

### Phase 5

- Agent-native access patterns
- Short-lived scoped machine credentials

## 25. Future Direction: Agent-Native Secrets

This should remain explicitly future work.

Likely model:

- Dedicated machine or agent collections
- Narrow-scoped credentials with short TTLs
- Auditable issuance from a user-controlled client
- Separation between human vaults and automated runtime access

This should not distort the v1 human-centric data model.

## 26. Final Decisions for v1

- OAuth is for identity; the master password is for cryptographic unlock
- Object storage is the durable data plane
- No traditional backend database for primary vault state
- Clients are local-first and maintain encrypted local state
- Sharing is collection-based
- Password rotation re-wraps the AMK rather than re-encrypting the full vault
- Recovery requires a separately stored high-entropy recovery key
- Revocation protects future access, not already-exported plaintext

## 27. Summary

The v1 system is a client-side encrypted secret manager built on a thin control plane plus S3-compatible object storage. The design keeps decryption on the client, uses immutable objects and an append-only event log for synchronization, and relies on snapshots for fast bootstrap.

The critical implementation areas are:

- Correct key wrapping and recovery flows
- Deterministic sync and conflict handling
- Safe collection sharing and revocation
- Browser extension isolation

If those areas are implemented rigorously, the architecture is viable. If they are hand-waved, the product will look simple on paper and fail in production.
