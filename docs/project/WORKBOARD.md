# Safe Workboard

## Purpose

This file is the in-repo source of truth for current execution.

Use it to answer:

- what milestone the team is working toward
- who owns which slice
- which files are safe to change
- what blocks what
- what is ready to merge

Keep this file current. If reality changes, update this file first.

GitHub companion artifacts:

- project board: `https://github.com/users/cyber-nic/projects/1`
- planning issues: `https://github.com/cyber-nic/safe/issues`
- CLI guide: `docs/project/GITHUB_PROJECTS.md`

Important:

- GitHub issues and the project board are the primary communication layer for progress, blockers, handoffs, and discussion
- this file is the canonical technical workboard — contracts, ownership, and write scopes live here
- every update to this file that affects a task or contract should reference the matching GitHub issue number

## Current Milestone

Milestone: `M2 - Multi-Device Single-User Sync Foundations`

GitHub issue:

- `#30`

Goal:

- two trusted devices can converge on one account through a real object-storage sync path
- clients verify signed mutable metadata and reject stale or replayed state before trusting it
- device enrollment requires existing-device approval or recovery-key bootstrap
- the shipped clients can prove the account loop across two devices, not only one local runtime

Current status:

- M1 remains complete on `main`; the CLI and local web client already close the first trustworthy local loop (`refs #1`, `#22`)
- post-M1 groundwork is in place: recovery-key wrap and unwrap shipped in code (`refs #19`) and rollback rules are now frozen as a contract (`refs #18`)
- M2 planning is now live in GitHub and in repo docs; no M2 implementation slice has landed yet (`refs #30`, `#31`, `#32`, `#33`, `#34`, `#35`, `#36`, `#37`)

Non-goals for this milestone:

- multi-user sharing and revocation
- browser-native adapter redesign
- broad web-product UX polish beyond the sync proof path
- OAuth production hardening beyond the narrow control-plane access path needed for single-account sync

Milestone kickoff status:

- W11 defines the M2 boundary and closes the planning reset (`refs #31`)
- W12, W13, and W15 are the critical-path contract and implementation slices for sync integrity (`refs #36`, `#35`, `#33`)
- W14, W16, and W17 stay explicit downstream tasks rather than hidden scope (`refs #34`, `#32`, `#37`)

## Execution Rules

- Work from the critical path outward, not from surface area inward.
- Each task must have one clear owner.
- Each task must declare its write scope before implementation starts.
- Shared interfaces must land before downstream wiring.
- Do not combine protocol changes and multiple consumer integrations in one commit unless required.
- Keep commits single-purpose and small enough to revert cleanly.
- Post a GitHub issue comment when work starts, blocks, hands off, or completes — this is required, not optional.
- Keep technical contract decisions in repo docs; issue comments are for communication, not canonical state.
- Every repo doc change that affects a tracked task must include the GitHub issue number (e.g. `refs #4`).
- When coordinating through GitHub, identify the speaking agent explicitly because all comments are posted via `cyber-nic`.

## Team

### Engineer1

Role:

- PM plus engineer
- owns milestone sequencing, interface definition, and integration decisions

Current owner:

- Codex

### Engineer2

Role:

- implementation owner for crypto and sync-heavy implementation slices

Status:

- W2 complete (`refs #4`)
- W8 complete (`refs #19`)
- W14 and W15 queued for M2 (`refs #34`, `#33`)

Current owner:

- Claude

### Engineer3

Role:

- reserved for the next parallel slice after local runtime contracts stabilize

Status:

- unassigned

## Ownership Map

These are default owners. Cross-boundary edits require a note in `docs/project/HANDOFFS.md`.

| Area                                     | Owner     | Write Scope                                                                                                        |
| ---------------------------------------- | --------- | ------------------------------------------------------------------------------------------------------------------ |
| Planning and sequencing                  | Engineer1 | `docs/project/WORKBOARD.md`, `docs/project/DECISIONS.md`, `docs/project/INTERFACES.md`, `docs/project/HANDOFFS.md` |
| Local runtime contract and integration   | Engineer1 | `cmd/safe/**`, `apps/web/**`, interface wiring only                                                                |
| Durable local persistence implementation | Engineer2 | `internal/storage/**`, new local-runtime packages under `internal/**`                                              |
| Crypto and password unlock primitives    | Engineer1 | `internal/crypto/**`, protocol-facing account/unlock records                                                       |
| Sync protocol hardening                  | Engineer1 | `internal/sync/**`, `packages/ts-sdk/**`, `packages/test-vectors/**`                                               |
| Control plane                            | Engineer1 | `cmd/control-plane/**`, `internal/auth/**`                                                                         |

## Interface Locks

The following interfaces are considered active contracts for this milestone. Do not change them casually.

1. Local runtime save or load contract
   Output needed:
   - initialize account-local storage
   - persist account config, collection head, vault item records, vault event records, and secret material
   - save encrypted secret material
   - load encrypted secret material after restart
   - lock and unlock boundaries

2. CLI integration path
   Output needed:
   - `cmd/safe` must stop bootstrapping from a fresh in-memory store for the main save/read path

3. Minimal web persistence path
   Output needed:
   - web model can hydrate from a durable local snapshot only after the runtime contract is defined

If one of these contracts changes, update `docs/project/INTERFACES.md` and record the handoff.

## Active Tasks

### W1 - Freeze local-runtime contract

Owner:

- Engineer1

Status:

- completed

Write scope:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- integration notes only in `cmd/safe/**` and `apps/web/**`

Output:

- minimal contract for password-derived unlock, encrypted persistence, and save/read lifecycle

Acceptance:

- contract is documented
- downstream implementation can proceed without guessing

Dependencies:

- none

GitHub issue:

- `#6`

### W2 - Implement durable local persistence adapter

Owner:

- Engineer2

Status:

- completed (`refs #4`)

Write scope:

- `internal/storage/**`
- optional new package under `internal/` if needed for local runtime

Output:

- non-memory persistence adapter for local vault state
- tests for restart-safe save or load behavior
- no CLI or web wiring in this task

Acceptance:

- can persist account config, collection head, vault item records, event records, and secret material
- event loading is deterministic by record sequence, not backend listing accidents
- related record writes do not expose a partial new head after failure
- survives process restart in tests
- does not require network or control-plane access

Dependencies:

- `docs/project/INTERFACES.md` local persistence contract

GitHub issue:

- `#4`

### W3 - Add password-derived local encryption primitives

Owner:

- Engineer1

Status:

- completed (`refs #5`)

Write scope:

- `internal/crypto/**`
- supporting domain records if required

Output:

- password derivation
- key wrapping format for local runtime
- negative tests for wrong password and corrupted payload

Dependencies:

- W1

GitHub issue:

- `#5`

### W4 - Replace CLI starter bootstrapping on the save/read path

Owner:

- Engineer1

Status:

- completed (`refs #3`)

Write scope:

- `cmd/safe/**`

Output:

- CLI saves to durable local runtime
- CLI can reopen and read after restart or unlock

Dependencies:

- W2
- W3

GitHub issue:

- `#3`

### W5 - Minimal web runtime integration

Owner:

- Engineer1

Status:

- completed (`refs #2`)

Write scope:

- `apps/web/**`

Output:

- web surface can consume the same local runtime contract without inventing a separate model

Dependencies:

- W1
- W2
- W3
- W4

GitHub issue:

- `#2`

### W6 - Close out M1 local-runtime loop

Owner:

- Engineer1

Status:

- completed (`refs #21`)

Write scope:

- `docs/project/**`
- `README.md`
- test-only updates in `cmd/safe/**` or `apps/web/**` if verification gaps require them

Output:

- verified milestone closeout notes for the shipped runtime work
- explicit follow-up tasks for any remaining critical-path gap

Dependencies:

- W5

GitHub issue:

- `#21`

### W11 - Freeze M2 sync-foundation scope and acceptance

Owner:

- Engineer1

Status:

- completed (`refs #31`)

Write scope:

- `docs/project/WORKBOARD.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- minimal alignment notes in `docs/architecture/IMPLEMENTATION_PLAN.md` if wording drifts

Output:

- explicit M2 milestone boundary, acceptance checks, and deferred-work list

Dependencies:

- M2 milestone issue `#30`

GitHub issue:

- `#31`

### W12 - Implement signed mutable-metadata verification in code

Owner:

- Engineer1

Status:

- planned (`refs #36`)

Write scope:

- `internal/sync/**`
- `internal/domain/**` if shared authenticated-record types are required
- `packages/test-vectors/**` if shared fixtures are needed
- narrow contract clarifications only in `docs/project/**`

Output:

- one verification path for signed mutable metadata plus stale-state rejection tests

Dependencies:

- W9
- W11

GitHub issue:

- `#36`

### W13 - Freeze device-enrollment contract for account recovery and new devices

Owner:

- Engineer1

Status:

- planned (`refs #35`)

Write scope:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- alignment-only notes in `docs/architecture/SYSTEM_DESIGN.md` and `docs/architecture/PROTOCOL.md` if needed

Output:

- frozen contract for existing-device approval and recovery-key bootstrap

Dependencies:

- W8
- W9
- W11

GitHub issue:

- `#35`

## Next Assignment

The next implementation wave is now defined and on the board.

Current focus:

- merge W11 so the repo workboard, decisions log, and GitHub project stay aligned on M2
- treat W12 and W13 as the contract and integrity gates before downstream sync or client wiring
- keep browser-native adapter work explicitly deferred until the two-device sync proof exists

## Queued Tasks

### W14 - Implement device-enrollment primitives and persisted tests

Owner:

- Engineer2

Status:

- planned (`refs #34`)

Write scope:

- `internal/crypto/**`
- `internal/domain/**` for enrollment records
- `packages/test-vectors/**` if cross-runtime fixtures are required

Output:

- device-enrollment primitives plus persisted success and failure tests

Dependencies:

- W13
- W8

GitHub issue:

- `#34`

### W15 - Implement account-path object-store sync and commit protocol

Owner:

- Engineer2

Status:

- planned (`refs #33`)

Write scope:

- `internal/sync/**`
- `internal/storage/**` if the object-store adapter boundary needs expansion
- `packages/test-vectors/**` if integration fixtures are needed

Output:

- single-user object-store sync path with explicit commit semantics and stale-head rejection

Dependencies:

- W12
- W11

GitHub issue:

- `#33`

### W16 - Implement minimal control-plane access mediation for single-account sync

Owner:

- Engineer1

Status:

- planned (`refs #32`)

Write scope:

- `cmd/control-plane/**`
- `internal/auth/**`
- integration tests only where required

Output:

- narrow account-scoped object-access path for single-user sync

Dependencies:

- W11
- W13

GitHub issue:

- `#32`

### W17 - Deliver a two-device single-user sync smoke path

Owner:

- Engineer1

Status:

- planned (`refs #37`)

Write scope:

- `apps/web/**`
- `cmd/safe/**`
- `docs/project/**` for milestone closeout only if needed

Output:

- reproducible two-device sync proof through the shipped clients

Dependencies:

- W14
- W15
- W16

GitHub issue:

- `#37`

## Merge Order

1. W11 planning reset
2. W12 signed-metadata verification
3. W13 device-enrollment contract
4. W14 enrollment primitives
5. W15 object-store sync and commit protocol
6. W16 control-plane access mediation
7. W17 two-device smoke path

## PR Template

Every PR should include:

- task ID
- owner
- write scope
- dependency status
- acceptance checks run
- risks or open questions

## Daily Coordination

- comment on the matching GitHub issue when work starts, blocks, hands off, or completes — do this first
- update the GitHub Project item status to match the current task state
- update `Status` in this file when work starts or finishes; include the GitHub issue number in the entry
- append handoffs to `docs/project/HANDOFFS.md`; post a matching comment on the GitHub issue
- record durable decisions in `docs/project/DECISIONS.md`; reference the GitHub issue that drove the decision
- update `docs/project/INTERFACES.md` before implementing against a new shared contract
- use `docs/project/GITHUB_PROJECTS.md` for `gh` commands
