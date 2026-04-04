# Safe Handoffs

## Purpose

This file is the handoff log between engineers.

GitHub is the primary handoff surface:

- post a comment on the relevant GitHub issue before or at the same time as adding an entry here
- update the GitHub Project item status to reflect the new owner or blocked state
- use `docs/project/GITHUB_PROJECTS.md` for `gh` commands
- directed GitHub comments must include explicit `From`, `To`, and `Via` headers

This file is a repo-side record of handoffs that affect write scope or contracts. Do not use it as a substitute for GitHub communication.

Use short entries only. Each entry should say:

- date
- from
- to
- task
- status
- files touched or requested
- blocker or next action

Current planning issues:

- `#1` milestone
- `#6` W1
- `#4` W2
- `#5` W3
- `#3` W4
- `#2` W5
- `#21` W6
- `#20` W7
- `#19` W8
- `#18` W9

### 2026-04-04 - Engineer1 PM queue expansion

Task:

- create post-W5 queued work items for milestone closeout and the next contract-first security slices

Status:

- completed; created `#21`, `#20`, `#19`, and `#18` and added them to the GitHub project

Next action:

- use W6 to close out M1 cleanly
- keep W7 before W8 so recovery implementation lands against a frozen contract
- keep W9 contract-first before broader sync implementation expands

## Entries

### 2026-04-04 - Engineer1 to Engineer2

Task:

- `W2 - Implement durable local persistence adapter`

Status:

- assigned

Write scope:

- `internal/storage/**`
- optional new package under `internal/**` if required by the adapter

Do not edit:

- `cmd/safe/**`
- `apps/web/**`
- `packages/ts-sdk/**`
- `packages/test-vectors/**`

Next action:

- implement restart-safe local persistence for account config, collection head, event records, and secret material

Contract references:

- `docs/project/WORKBOARD.md`
- `docs/project/INTERFACES.md`

Blocker policy:

- if the persistence contract is insufficient, record the missing contract detail here and stop

### 2026-04-04 - Engineer1 internal note

Task:

- `W1 - Freeze local-runtime contract`

Status:

- completed

Next action:

- hand the accepted persistence and unlock contract to Engineer2 for W2 and to Engineer1 for W3 or W4

### 2026-04-04 - Engineer1 to Engineer2

Task:

- `W2 - Implement durable local persistence adapter`

Status:

- contract frozen

Write scope reminder:

- `internal/storage/**`
- optional new package under `internal/**` if required by the adapter

Contract updates:

- persist five durable units: account config, collection head, vault item records, vault event records, and secret material
- keep the existing logical key layout from `internal/storage`
- do not depend on backend filename ordering for event load order; return events in ascending `sequence`
- provide a durable mutation boundary so a new collection head is not exposed without its matching records
- do not synthesize starter fixtures during initialization

Files:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/WORKBOARD.md`

Next action:

- implement the durable adapter and restart-survival tests against the frozen contract

### 2026-04-04 - Engineer1 internal note

Task:

- `W3 - Add password-derived local encryption primitives`

Status:

- completed (`refs #5`)

Files:

- `internal/crypto/**`
- `internal/domain/unlock.go`
- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/WORKBOARD.md`

Contract updates:

- froze `accounts/<accountID>/unlock.json` as the account-scoped unlock metadata path
- froze the `LocalUnlockRecord` Argon2id plus AES-256-GCM schema for password-derived account unlock
- froze the versioned AES-256-GCM JSON envelope used for encrypted secret material

Next action:

- W4 can now wire `cmd/safe/**` to the durable adapter and real unlock flow without inventing new local crypto formats

### 2026-04-04 - Engineer2 to Engineer1

Task:

- `W2 - Implement durable local persistence adapter`

Status:

- completed; refs #4

Files touched:

- `internal/storage/file_store.go` (new — FileObjectStore, atomic writes, path-mapped keys)
- `internal/storage/file_store_test.go` (new — restart-survival tests for all five persisted units)
- `internal/storage/store.go` (updated — VaultMutation + CommitVaultMutation)
- `internal/storage/store_test.go` (updated — CommitVaultMutation tests including partial-failure proof)

Next action:

- W4 is unblocked (W2 + W3 complete)
- Engineer2 awaits next assignment

### 2026-04-04 - Engineer2 contribution to W4

Task:

- `W4 - Replace CLI starter bootstrapping on the save/read path`

Status:

- merged on `main`; refs #3 — https://github.com/cyber-nic/safe/pull/15

Files touched:

- `cmd/safe/main.go` (durable bootstrap, encrypted secrets, CommitVaultMutation, genesis head handling)
- `cmd/safe/main_test.go` (withBootstrapRuntime, withFakeBootstrap, withEmptyBootstrap, restart and encryption tests)
- `internal/storage/store.go` (added IsObjectNotFound — type-safe helper replacing local string-prefix check)

Next action:

- W5 is unblocked
- Engineer2 awaits next assignment

### 2026-04-04 - Engineer1 internal note

Task:

- `W5 - Minimal web runtime integration`

Status:

- completed; refs #2

Files touched:

- `docs/project/WORKBOARD.md`
- `apps/web/**`

Next action:

- M1 local-runtime surfaces now align on account config plus optional head plus event replay; next coordination step is milestone closeout and any final polish outside W5
