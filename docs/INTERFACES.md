# Safe Interfaces

## Purpose

This file defines the current shared contracts between workstreams.

Related planning artifacts:

- `docs/WORKBOARD.md`
- `docs/HANDOFFS.md`
- `docs/GITHUB_PROJECTS.md`

If you need to change one of these contracts:

1. update this file first
2. note the change in `docs/HANDOFFS.md`
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

Not yet defined:

- exact unlock record schema
- exact crypto envelope
- password rotation flow
- recovery-key support

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

- web workspace helpers currently operate on replay inputs and optional secret material provided by the caller

Target:

- web surface consumes the same local-runtime concepts as the CLI, with a browser-specific storage adapter if needed

Rules:

- do not invent a separate unlock model in `apps/web`
- persisted workspace snapshots are transitional and should not become the final runtime API

## I5 - Handoff Protocol

Status:

- accepted

Rules:

- implementation owner updates `docs/HANDOFFS.md` when a task is ready for another engineer
- receiver acknowledges by updating the relevant task status in `docs/WORKBOARD.md`
- if a task blocks on an interface question, stop and record the exact blocker instead of guessing
