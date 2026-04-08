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

Milestone: `M3 - Web Product MVP for Single User`

GitHub issue:

- `#48`

Goal:

- a single developer can authenticate through the real OAuth path in both CLI and web (`refs #48`, `#50`)
- the web app can onboard a first-time account, including recovery-key acknowledgement, without falling back to a dev session (`refs #48`, `#51`)
- the web app can browse and mutate the personal vault after unlock instead of stopping at a thin proof surface (`refs #48`, `#52`)
- sync and enrolled-device visibility are surfaced in the web product without reopening the single-user trust model (`refs #48`, `#53`)

Current status:

- M1 and M2 are complete on `main`; the repo now has a trustworthy local loop plus a restart-safe two-device sync proof through the shipped clients (`refs #1`, `#22`, `#30`, `#37`)
- M3 is now complete on `main`; the repo ships the planned single-user web-product slices for real identity bootstrap, onboarding, vault CRUD, and sync or device-management visibility (`refs #48`, `#49`, `#50`, `#51`, `#52`, `#53`)
- follow-on work now moves beyond the frozen M3 boundary into real-provider OAuth and demoability improvements rather than reopening the shipped milestone scope; W24 is complete on `main`, and W23 is the active Engineer1 follow-up slice (`refs #59`, `#60`)

Non-goals for this milestone:

- multi-user sharing, revocation, and collection-key rotation
- mobile clients, browser extension work, or browser-native adapter redesign
- advanced audit UX, admin tooling, or organization features
- broad polish outside the critical onboarding, vault, sync, and device-management loops

Milestone completion status:

- W18 defines the M3 boundary and acceptance checks before implementation starts (`refs #49`)
- W19 replaces the dev-session identity shortcut with the real OAuth login path shared by CLI and web (`refs #50`)
- W20, W21, and W22 layer onboarding, vault CRUD, and sync or device-management UX on top of the shipped M2 foundation (`refs #51`, `#52`, `#53`)

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
- W13 cross-boundary complete (`refs #35`)
- W14 complete (`refs #34`)
- W15 complete (`refs #33`)

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

1. Production identity path
   Output needed:
   - CLI and web authenticate through the same non-dev identity boundary
   - `/v1/access/account` validates real user identity before issuing account-scoped capabilities

2. Web onboarding path
   Output needed:
   - first-time users create local account state, see the recovery key once, and must acknowledge it before vault access
   - returning users go straight to unlock instead of replaying onboarding

3. Web vault runtime path
   Output needed:
   - unlocked web state exposes real personal-vault list, item detail, create, edit, delete, and search flows
   - vault access remains gated on the existing local unlock boundary

4. Web sync and device-management surface
   Output needed:
   - sync push and pull can be triggered from the web client and fail closed on stale or rejected state
   - enrolled-device visibility and approval paths stay aligned with the M2 trust model

If one of these contracts changes, update `docs/project/INTERFACES.md` and record the handoff (`refs #49`).

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

- completed (`refs #36`)

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

- Engineer1 (default); completed cross-boundary by Engineer2 (`refs #35`)

Status:

- completed cross-boundary; Engineer1 review pending (`refs #35`)

Write scope:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`

Output:

- I8 device-enrollment contract: device record schema, DeviceEnrollmentBundle (X25519+HKDF+AES-256-GCM), two enrollment flows, negative test requirements
- D13: ECIES variant for AMK transfer
- D14: DeviceID format (16-byte random hex)

Dependencies:

- W8
- W9
- W11

GitHub issue:

- `#35`

## Current Focus

The next implementation wave is now defined and on the board (`refs #48`, `#49`).

Current focus:

- freeze the M3 boundary first so the control-plane and web work stays scoped to a real single-user product loop
- keep sharing, revocation, and broader collection work explicitly queued behind the web-product MVP
- treat any contract changes that affect onboarding, identity, or device approval as explicit doc-first updates

## Recent Completions

### W14 - Implement device-enrollment primitives and persisted tests

Owner:

- Engineer2

Status:

- completed (`refs #34`)

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

- completed (`refs #33`)

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

- completed (`refs #32`, PR #43)

Write scope:

- `cmd/control-plane/**`
- `internal/auth/**`
- integration tests only where required

Output:

- narrow account-scoped object-access path for single-user sync
- freeze I9 for signed account-scoped access capabilities with default `get` and `put` only

Dependencies:

- W11
- W13

GitHub issue:

- `#32`

### W17 - Deliver a two-device single-user sync smoke path

Owner:

- Engineer1

Status:

- completed (`refs #37`, PR #46)

Write scope:

- `apps/web/**`
- `cmd/safe/**`
- `docs/project/**` for milestone closeout only if needed

Output:

- reproducible two-device sync proof through the shipped clients
- CLI `sync push` and `sync pull` prove the real object-store path across two devices; web consumes the merged account-access capability route during identify

Dependencies:

- W14
- W15
- W16

GitHub issue:

- `#37`

## Active M3 Queue

### W18 - Freeze M3 web-product scope and acceptance

Owner:

- Engineer1

Status:

- in progress (`refs #49`)

Write scope:

- `docs/project/WORKBOARD.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- minimal alignment notes in `docs/architecture/IMPLEMENTATION_PLAN.md` if wording drifts

Output:

- explicit M3 milestone boundary, acceptance checks, and deferred-work list

Dependencies:

- M3 milestone issue `#48`

GitHub issue:

- `#49`

### W19 - Replace dev-session identity with production OAuth login

Owner:

- Engineer1

Status:

- in progress (`refs #50`)

Write scope:

- `cmd/control-plane/**`
- `internal/auth/**`
- `cmd/safe/**` for CLI login command wiring
- `apps/web/**` for web login callback wiring

Output:

- CLI and web both authenticate through the same production identity path before account-scoped capability issuance

Dependencies:

- W18

GitHub issue:

- `#50`

### W20 - Web account onboarding flow

Owner:

- Engineer1

Status:

- in progress (`refs #51`)

Write scope:

- `apps/web/**`
- `docs/project/**` for any updated onboarding contract

Output:

- first-time web onboarding that creates local account state and requires recovery-key acknowledgement
- returning web users go through unlock instead of onboarding

Dependencies:

- W19

GitHub issue:

- `#51`

### W21 - Web vault CRUD UX

Owner:

- Engineer1

Status:

- completed (`refs #52`)

Write scope:

- `apps/web/**`

Output:

- unlocked web vault list, item detail, create, edit, delete, local search, and TOTP display flows for the personal vault

Dependencies:

- W20

GitHub issue:

- `#52`

### W22 - Web sync and device management UX

Owner:

- Engineer1

Status:

- completed (`refs #53`)

Write scope:

- `apps/web/**`
- `cmd/safe/**` if CLI sync commands need narrow shared-hook adjustments

Output:

- sync push or pull visibility in the web app plus enrolled-device review and approval controls

Dependencies:

- W21
- W17

GitHub issue:

- `#53`

### W23 - Real OAuth provider integration

Owner:

- Engineer1

Status:

- in progress (`refs #59`)

Write scope:

- `cmd/control-plane/**`
- `internal/auth/**`
- `apps/web/**`

Output:

- a real provider-backed OAuth redirect flow plus control-plane verification that no longer depends on injected dev tokens for normal sign-in

Dependencies:

- M3 complete on `main`

GitHub issue:

- `#59`

### W24 - Make the localhost demo runnable by a human in one session

Owner:

- Engineer1

Status:

- completed on `main` via PR `#62` (`refs #60`)

Write scope:

- `compose.yaml`
- `Makefile`
- `.env.example`
- `README.md`
- `docker/**` for a narrow web dev-container addition if needed

Output:

- `make up` launches the full local demo stack, `make token` prints a usable dev JWT, and the README documents the browser-plus-CLI smoke path against the same local environment

Dependencies:

- none; this slice intentionally rides the existing HS256 dev-token path instead of waiting on W23

GitHub issue:

- `#60`

## PR Template

Every PR should include:

- task ID
- owner
- write scope
- dependency status
- acceptance checks run
- risks or open questions

## Daily Coordination

- comment on the matching GitHub issue when work starts, blocks, hands off, or completes — do this first, every time
- keep the GitHub Project item status current at all times:
  - **Todo** → **In Progress** the moment work starts
  - **In Progress** → **Blocked** immediately when a blocker is hit
  - **In Progress** → **In Review** when the PR opens
  - **In Review** → **Done** when the PR merges
- send a switchboard stand-up at each of the same moments — see `docs/project/SWITHBOARD.md` for format and trigger list
- update `Status` in this file when work starts or finishes; include the GitHub issue number in the entry
- append handoffs to `docs/project/HANDOFFS.md`; post a matching comment on the GitHub issue
- record durable decisions in `docs/project/DECISIONS.md`; reference the GitHub issue that drove the decision
- update `docs/project/INTERFACES.md` before implementing against a new shared contract
- use `docs/project/GITHUB_PROJECTS.md` for `gh` commands
