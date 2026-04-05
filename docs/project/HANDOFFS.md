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
- `#22` W10
- `#20` W7
- `#19` W8
- `#18` W9

## Entries

### 2026-04-05 - Engineer1 to Engineer1

Task:

- `W6 - M1 closeout audit and release checklist`

Status:

- assigned (`refs #1`)

Write scope:

- `docs/project/WORKBOARD.md`
- `docs/project/HANDOFFS.md`
- `docs/project/DECISIONS.md`
- release-readiness notes in `docs/architecture/IMPLEMENTATION_PLAN.md` only

Next action:

- reconcile milestone docs against GitHub issue and project statuses, then produce a closeout checklist and identified follow-up issue set

### 2026-04-05 - Engineer1 to Engineer2

Task:

- `W7 - Post-M1 UX and reliability backlog definition`

Status:

- planned; pending assignment acceptance (`refs #1`)

Write scope:

- GitHub planning issues and labels
- summary note in `docs/project/HANDOFFS.md`

Do not edit:

- `cmd/safe/**`
- `apps/web/**`
- `internal/storage/**`
- `internal/crypto/**`

Next action:

- draft and tag post-M1 backlog issues (`area/*`, `type/*`, `priority/*`) after W6 publishes the closeout checklist

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

### 2026-04-05 - Engineer2 completion note (W8)

Task:

- `W8 - Implement recovery-key wrap and recovery tests`

Status:

- completed; refs #19 — https://github.com/cyber-nic/safe/pull/26

Files touched:

- `internal/domain/recovery.go` (new — LocalRecoveryRecord, Validate, CanonicalJSON, ParseLocalRecoveryRecordJSON)
- `internal/crypto/local_recovery.go` (new — GenerateRecoveryKey, RecoveryKeyMnemonic, CreateLocalRecoveryRecord, OpenLocalRecoveryRecord)
- `internal/crypto/local_recovery_test.go` (new — round-trip, wrong key, corrupted ciphertext, wrong account ID, on-disk fixture)
- `internal/crypto/testdata/recovery_fixture.json` (new — serialized fixture for restart-survival test)
- `go.mod`, `go.sum` (added github.com/tyler-smith/go-bip39 v1.1.0)

Next action:

- Engineer1 review before merge
- Engineer2 awaits next assignment

### 2026-04-05 - Engineer2 cross-boundary note (W7)

Task:

- `W7 - Freeze recovery-key account contract`

Status:

- in progress; refs #20

Reason for cross-boundary edit:

- W7 write scope is normally Engineer1's; Engineer2 is taking it because W8 is blocked and no W7 branch was open
- cross-boundary edit noted here per D3; Engineer1 should review before W8 implementation starts

Files touched:

- `docs/project/INTERFACES.md` (new I6 — recovery-key contract)
- `docs/project/DECISIONS.md` (new D8 — no KDF on recovery key)
- `docs/project/WORKBOARD.md` (W7 and W8 task entries added; Engineer2 status updated)
- `docs/project/HANDOFFS.md` (this entry)

Next action:

- W8 is unblocked once W7 merges

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

### 2026-04-05 - Engineer1 W6 verification note

Task:

- `W6 - Close out M1 local-runtime loop`

Status:

- completed; verification run complete, milestone still open

Files reviewed:

- `cmd/safe/**`
- `apps/web/**`
- `docs/project/WORKBOARD.md`
- `docs/project/INTERFACES.md`
- `README.md`

Verification:

- `GOCACHE=/tmp/go-build go test ./...`
- `npm test --prefix apps/web`
- `npm run check --prefix apps/web` fails in this environment because `tsc` is not installed on `PATH`

Finding:

- the CLI path and web runtime helpers are real and test-backed
- `apps/web` still contains only package code plus tests, not a navigable authenticated client surface

Next action:

- record M1 as still open for the missing client-surface requirement
- close W5 cleanly and queue the missing client-surface slice as a separate task
