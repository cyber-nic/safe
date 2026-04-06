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

### 2026-04-06 - Engineer1 completion note (W17)

Task:

- `W17 - Deliver a two-device single-user sync smoke path`

Status:

- completed; refs #37 — https://github.com/cyber-nic/safe/pull/46

Files touched:

- `apps/web/src/server.ts`
- `apps/web/test/client-surface.test.mjs`
- `cmd/safe/main.go`
- `cmd/safe/main_test.go`
- `docs/project/WORKBOARD.md`
- `docs/project/HANDOFFS.md`

Outcome:

- the CLI now ships `sync push` and `sync pull` against the merged S3-backed object-store adapter
- the CLI persists local device signing material, uploads device records for signed-head verification, and proves restart-safe two-device sync in tests
- the proof path rejects tampered signed heads during `sync pull`
- the web client consumes the W16 account-access route during identify and surfaces the granted remote prefix and actions

Verification:

- `go test ./cmd/safe/...`
- `go test ./...`
- `pnpm --filter @safe/web test`

Next action:

- mark GitHub issue `#37` and the project item done; carry any residual web TypeScript cleanup as separate follow-up work rather than hidden W17 scope

### 2026-04-06 - Engineer1 progress note (W17)

Task:

- `W17 - Deliver a two-device single-user sync smoke path`

Status:

- in progress; refs #37

Files requested:

- `apps/web/**`
- `cmd/safe/**`
- `docs/project/**` for milestone closeout only if needed

Files touched:

- `apps/web/src/server.ts`
- `apps/web/test/client-surface.test.mjs`
- `cmd/safe/main.go`
- `cmd/safe/main_test.go`

Outcome:

- the web client now requests account-scoped remote access during identify and surfaces the granted prefix and actions in the unlocked vault view
- the CLI now requests account-scoped remote access during bootstrap and records the granted prefix, actions, and expiry in its bootstrap summary
- both shipped clients now consume the merged W16 capability endpoint instead of treating control-plane access mediation as doc-only
- the CLI now ships `sync push` and `sync pull` against the merged S3-backed `ObjectStoreWithCAS`, persists local device signing material, and proves restart-safe two-device sync in tests
- the W17 proof path now explicitly rejects tampered signed heads during `sync pull`

Next action:

- open the W17 PR and hand review over with the automated two-device CLI sync proof plus the web access-path consumption updates

### 2026-04-06 - Engineer1 completion note (W16)

Task:

- `W16 - Implement minimal control-plane access mediation for single-account sync`

Status:

- completed; refs #32 — https://github.com/cyber-nic/safe/pull/43

Files touched:

- `docs/project/INTERFACES.md` (new I9 account-scoped access capability contract)
- `docs/project/WORKBOARD.md`
- `docs/project/HANDOFFS.md`
- `internal/auth/access.go`
- `internal/auth/access_test.go`
- `cmd/control-plane/main.go`
- `cmd/control-plane/main_test.go`

Outcome:

- the control plane now issues short-lived signed account-path access capabilities via `POST /v1/access/account`
- access is bound to `accountId`, `deviceId`, `bucket`, allowed actions, and the exact `accounts/<accountID>/` prefix
- default issuance is `get` plus `put`; `list` remains explicit instead of implied

Next action:

- W17 can consume the merged W16 capability contract and endpoint from `main`

### 2026-04-06 - Engineer1 progress note (W16)

Task:

- `W16 - Implement minimal control-plane access mediation for single-account sync`

Status:

- in progress; refs #32

Files touched:

- `docs/project/INTERFACES.md` (new I9 account-scoped access capability contract)
- `docs/project/WORKBOARD.md` (W16 status and output update)
- `internal/auth/access.go` (new signed access-capability issuance and verification helpers)
- `internal/auth/access_test.go` (new method escalation, prefix overreach, list-by-default, expiry, and device-state tests)
- `cmd/control-plane/main.go` (new `/v1/access/account` issuance endpoint)
- `cmd/control-plane/main_test.go` (control-plane issuance endpoint coverage)

Outcome:

- W16 now has a frozen minimal capability contract for account-scoped object access at `accounts/<accountID>/`
- the control plane issues short-lived signed capabilities bound to `accountId`, `deviceId`, `bucket`, allowed actions, and expiry
- default issuance is `get` plus `put`; `list` remains explicit instead of implied

Next action:

- update GitHub issue `#32` and the project board to match repo state, then open a PR so W17 can consume the frozen W16 access path

### 2026-04-06 - Engineer2 completion note (W15)

Task:

- `W15 - Implement account-path object-store sync and commit protocol`

Status:

- completed; refs #33 — https://github.com/cyber-nic/safe/pull/41

Files touched:

- `internal/storage/cas.go` (new — ObjectStoreWithCAS interface, MemoryObjectStoreWithCAS, ErrCASConflict, ContentETag)
- `internal/sync/writer.go` (new — SyncWriter, CommitSyncMutation with CAS head advancement)
- `internal/sync/reader.go` (new — SyncReader, IncrementalSync)
- `internal/sync/verify.go` (new — VerifyHeadFunc interface, MonotonicVerifyHead stub for W12 integration)
- `internal/sync/sync_test.go` (new — 8 tests: single commit, two-runtime convergence, interrupt safety, stale head rejection, idempotent commit, CAS conflict, empty collection, delete convergence)

W12 integration:

- W15 ships with `VerifyHeadFunc` as an injectable interface
- `MonotonicVerifyHead` passes freshness checks but does NOT verify Ed25519 signatures
- Now that W12 is merged, wire `w12.VerifySignedHead` via `NewSyncWriter(store, verifyFn)` — no W15 changes needed

Next action:

- W15 and W14 are complete; W16 (Engineer1) and W17 (Engineer1) are on the critical path to two-device sync proof

### 2026-04-06 - Engineer1 completion note (W12)

Task:

- `W12 - Implement signed mutable-metadata verification in code`

Status:

- completed; refs #36

Files touched:

- `internal/sync/authenticated.go`, `internal/sync/verify.go`, `internal/sync/writer.go`, `internal/sync/reader.go`, `internal/sync/authenticated_test.go`, `internal/sync/sync_test.go`
- `docs/project/WORKBOARD.md`, `docs/project/INTERFACES.md`, `docs/project/HANDOFFS.md`

Outcome:

- the shared sync path now persists signed collection-head envelopes at the head key instead of unsigned head records
- sync readers and writers verify the signed head against the referenced latest event plus the active authoring device record before trusting or advancing mutable state
- W15 no longer depends on the old monotonic-only stub integration note

Next action:

- W16 and W17 can consume the landed signed-head verification path on `main` without reopening the W12 boundary

### 2026-04-06 - Engineer2 progress note (W15)

Task:

- `W15 - Implement account-path object-store sync and commit protocol`

Status:

- in progress; refs #33

Files touched:

- `internal/storage/cas.go` (new — ObjectStoreWithCAS interface, MemoryObjectStoreWithCAS, ErrCASConflict, ContentETag)
- `internal/sync/writer.go` (new — SyncWriter, CommitSyncMutation with CAS head advancement)
- `internal/sync/reader.go` (new — SyncReader, IncrementalSync)
- `internal/sync/verify.go` (new — VerifyHeadFunc interface, MonotonicVerifyHead stub for W12 integration)
- `internal/sync/sync_test.go` (new — 8 tests: single commit, two-runtime convergence, interrupt safety, stale head rejection, idempotent commit, CAS conflict, empty collection, delete convergence)
- `docs/project/WORKBOARD.md`, `docs/project/HANDOFFS.md` (status updates)

W12 integration note:

- W15 ships with `VerifyHeadFunc` as an injectable interface (W12 integration point)
- Default stub `MonotonicVerifyHead` passes freshness checks but does NOT verify Ed25519 signatures
- When W12 defines the event signature envelope, inject the real verifier via `NewSyncWriter(store, w12VerifyHead)` — no W15 code changes needed

Next action:

- W12 signature verification integration once Engineer1 opens that branch
- Engineer1 review of W15 before merge

### 2026-04-06 - Engineer2 cross-boundary note (W13)

Task:

- `W13 - Freeze device-enrollment contract for account recovery and new devices`

Status:

- completed cross-boundary; refs #35

Reason for cross-boundary edit:

- W14 (my task, #34) is blocked on W13 and W11; no W13 branch was open and Engineer1 had not started
- same pattern as W7 (Engineer2 took W7 cross-boundary to unblock W8)
- Engineer1 must review I8, D12, D13 before W14 merges

Files touched:

- `docs/project/INTERFACES.md` (new I8 — device enrollment contract)
- `docs/project/DECISIONS.md` (new D12 — ECIES for AMK transfer; new D13 — device ID format)
- `docs/project/HANDOFFS.md` (this entry)
- `docs/project/WORKBOARD.md` (W13 and W14 entries updated)

Next action:

- W14 implementation is unblocked on this branch
- Engineer1 should review I8, D12, D13 and flag any contract changes before W14 merges

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
- `docs/project/HANDOFFS.md`

Next action:

- begin the durable local persistence implementation against the frozen interface and decision set

### 2026-04-04 - Engineer1 internal note

Task:

- `W3 - Add password-derived local encryption primitives`

Status:

- completed

Files touched:

- `internal/crypto/local_unlock.go`
- `internal/crypto/local_unlock_test.go`
- `internal/crypto/testdata/local_unlock_fixture.json`
- `internal/domain/account_config.go`
- `internal/storage/store.go`
- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/WORKBOARD.md`
- `docs/project/HANDOFFS.md`

Next action:

- W4 is unblocked once W2 merges; use the accepted account-local unlock record and storage key

### 2026-04-04 - Engineer2 progress note (W2)

Task:

- `W2 - Implement durable local persistence adapter`

Status:

- in progress

Files touched:

- `internal/storage/store.go`
- `internal/storage/store_test.go`

Blocker:

- none

Next action:

- finish restart-safe persistence tests and mutation boundary handling, then request review

### 2026-04-04 - Engineer2 completion note (W2)

Task:

- `W2 - Implement durable local persistence adapter`

Status:

- completed; refs #4 — https://github.com/cyber-nic/safe/pull/10

Files touched:

- `internal/storage/store.go`
- `internal/storage/store_test.go`

Next action:

- Engineer1 review before merge
- W4 remains blocked on W3

### 2026-04-04 - Engineer1 contribution to W4

Task:

- `W4 - Replace CLI starter bootstrapping on the save/read path`

Status:

- merged on `main`; refs #3 — https://github.com/cyber-nic/safe/pull/14

Files touched:

- `cmd/safe/main.go` (durable bootstrap, encrypted secrets, CommitVaultMutation integration, no auto-seed)
- `cmd/safe/main_test.go` (real tests for identify, save, read, reopen)
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

### 2026-04-05 - Engineer1 completion note (W10)

Task:

- `W10 - Deliver a real M1 client surface`

Status:

- completed; refs #22

Files touched:

- `apps/web/src/local-runtime.ts`
- `apps/web/src/server.ts`
- `apps/web/src/main.ts`
- `apps/web/test/client-surface.test.mjs`
- `apps/web/package.json`
- `docs/project/WORKBOARD.md`
- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- `README.md`

Verification:

- `npm test --prefix apps/web`
- `GOCACHE=/tmp/go-build go test ./...`

Outcome:

- `apps/web` now ships a real local client surface with routes for identify, unlock, save, lock, and reopen
- the web client persists account config, collection head, item records, replayable events, and encrypted secret material under the accepted account-local unlock contract
- M1 is complete; remaining browser-native adapter questions stay queued as explicit post-M1 work

Next action:

- close issue `#22` and milestone issue `#1`
- keep W8 and W9 explicit as post-M1 follow-up work

### 2026-04-05 - Engineer1 internal note (W9)

Task:

- `W9 - Freeze signed mutable metadata and rollback rules`

Status:

- completed; refs #18

Files touched:

- `docs/project/INTERFACES.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`
- `docs/project/WORKBOARD.md`
- `docs/architecture/PROTOCOL.md`
- `docs/architecture/SYSTEM_DESIGN.md`

Contract updates:

- froze the mutable metadata families that are mandatory trust anchors: account config, collection heads, snapshot pointers, device metadata, membership state, and invite state
- froze signer ownership, required scope bindings, and monotonic freshness rules for those records
- required clients to reject stale, divergent, or unsigned mutable metadata before replay, snapshot restore, or authorization-sensitive state transitions

Next action:

- future sync and sharing implementation should add authenticated record envelopes and stale-state rejection tests against the frozen W9 contract

### 2026-04-06 - Engineer1 internal note (M2 planning reset)

Task:

- `M2 - Multi-Device Single-User Sync Foundations`

Status:

- planned (`refs #30`)

Files touched:

- `docs/project/WORKBOARD.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`

Next action:

- move the board from closed M1 work to the explicit M2 queue and keep browser-native adapter work out of the critical path until two-device sync is proven

### 2026-04-06 - Engineer1 to Engineer1

Task:

- `W11 - Freeze M2 sync-foundation scope and acceptance`
- `W12 - Implement signed mutable-metadata verification in code`
- `W13 - Freeze device-enrollment contract for account recovery and new devices`

Status:

- assigned (`refs #31`, `#36`, `#35`)

Write scope:

- `docs/project/**`
- `docs/architecture/IMPLEMENTATION_PLAN.md` for wording alignment only
- `docs/architecture/PROTOCOL.md` and `docs/architecture/SYSTEM_DESIGN.md` for contract alignment only
- `internal/sync/**`
- `internal/domain/**` if authenticated-record types are needed
- `packages/test-vectors/**` if shared fixtures are needed

Next action:

- freeze the M2 acceptance boundary first, then land signed mutable-metadata verification and the device-enrollment contract before downstream sync or client wiring starts

### 2026-04-06 - Engineer1 to Engineer2

Task:

- `W14 - Implement device-enrollment primitives and persisted tests`
- `W15 - Implement account-path object-store sync and commit protocol`

Status:

- planned; pending assignment acceptance (`refs #34`, `#33`)

Write scope:

- `internal/crypto/**`
- `internal/domain/**` for enrollment records
- `internal/sync/**`
- `internal/storage/**` if the object-store adapter boundary needs expansion
- `packages/test-vectors/**` if shared fixtures are needed

Do not edit:

- `apps/web/**`
- `cmd/safe/**`
- `cmd/control-plane/**`
- `internal/auth/**`

Next action:

- wait for W12 and W13 to freeze the verification and enrollment boundaries, then implement the persisted enrollment and account-path sync slices against those contracts

### 2026-04-06 - Engineer1 completion note (W11)

Task:

- `W11 - Freeze M2 sync-foundation scope and acceptance`

Status:

- completed; refs #31

Files touched:

- `docs/project/WORKBOARD.md`
- `docs/project/DECISIONS.md`
- `docs/project/HANDOFFS.md`

Next action:

- carry the M2 planning reset through GitHub project status and issue updates, then start W12 and W13 against the accepted M2 boundary
