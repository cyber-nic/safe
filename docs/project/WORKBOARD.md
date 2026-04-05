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

Milestone: `M1 - First Trustworthy Local Loop`

GitHub issue:

- `#1`

Goal:

- a user can identify into a real client surface
- unlock local state with a real password-derived path
- save one secret into durable encrypted local storage
- close or lock the client
- reopen or unlock and read that secret back

Current status:

- CLI local-runtime save or read is shipped and test-backed
- a local server-rendered web client now ships in `apps/web` and completes the same identify, unlock, save, lock, and reopen loop
- M1 is complete on `main`; remaining work is post-M1 hardening and expansion, not a hidden milestone gap

Non-goals for this milestone:

- sharing
- invites
- multi-device sync
- OAuth production hardening
- rich vault UX beyond what is needed to prove the loop

Milestone closeout status:

- implementation slices W1-W7 and W10 are complete (`refs #2`, `#3`, `#4`, `#5`, `#6`, `#20`, `#22`)
- the first trustworthy local loop is now closed for both the CLI and the local web client (`refs #1`)
- remaining queued work is explicitly post-M1: recovery implementation, rollback rules, and broader browser-native adapter decisions

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

- implementation owner for the first durable local runtime slice

Status:

- W2 complete (`refs #4`); W4 merged (`refs #3`); W7 in progress (`refs #20`)

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

## Next Assignment

No new implementation slice is assigned on this branch.

Current focus:

- keep M1 closed and avoid reintroducing hidden milestone scope
- move the queue forward through recovery implementation, rollback rules, and next-milestone definition

## Queued Tasks

### W10 - Deliver a real M1 client surface

Owner:

- Engineer1

Status:

- completed (`refs #22`)

Write scope:

- `apps/web/**`
- `cmd/control-plane/**` if the client surface needs the existing dev identity bootstrap path
- minimal doc updates in `docs/project/**` if the surface contract changes

Output:

- a navigable client surface where a user can identify, save one secret, lock or reload, and read it back

Outcome:

- `apps/web` now ships a local server-rendered client surface with real routes for identify, unlock, save, lock, and reopen
- the local web client persists account config, collection head, replayable events, item records, and encrypted secret material on disk
- the local web client uses the accepted account-scoped Argon2id plus AES-256-GCM unlock contract rather than inventing a second web-only unlock format

Dependencies:

- W5

GitHub issue:

- `#22`

### W7 - Freeze recovery-key account contract

Owner:

- Engineer2 (Claude), cross-boundary per D3

Status:

- completed (`refs #20`)

Write scope:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`

Output:

- frozen persisted recovery-key contract for account bootstrap and account recovery

Dependencies:

- W1
- W3

GitHub issue:

- `#20`

### W8 - Implement recovery-key wrap and recovery tests

Owner:

- Engineer2

Status:

- completed (`refs #19`)

Write scope:

- `internal/crypto/**`
- supporting account-domain records if required
- test vectors or fixtures if needed

Output:

- recovery-key generation plus AMK wrap and unwrap support
- persisted-account recovery tests and wrong-key or corrupted-payload coverage

Dependencies:

- W7

GitHub issue:

- `#19`

### W9 - Freeze signed mutable metadata and rollback rules

Owner:

- Engineer1

Status:

- todo (`refs #18`)

Write scope:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- architecture notes only where wording needs alignment

Output:

- frozen contract for signed mutable metadata, freshness checks, and rollback-sensitive objects

Dependencies:

- W1

GitHub issue:

- `#18`

## Merge Order

1. W1 contract docs
2. W2 persistence adapter
3. W3 crypto primitives
4. W4 CLI integration
5. W5 web integration

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
