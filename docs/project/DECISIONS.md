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
- remaining decisions should focus on browser-specific adapter details and post-M1 polish, not on inventing a second runtime model
