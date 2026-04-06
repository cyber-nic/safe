# Safe Decisions

## Purpose

This file records decisions that should not be rediscovered in code review.

Each decision should include:

- status
- date
- owner
- decision
- rationale
- downstream impact

## Active Decisions

### D1 - GitHub is the communication layer; repo docs are the technical source of truth

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- GitHub issues and the project board are the primary communication layer for progress, blockers, handoffs, and discussion
- repo docs are the canonical technical record for contracts, interfaces, ownership, and decisions

Rationale:

- the team is still small but spans human and agent contributors who cannot share a chat channel
- GitHub provides a shared, asynchronous, auditable surface for all contributors
- keeping contracts next to code in repo docs reduces drift and avoids stale issue threads becoming canonical

Impact:

- post a GitHub issue comment at every state transition: start, block, hand off, complete
- update the GitHub Project item status to match the current task state
- `docs/project/WORKBOARD.md` is the active work queue and write-scope authority
- `docs/project/INTERFACES.md` is the contract source of truth
- every repo doc change that affects a tracked task must reference the matching GitHub issue number
- `docs/project/GITHUB_PROJECTS.md` is the CLI access guide

### D2 - Optimize for the first trustworthy local loop

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- current execution priority is the first real local save/read loop, not broader workspace richness

Rationale:

- this matches the implementation plan and current repo gap
- the project has enough fixture-backed surface area already

Impact:

- new work must justify itself against the local-runtime milestone
- non-critical surface expansion should be deferred
- runtime helper packages do not count as milestone completion unless they are exposed through a real client surface

### D3 - Ownership is directory-first

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- contributors default to one owned directory tree per task

Rationale:

- this lowers merge conflicts
- it keeps commits small and attributable

Impact:

- cross-boundary edits need an explicit handoff note
- shared contracts should land before consumer wiring

### D4 - Contract-first merge order

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- shared contract docs and interface locks land before downstream implementation branches

Rationale:

- Engineer2 and Engineer3 need stable boundaries
- this avoids parallel guessing in `cmd/safe`, `apps/web`, and `internal/storage`

Impact:

- `docs/project/INTERFACES.md` should be updated before implementation starts on a new shared seam

### D5 - The first durable local runtime keeps the existing object-key record model

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- the M1 durable local runtime will keep the current account and collection object-key layout already exposed by `internal/storage`

Rationale:

- the repo already has canonical record types and key helpers for account config, collection head, item records, event records, and secret material
- freezing that shape lets W2 implement restart-safe durability without waiting for a second schema design
- this keeps W4 focused on replacing the CLI bootstrap path rather than translating between two local models

Impact:

- W2 should implement a durable adapter that preserves the existing logical keys
- the backend may be file-backed for M1 as long as higher-level consumers stay backend-agnostic
- item records are part of the durable runtime contract, not an optional cache

### D6 - Local runtime mutations require a durable commit boundary

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- the local runtime must treat a vault mutation as one logical durable commit instead of a sequence of unrelated writes

Rationale:

- `cmd/safe` mutates secret material, item state, event history, and collection head together
- exposing a new head before the matching records are durable would break restart-safety and make later sync semantics harder to trust

Impact:

- W2 must provide a write path that can commit related records without exposing a partial new head after failure
- W4 should wire CLI mutations through that commit boundary instead of calling raw `Put` operations ad hoc

## Accepted Decisions

### D7 - Local unlock uses an account-scoped Argon2id record plus AES-GCM envelopes

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- W3 freezes the local unlock record at `accounts/<accountID>/unlock.json`
- the password path derives a 32-byte KEK with Argon2id and unwraps a random 32-byte account master key with AES-256-GCM
- encrypted secret material uses a versioned AES-256-GCM JSON envelope so the storage layer still treats it as opaque bytes

Rationale:

- W4 needs a stable unlock and secret-material format before wiring the CLI to durable local storage
- the architecture docs already point at Argon2id and an account master key instead of deriving vault data keys directly from the password
- a versioned JSON envelope is easy to inspect in tests while still keeping the storage contract backend-agnostic

Impact:

- `internal/crypto/**` is now the source of truth for local unlock and secret-material envelope parsing
- `internal/storage/**` stores secret material as opaque bytes and persists the account-scoped unlock record without understanding the crypto payload
- W4 should consume the frozen unlock record and envelope rather than introducing a second CLI-local format

Refs:

- `#5`

### D8 - Post-M1 planning uses explicit stabilization plus UX backlog slices

Status:

- accepted

Date:

- 2026-04-05

Owner:

- Engineer1

Decision:

- once W1-W5 are complete, the next planning step is split into:
  - W6: milestone closeout audit and release checklist
  - W7: post-M1 UX and reliability backlog definition with consistent issue tagging
- W6 and W7 should be tracked as dedicated issues after bootstrap under milestone issue `#1`

Rationale:

- the team needs clear separation between “verify what is done” and “define what comes next”
- two-engineer execution (Codex plus Claude) works best when one engineer handles closeout correctness while the other prepares backlog quality
- a normalized label taxonomy lowers future planning ambiguity and keeps the project board sortable

Impact:

- milestone-close communication should reference W6 and W7 explicitly instead of reopening completed W1-W5 issues
- new backlog issues should include `area/*`, `type/*`, and `priority/*` labels before being added to the project board
- PM should treat issue or project status drift as a blocking planning bug during closeout

Refs:

- `#1`

### D9 - Recovery key is raw high-entropy bytes; no KDF is applied

Status:

- accepted

Date:

- 2026-04-05

Owner:

- Engineer2 (Claude)

Decision:

- the recovery key is 32 random bytes used directly as the AES-256-GCM key for wrapping the AMK
- no KDF (Argon2id or similar) is applied to the recovery key before use

Rationale:

- the recovery key is generated by the client, not derived from a human-memorized secret
- 32 bytes of CSPRNG output already has 256 bits of entropy; a KDF would add computational cost without any security benefit
- applying a KDF would create confusion with I2 where a KDF is required precisely because the password has low entropy
- this keeps the recovery wrap/unwrap path simple and distinguishable from the password path

Downstream impact:

- W8 must not apply Argon2id or any KDF in the recovery wrap/unwrap path
- the recovery record schema has no `kdf` field; its absence is a deliberate contract signal, not an omission
- callers who try to reuse the password path for recovery will produce a type error at the interface boundary

Refs:

- `#20`

### D10 - W10 closes M1 with a local web client, not a browser-only crypto rewrite

Status:

- accepted

Date:

- 2026-04-05

Owner:

- Engineer1

Decision:

- W10 closes M1 by shipping a local server-rendered client surface under `apps/web`
- that surface uses the accepted account-scoped unlock record and encrypted secret-material envelope instead of inventing a second web-only unlock format
- a fully browser-native storage adapter remains a follow-up decision, not a prerequisite for closing the first trustworthy local loop

Rationale:

- the missing milestone gap was a navigable client surface, not another rewrite of the frozen local unlock contract
- Node can execute the accepted Argon2id plus AES-256-GCM path today, which lets the web client complete the real save or read loop without hiding new crypto divergence in `apps/web`
- closing M1 cleanly is better than letting browser-adapter ambiguity silently keep the milestone open

Downstream impact:

- `docs/project/WORKBOARD.md` and `README.md` should treat M1 as complete once W10 lands
- post-M1 planning should keep browser-native adapter work explicit instead of backfilling it into W10
- the web client surface should continue consuming the shared runtime model from `apps/web/src/index.ts` rather than bypassing it

Refs:

- `#22`

### D11 - Signed mutable metadata is a mandatory trust boundary

Status:

- accepted

Date:

- 2026-04-05

Owner:

- Engineer1

Decision:

- account config, collection head pointers, snapshot pointers, device metadata, membership state, and invite state are mandatory authenticated mutable metadata
- clients must verify canonical bytes plus signer ownership and monotonic freshness for those records before they trust referenced immutable state
- stale, divergent, or unsigned mutable metadata is rejected as an integrity failure, not accepted with a warning or repaired heuristically

Rationale:

- the security review already identifies mutable metadata rollback as the main integrity risk after the local-runtime milestone
- the repo has canonical account and head records plus monotonic head checks, but the project docs had not yet frozen the broader signer and freshness contract for sync-critical mutable state
- freezing the trust boundary now lets later sync, sharing, and multi-device work build against one rule instead of each slice inventing its own rollback semantics

Downstream impact:

- future sync work must treat mutable metadata verification as blocking behavior before event replay or snapshot restore
- device enrollment, membership, and invite flows must bind the exact account or device identities they authorize and must reject replay to earlier trusted states
- protocol and test-vector work should add authenticated record fixtures and stale-state rejection cases rather than relying on control-plane honesty

Refs:

- `#18`

### D12 - Post-M1 execution moves to multi-device single-user sync foundations

Status:

- accepted

Date:

- 2026-04-06

Owner:

- Engineer1

Decision:

- the next milestone after M1 is `M2 - Multi-Device Single-User Sync Foundations`
- M2 prioritizes signed mutable-metadata verification in code, device-enrollment contracts and primitives, object-store sync, and the narrow control-plane path needed to prove two-device single-user sync
- browser-native adapter redesign, multi-user sharing, and broad web-product polish stay out of the critical path until the two-device sync proof exists

Rationale:

- M1 is closed, so the next highest-risk gap is not another local-runtime surface change; it is whether Safe can move from one trusted device to two without losing integrity guarantees
- the security and implementation docs both identify rollback protection, device enrollment, and object-store sync as the hard blockers between a local demo and a real zero-knowledge product
- keeping browser-adapter work and broader UX polish explicitly deferred prevents the team from rebuilding surface area before the sync and trust model is proven

Downstream impact:

- W11-W17 become the active queue under milestone issue `#30`
- W12 and W13 are contract and trust-boundary gates for downstream implementation work
- future planning should treat browser-native storage questions as a follow-up stream, not as grounds to dilute M2 acceptance

Refs:

- `#30`
- `#31`

### D13 - Device enrollment uses X25519-ECDH + HKDF-SHA256 + AES-256-GCM for AMK transfer

Status:

- accepted

Date:

- 2026-04-06

Owner:

- Engineer2 (Claude), cross-boundary per W13 (`refs #35`)

Decision:

- existing-device approval wraps the AMK using an ECIES-like scheme: ephemeral X25519 key agreement, HKDF-SHA256 for key derivation, and AES-256-GCM for authenticated encryption
- the ECDH shared secret is never used directly as an encryption key; HKDF-SHA256 is always applied with the enrollment AAD as the info string
- the approving device generates a fresh ephemeral X25519 key pair for each bundle; the ephemeral private key is discarded after sealing
- AAD binds the ciphertext to its account ID and prevents cross-account bundle transplant

Rationale:

- X25519 + HKDF + AES-256-GCM is the same primitive combination used in Signal, TLS 1.3, and Noise; it is well-audited and maps cleanly onto `golang.org/x/crypto`
- HKDF step ensures the raw ECDH output (which has structure) is properly extracted before use as a symmetric key
- this avoids inventing a new protocol and stays consistent with the project's existing AES-256-GCM envelope pattern

Downstream impact:

- W14 implementation uses `golang.org/x/crypto/curve25519` for X25519 and `golang.org/x/crypto/hkdf` for key derivation
- no additional dependencies beyond what is already in go.mod

Refs:

- `#35`

### D14 - Device IDs are random 16-byte values hex-encoded without separators

Status:

- accepted

Date:

- 2026-04-06

Owner:

- Engineer2 (Claude), cross-boundary per W13 (`refs #35`)

Decision:

- device IDs are 16 random bytes from `crypto/rand`, hex-encoded as a 32-character lowercase string (no hyphens, no UUID formatting)
- device IDs are generated once at enrollment time and are stable for the lifetime of the device record

Rationale:

- 128-bit random ID space is sufficient for collision avoidance in a personal secret manager
- avoiding UUID formatting removes a UUID library dependency; the ID is opaque and never parsed, only compared
- hex encoding is unambiguous and embeds safely in storage paths without escaping

Downstream impact:

- device record path is `accounts/<accountID>/devices/<deviceID>.json` where `<deviceID>` is the 32-character hex string
- generation requires only `crypto/rand` and `encoding/hex`

Refs:

- `#35`

### D15 - M3 prioritizes the first real single-user web product loop after M2

Status:

- accepted

Date:

- 2026-04-06

Owner:

- Engineer1

Decision:

- the next milestone after M2 is `M3 - Web Product MVP for Single User`
- M3 prioritizes four slices in order: production OAuth identity, web onboarding with recovery-key acknowledgement, real web vault CRUD, and web sync or device-management visibility
- multi-user sharing, revocation, browser extension work, and broader product polish stay out of the M3 critical path

Rationale:

- M2 proved the trust model and object-store path, but the shipped web surface is still a narrow proof client rather than the first product users will judge
- the implementation plan already prefers a single-user product before shared collections, so the next milestone should productize the current single-account foundation instead of reopening storage or protocol scope
- explicit deferral prevents the team from smuggling collection or sharing work into the web-product milestone

Downstream impact:

- W18-W22 are now the active Engineer1 queue under milestone issue `#48`
- W19 must establish one real identity path consumed by both CLI and web before onboarding or vault UX expands
- M3 acceptance is measured by user-visible web flows, not by additional protocol helpers without product exposure

Refs:

- `#48`
- `#49`

## Open Questions

### P3 - Web local runtime storage boundary

Status:

- pending

Owner:

- Engineer1

Question:

- how much of the CLI local runtime contract can be shared with the web surface before browser-specific storage adapters diverge

Decision driver:

- avoid inventing a second runtime model
- W5 aligned the web runtime around account config, optional head state, replayable events, and locked secret-material boundaries
- W10 closed M1 with a local web client; remaining decisions should focus on browser-specific adapter details and post-M1 polish, not on inventing a second runtime model
