# Secret Manager Implementation Plan (v1)

## 1. Purpose

This document turns [SYSTEM_DESIGN.md](/Users/ndelorme/cipher/SYSTEM_DESIGN.md) into an execution plan for building v1 of the secret manager.

It is intended to answer:

- What gets built first
- What can be deferred safely
- How to structure the codebase and workstreams
- What "done" means for each milestone
- What risks must be validated early

This plan assumes a small team optimizing for correctness, speed of iteration, and low operational complexity.

## 2. Delivery Strategy

The delivery strategy for v1 is:

1. Prove the cryptographic and sync core before building surface area.
2. Ship a single-user product before shared collections.
3. Add the browser extension only after local storage, unlock, and sync are stable.
4. Treat sharing, revocation, and recovery as security-sensitive features, not "later polish".
5. Keep the backend narrow from day one to avoid accidental server-side state creep.

The critical path is not UI. The critical path is:

- Key management
- Local encrypted storage
- Deterministic sync
- Correct object-store semantics
- Rollback detection and integrity protection for mutable metadata
- Safe device and sharing flows

## 3. v1 Scope

### Must Ship

- OAuth identity
- Master-password unlock
- Account key wrapping and recovery key support
- Explicit device enrollment via existing-device approval or recovery-key bootstrap
- Desktop app
- CLI
- Local encrypted cache
- Event-log plus snapshot sync against S3-compatible storage
- Personal vault
- Local search
- Browser extension with conservative autofill
- Collection-based sharing
- Member revocation with collection key rotation
- Export and import

### Can Slip Without Breaking v1

- Mobile clients
- Rich item templates beyond passwords, notes, API keys, and SSH keys
- Read-only sharing role
- Advanced audit UX
- Polished admin controls for multiple devices
- Agent-native secrets

### Must Not Be Added in v1

- Server-side decryption
- Server-side search
- Traditional primary database for vault contents
- Fine-grained ACLs below collection level
- Real-time collaboration

## 4. Recommended Repository Layout

The implementation should start with a monorepo. v1 has too much shared logic to split early.

Suggested structure:

```text
/apps
  /desktop
  /extension
  /cli
  /control-plane
/packages
  /crypto
  /data-model
  /sync-engine
  /storage-adapter
  /auth-client
  /vault-core
  /search
  /ui-kit
  /test-vectors
/infra
  /terraform
/docs
  SYSTEM_DESIGN.md
  IMPLEMENTATION_PLAN.md
```

Principles:

- All cryptographic logic lives in one shared package
- All object schemas and serialization logic live in one shared package
- Sync behavior is implemented once and reused across desktop, CLI, and extension where practical
- The backend should depend on shared schemas but not on client-side secret-handling code
- The shared schemas package should include canonical bytes for signing and rollback detection, not just validation types

## 5. Technology Choices

These choices should be fixed early to reduce rework.

### Core Language

- TypeScript across clients and control plane

Reason:

- Shared logic across desktop, extension, CLI, and backend
- Faster iteration for a small team
- Easier schema and test-vector reuse

### Desktop

- Electron or Tauri

Recommendation:

- Choose Electron if team speed and extension-like web UI reuse matter more than binary size
- Choose Tauri if the team is comfortable with Rust ownership for shell/native boundaries

For v1, consistency of the shared vault core matters more than theoretical footprint gains.

### CLI

- Node.js-based CLI using the same core packages as desktop

### Browser Extension

- Manifest V3 extension
- Shared core logic plus extension-specific secure boundary code

### Control Plane

- TypeScript service deployed as serverless functions or a small container service
- Posture should remain stateless except for minimal identity/session coordination if absolutely required

### Local Storage

- Desktop/CLI: SQLite
- Extension: IndexedDB

### Infrastructure

- Terraform for provisioning
- Initial target object store: AWS S3
- Abstraction layer for R2 and GCS compatibility

## 6. Workstreams

The work should be split into parallel workstreams with explicit ownership.

### Workstream A: Cryptography and Key Management

Scope:

- Argon2id parameterization
- KEK/AMK wrapping
- Collection key and item key wrapping
- Recovery key design
- Device key generation and storage interfaces
- Test vectors and interoperability tests

Outputs:

- Stable crypto package
- Versioned key schema
- Deterministic test vectors

### Workstream B: Data Model and Serialization

Scope:

- Canonical object schemas
- Versioned serialization format
- Object validation
- Integrity-hash strategy
- Migration hooks for future schema changes

Outputs:

- Shared model package
- Validation library
- Golden fixtures

### Workstream C: Sync Engine

Scope:

- Event fetch/apply loop
- Snapshot creation and restore
- Cursor management
- Optimistic concurrency
- Compare-and-swap head commit protocol
- Rollback detection for mutable pointers and snapshots
- Idempotent replay
- Recovery from partial failure

Outputs:

- Reusable sync engine package
- End-to-end sync tests
- Fault-injection test coverage

### Workstream D: Local Vault Runtime

Scope:

- Unlock lifecycle
- Encrypted local cache
- Search index
- Session lock and wipe behavior
- Import/export pipeline

Outputs:

- Shared vault runtime package
- Local persistence adapters

### Workstream E: Control Plane

Scope:

- OAuth integration
- Device registration
- Signed URL issuance or scoped object credentials
- Shared collection path authorization
- Invite coordination
- Region routing
- Rate limiting and audit events

Outputs:

- Minimal backend service
- Auth and policy middleware
- Integration tests against object storage

### Workstream F: Desktop App

Scope:

- Account onboarding
- Unlock UI
- Vault browsing/editing
- Search
- Device management
- Recovery and export UX

Outputs:

- Primary v1 user client

### Workstream G: Browser Extension

Scope:

- Background/service worker secret boundary
- Site matching
- Secure lookup and fill flows
- Lock state handling

Outputs:

- Conservative autofill-capable extension

### Workstream H: CLI

Scope:

- Login and unlock
- Sync
- Read/create/update secrets
- Export
- Script-friendly output modes with safe defaults

Outputs:

- Operational CLI for power users and future automation

## 7. Phase Plan

### Phase 0: Foundations

Goal:

- Make irreversible decisions early and remove architectural ambiguity.

Build:

- Monorepo scaffold
- CI pipeline
- Package boundaries
- Linting, typecheck, formatting, and test harness
- Basic Terraform layout
- Secrets-free local dev environment

Decisions to lock:

- Primary language/runtime
- Crypto primitive set
- Object schema strategy
- Event cursor format
- Snapshot encoding
- Local database strategy
- Signed event/head format
- Device enrollment ceremony
- Shared-object authorization model

Exit criteria:

- Repository structure exists
- CI runs tests and typecheck
- One package can publish shared model types to all apps
- Terraform can provision a non-production bucket and control-plane skeleton

### Phase 1: Crypto and Data Model Core

Goal:

- Build the security-critical primitives before any real product features depend on them.

Build:

- Argon2id-based password derivation
- AMK wrapping and unwrap flow
- Recovery key generation and wrap path
- End-to-end recovery using serialized account config
- Collection key and item data key wrapping
- Device key pair generation
- Device-enrollment protocol for existing-device approval and recovery-key bootstrap
- Canonical schemas for account config, device, collection, membership, item, event, snapshot, invite

Tests:

- Unit tests for all wrap/unwrap paths
- Wrong-password and corrupted-ciphertext negative tests
- Recovery-key success/failure tests against persisted account state
- Device-enrollment tests for both enrollment paths
- Stable fixtures for each object type

Exit criteria:

- The crypto package is versioned and consumed by at least one app package
- All object types serialize and validate deterministically
- Password rotation can re-wrap the AMK without rewriting item data
- Recovery and device enrollment work end to end against real account fixtures

### Phase 2: Local Vault Runtime

Goal:

- Make single-device offline usage reliable before any remote sync is introduced.

Build:

- Local vault database
- Unlock/lock state machine
- Encrypted ciphertext cache
- Decrypted metadata index
- Local search
- Create/read/update/delete item flows
- Session wipe on sign-out

Tests:

- Lock/unlock persistence tests
- Search index rebuild tests
- Crash recovery tests

Exit criteria:

- A desktop or CLI client can create and manage secrets entirely offline
- Local cache remains inaccessible while locked
- Local search works across supported fields

### Phase 3: Object Storage and Sync

Goal:

- Make multi-device single-user sync correct.

Build:

- Storage adapter for S3
- Immutable object upload/download
- Event append flow
- Snapshot generation and restore
- Head pointer updates with optimistic concurrency
- Idempotent write protocol with explicit commit point and retry rules
- Rollback detection for stale heads and snapshots
- Cursor persistence and replay

Tests:

- Empty bootstrap from object storage
- Replay from snapshot plus delta events
- Duplicate event application
- Mid-sync interruption and recovery
- Concurrent writes from two devices
- Compare-and-swap contention and safe retry
- Replayed stale head or snapshot rejection

Exit criteria:

- Two devices can converge on the same state through object storage
- Replay is idempotent
- Partial failure does not corrupt local state or advance the cursor incorrectly
- Clients detect and reject stale mutable metadata instead of silently rolling back

### Phase 4: Control Plane and Identity

Goal:

- Add account identity and authorized object access without weakening the zero-knowledge model.

Build:

- OAuth login
- Account creation flow
- Device registration
- Signed URL issuance or scoped storage credentials
- Collection-scoped authorization for shared data
- Basic audit events
- Rate limiting

Tests:

- OAuth success/failure flows
- Device registration and revocation flows
- Path scoping for object access
- Shared collection access without owner-account overreach

Exit criteria:

- A user can sign in with OAuth, unlock locally, and sync only their own account path
- Revoked devices lose future object access through the control plane
- Shared collection members can fetch only the collection paths they are authorized to read

### Phase 5: Desktop Product MVP

Goal:

- Ship a usable single-user desktop product.

Build:

- Onboarding UX
- Unlock UX
- Vault list/detail views
- Item creation and editing
- Search
- Manual sync controls
- Export/import UX
- Recovery-key presentation and backup confirmation flow
- Existing-device approval flow for new desktop enrollment

Tests:

- End-to-end desktop smoke tests
- Regression tests for lock state, sync, and export
- Recovery-key acknowledgement and restore drill tests

Exit criteria:

- A normal user can sign up, back up their recovery key, create secrets, search them, sync them, and export them

### Phase 6: Browser Extension

Goal:

- Add secure lookup and autofill without compromising the core security model.

Build:

- Extension onboarding tied to existing account/device
- Lock/unlock behavior
- Site matching rules
- Manual fill and conservative autofill
- Secret lookup UI

Tests:

- Origin-binding tests
- No-page-JS access tests
- Fill behavior tests across login form variants

Exit criteria:

- The extension can securely retrieve entries and fill matching sites
- The extension fails closed on ambiguous origin matches

### Phase 7: Sharing and Revocation

Goal:

- Add encrypted multi-user sharing with clear semantics and safe defaults.

Build:

- Collection creation
- Invite issuance
- Membership acceptance
- Per-recipient collection-key wrapping
- Membership revocation
- Collection key rotation
- Shared-object namespace or equivalent collection-scoped access path

Tests:

- Invite lifecycle tests
- Shared collection sync across users
- Revocation tests for future object access
- Re-key behavior tests after member removal
- Authorization boundary tests for collection-scoped credentials

Exit criteria:

- Two accounts can share a collection securely
- Revoked members cannot decrypt newly protected data after rotation
- Existing limitations of revocation are clearly surfaced in UX and docs

### Phase 8: Recovery, Hardening, and v1 Release

Goal:

- Close the highest-risk gaps before public release.

Build:
- Recovery flow UX refinements
- Password change flow
- Device management UI
- Audit/log redaction review
- Operational dashboards and alerts
- Backup/restore runbooks

Tests:

- Recovery key end-to-end flow
- Password rotation flow
- Structured security review
- Load and cost validation against realistic sync patterns

Exit criteria:

- Recovery works end to end without backend escrow
- Password rotation does not require full vault re-encryption
- Major security review findings are resolved or explicitly accepted

## 8. Milestones

The following milestones should be treated as external checkpoints.

### Milestone A: Crypto Core Ready

Definition:

- Shared crypto package is stable
- Test vectors exist
- Recovery and rotation paths work
- Device-enrollment paths work

### Milestone B: Single-Device Vault Ready

Definition:

- Offline local vault works
- Search works
- Lock/unlock is reliable

### Milestone C: Multi-Device Single-User Ready

Definition:

- Sync converges reliably across devices
- Snapshot and replay behavior is validated
- Rollback detection is validated

### Milestone D: Desktop MVP Ready

Definition:

- Desktop app is usable for a real primary account

### Milestone E: Extension Ready

Definition:

- Extension can safely retrieve and fill credentials on supported sites

### Milestone F: Shared Collections Ready

Definition:

- Invite, accept, sync, revoke, and key rotation all work

### Milestone G: Release Candidate

Definition:

- Recovery works
- Security review complete
- Operational runbooks exist

## 9. Acceptance Criteria by Capability

### Account Creation

Done when:

- A new user can authenticate with OAuth
- The client generates key material locally
- The AMK is wrapped for password and recovery use
- Initial account config is written successfully
- The recovery flow can unwrap that same AMK from persisted account data

### Unlock

Done when:

- Correct password unlocks the local vault and sync begins
- Wrong password never results in partial plaintext state
- Locked clients expose no decrypted data

### Item Management

Done when:

- Items can be created, edited, versioned, deleted, and restored from prior immutable versions if needed

### Sync

Done when:

- Clients converge deterministically
- Repeated replay is safe
- Interrupted sync can resume without corruption
- Stale mutable pointers are detected and not accepted silently

### Sharing

Done when:

- A collection can be shared to another account
- The recipient can decrypt data only after invite acceptance
- Revocation plus key rotation prevents future access
- Shared object access is scoped to authorized collection paths only

### Recovery

Done when:

- A user with OAuth identity and recovery key can recover access without contacting support for plaintext or escrowed keys

## 10. Security Gates

The following must block release if incomplete.

### Gate 1: Crypto Review

- Primitive choices fixed
- Key hierarchy documented
- Serialization format frozen
- Wrap/unwrap test vectors complete
- Device-enrollment ceremony documented and tested

### Gate 2: Sync Review

- Event application proven idempotent
- Cursor advancement rules tested
- Snapshot consistency verified
- Mutable head commit protocol and rollback detection verified

### Gate 3: Extension Review

- Origin binding validated
- No secret exposure to page JS
- Sensitive logging disabled

### Gate 4: Recovery Review

- Recovery key generation, storage guidance, and re-wrap path validated

### Gate 5: Logging and Telemetry Review

- No plaintext secret material
- No decrypted search terms in logs
- No key material in traces or metrics

## 11. Testing Plan

Testing should be layered.

### Unit Tests

- Crypto primitives and wrapping flows
- Object validation and migrations
- Search indexing logic

### Integration Tests

- Local database plus vault runtime
- Control plane plus object-store access
- Sync between multiple clients
- Shared collection authorization boundary enforcement

### End-to-End Tests

- Sign up, unlock, create item, sync to second device
- Install extension, unlock, fill site
- Share collection, accept invite, revoke member
- Recover account with recovery key

### Adversarial and Failure Tests

- Corrupted object bodies
- Missing events
- Duplicated events
- Replay of expired signed URLs
- Interrupted uploads
- Locked-device local disk inspection
- Replay of stale heads, stale snapshots, or revoked shared-path credentials

## 12. Operational Plan

Even v1 needs operational discipline.

### Required Runbooks

- Region bring-up
- Bucket recovery and version rollback
- OAuth outage handling
- Signed URL or credential issuance outage
- Incident response for suspected metadata exposure

### Required Telemetry

- Sync success/failure counts
- Event replay latency
- Snapshot generation latency
- Object-store error rates
- OAuth error rates
- Invite acceptance success/failure
- Compare-and-swap write contention rate
- Rollback-detection failures

Telemetry must be structured to avoid plaintext leakage.

## 13. Team Sequencing

For a small team, recommended sequencing is:

1. One owner on crypto plus object model.
2. One owner on sync and storage adapter.
3. One owner on desktop and shared vault runtime.
4. One owner on control plane and infra.
5. Extension work starts only after shared core packages are stable.

If the team is very small, do not start extension and sharing in parallel. Sharing is the harder security problem.

## 14. Top Risks During Execution

### Risk 1: UI-Led Development

Failure mode:

- The product looks advanced before the key and sync model are stable.

Mitigation:

- Block UI polish work behind crypto and sync milestones.

### Risk 2: Backend Scope Creep

Failure mode:

- The control plane slowly becomes a real stateful backend.

Mitigation:

- Enforce a written API boundary and review every new backend field for zero-knowledge impact.

### Risk 3: Ambiguous Commit Semantics

Failure mode:

- Concurrent writers commit unreachable objects, duplicate mutations, or divergent heads because the write protocol is under-specified.

Mitigation:

- Freeze an explicit compare-and-swap commit protocol with idempotency keys before sync implementation starts.

### Risk 4: Extension Security Shortcuts

Failure mode:

- Autofill convenience bypasses origin and context isolation rules.

Mitigation:

- Conservative defaults and explicit extension security review.

### Risk 5: Revocation Ambiguity

Failure mode:

- Product language implies stronger revocation than the system can provide.

Mitigation:

- Align UX, docs, authorization model, and tests with actual forward-secrecy limits.

### Risk 6: Recovery UX Failure

Failure mode:

- Users never store the recovery key, then lose access.

Mitigation:

- Force clear recovery-key acknowledgement during onboarding and test the flow early.

### Risk 7: Shared-Path Authorization Drift

Failure mode:

- Cross-account sharing is implemented by broadening storage credentials until they cover more than the intended collection scope.

Mitigation:

- Define a collection-scoped authorization model and verify it with boundary tests before shipping sharing.

## 15. Definition of v1 Done

v1 is done when all of the following are true:

- A user can create an account, store a recovery key, unlock the vault, and manage secrets locally
- Two devices can sync through S3-compatible storage reliably
- Desktop is usable as the main client
- CLI supports core vault operations
- Browser extension supports secure lookup and conservative autofill
- Shared collections work across two accounts
- Revocation plus collection key rotation protects future shared data
- Recovery works without backend escrow
- New devices can be added without weakening the zero-knowledge model
- Clients detect stale mutable metadata and fail safely
- Security gates have been passed
- Operational runbooks and telemetry are in place

## 16. Immediate Next Steps

If implementation starts now, the first concrete steps should be:

1. Create the monorepo and package boundaries.
2. Freeze the crypto primitive set and object schemas.
3. Freeze the device-enrollment ceremony, signed metadata format, and shared-object authorization model.
4. Build test vectors before UI.
5. Implement local vault runtime and offline item flows.
6. Implement S3 storage adapter and sync engine.
7. Add the control plane only to the extent needed for identity and access mediation.

That sequence keeps the hardest engineering risks on the critical path and avoids building product surface on unstable foundations.
