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

- draft

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

Required persisted units:

- account config record
- collection head record
- vault event records
- secret material records

Required behaviors:

- initialize an empty account-local store
- write one or more records
- load all records needed to rebuild current state
- preserve deterministic ordering where tests depend on it
- survive process recreation in tests

Constraints:

- no network dependency
- no control-plane dependency
- no implicit starter fixtures
- no plaintext format assumptions in higher-level consumers

Notes:

- the first persistence adapter does not need to solve final encryption by itself
- it must not make future encryption layering awkward

## I2 - Local Unlock Contract

Status:

- draft

Owner:

- Engineer1

Goals:

- derive a local key from user password
- open encrypted local runtime state
- fail safely on wrong password or corrupted payload

Required behaviors:

- create new local unlock state for a first-use account
- reopen existing local state with the same password
- reject wrong password
- reject corrupted ciphertext or metadata

Not yet defined:

- exact record schema
- exact crypto envelope
- password rotation flow
- recovery-key support

## I3 - CLI Integration Contract

Status:

- draft

Owner:

- Engineer1

Current gap:

- `cmd/safe` currently constructs a fresh `MemoryObjectStore` and starter data on each run

Target:

- main save/read path uses durable local persistence plus real unlock flow

Rules:

- CLI wiring should consume the local runtime contract, not bypass it
- legacy starter bootstrap may remain only behind clearly marked dev-only paths while migration is in progress

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
