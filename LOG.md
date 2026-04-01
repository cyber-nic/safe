# Change Log

## 2026-03-31T12:04:45Z

- Added `safe secret search <query>` to the CLI with case-insensitive matching across IDs, titles, usernames, tags, URLs, issuer, and account fields.
- Added CLI tests covering search matches by title and tag, no-result output, and the empty-query matcher edge case.
- Continued the repo's CLI iteration toward the implementation-plan local search milestone.

## 2026-03-31T12:07:01Z

- Added `safe secret update <item-id> <title> <username>` for login items so the CLI now covers a concrete update flow in addition to create, read, and search.
- Refactored CLI write persistence through a shared item-mutation helper used by both `secret add` and `secret update`.
- Added tests for successful updates, missing-item handling, and rejection of non-login updates.

## 2026-03-31T12:09:21Z

- Added first-class `delete_item` vault events plus replay and mutation support so item removal is modeled in the domain instead of being faked in the CLI.
- Added `safe secret delete <item-id>` to the CLI and reused the same replay-driven state model to report the remaining item count after deletion.
- Added domain, sync, and CLI tests covering delete event serialization, replay behavior, mutation construction, and missing-item handling.

## 2026-03-31T12:12:29Z

- Added `safe secret restore <item-id>` so deleted items can be replayed back into the active vault state from their stored item record.
- Fixed CLI mutation helpers to load the latest collection head from storage before writing, which makes chained operations like delete-then-restore sequence correctly.
- Added tests for successful restore after delete, active-item rejection, and missing-version rejection.

## 2026-03-31T12:14:40Z

- Added `safe secret history <item-id>` so the CLI can inspect append-only event history for a specific secret instead of only mutating current state.
- Refactored CLI event loading so history and replayed projection reads share the same collection event source.
- Added tests for starter history output, delete-then-restore history, missing-history handling, and item targeting across put and delete events.

## 2026-03-31T12:18:32Z

- Added `safe secret export [item-id]` so the CLI can emit deterministic JSON for the full active vault projection or a single targeted secret.
- Sorted exported records by item ID to keep automation-friendly output stable across runs.
- Added CLI tests covering full export payloads, single-item export, and missing-item failures.

## 2026-03-31T12:21:32Z

- Added `safe secret import` so the CLI can read exported vault JSON from stdin and replay it back into the active collection through normal put-item mutations.
- Refactored CLI execution to accept stdin explicitly, which keeps command plumbing testable while preserving the existing terminal entrypoint.
- Added CLI tests covering single-item import, full export/import round-trips, deleted-vault restoration, and invalid import payload handling.

## 2026-03-31T12:25:01Z

- Added a global `--json` CLI mode so overview, list, search, show, history, and mutation commands can emit machine-readable payloads for automation while keeping the existing human-readable default output.
- Suppressed the bootstrap banner in JSON mode and centralized pretty JSON encoding through a shared helper.
- Added CLI tests covering JSON overview output, JSON list/show/history responses, and JSON responses for add and import flows.

## 2026-03-31T12:28:33Z

- Extended the Go domain model to validate, summarize, and canonically serialize `note`, `apiKey`, and `sshKey` items so it now matches the broader TypeScript item taxonomy instead of only accepting `login` and `totp`.
- Updated CLI secret rendering and import parsing so those additional item kinds can round-trip through `safe secret import` and `safe secret show` without being rejected or misparsed.
- Added domain and CLI tests covering the new item-kind validation paths plus note, API key, and SSH key import/show/search flows.

## 2026-04-01T08:58:27Z

- Added `delete_item` support to the TypeScript SDK event model so TS protocol parsing and canonical serialization now match the Go domain and CLI behavior.
- Added `createDeleteItemEventRecord` plus branching delete-event handling in `parseVaultEventRecord` and `serializeCanonicalVaultEventRecord`.
- Added a lightweight Node test suite for the TS SDK covering delete-event creation, parsing, canonical serialization, and regression coverage for existing put-item serialization.

## 2026-04-01T09:00:55Z

- Added a shared `delete-event-record.json` protocol fixture under `packages/test-vectors` so delete events now have the same language-neutral fixture coverage as put-item records.
- Exported parsed and canonicalized delete-event vectors from the test-vectors package for downstream consumers.
- Updated the Go delete-event canonicalization test to compare against the shared fixture instead of a hardcoded JSON string.

## 2026-04-01T09:09:53Z

- Added a lightweight Node test suite for `@safe/test-vectors` so the shared vector package now verifies its exported starter items, canonical item records, starter events, and delete-event vector directly.
- Updated the test-vectors source entrypoint to use Node-compatible JSON import attributes and source-to-source TypeScript imports so the package can be executed from source in local tests.
- Added a `test` script to `packages/test-vectors/package.json` to make the vector-package checks directly runnable with `pnpm --filter @safe/test-vectors test`.

## 2026-04-01T09:23:24Z

- Updated `IMPLEMENTATION_PLAN.md` to record the current repository status explicitly: useful CLI and protocol progress, but continued gaps on crypto, encrypted local persistence, signed mutable metadata, and rollback handling.
- Updated the plan's phase notes to clarify that foundations are mostly in place, data-model work is ahead of crypto, and the local-vault-runtime phase has not started in the sense intended by the docs.
- Updated `README.md` so the repo status and next steps now point at signer and rollback rules, key hierarchy work, and durable encrypted local persistence instead of the original scaffolding checklist.

## 2026-04-01T09:27:11Z

- Added sync-side head verification so replay can now assert that a collection head's latest sequence and latest event actually match the appended event log instead of trusting mutable head state blindly.
- Added an explicit monotonic trusted-head helper that rejects stale heads and same-sequence different-event mismatches as the first concrete rollback-detection primitive in the sync package.
- Wired the CLI through verified head-plus-event loading and added tests covering mismatched head rejection on both read and write paths.

## 2026-04-01T09:29:47Z

- Added `CollectionHeadRecord` and `AccountConfigRecord` parsing and canonical serialization helpers to the TypeScript SDK so mutable metadata records now have TS-side protocol support alongside the Go sync package.
- Added `ensureMonotonicHead` to the TypeScript SDK to mirror the new Go rollback-detection primitive for stale heads and same-sequence divergent heads.
- Extended the TS SDK Node test suite with coverage for canonical collection-head and account-config records plus monotonic-head acceptance and rejection cases.

## 2026-04-01T09:32:19Z

- Added shared `collection-head-record.json` and `account-config-record.json` fixtures under `packages/test-vectors` so mutable metadata records now have the same language-neutral vector coverage as vault items and events.
- Exported parsed and canonicalized collection-head and account-config records from the test-vectors package and extended its Node test suite to verify both forms.
- Updated Go domain canonicalization tests to compare against the shared mutable-metadata fixtures instead of hardcoded JSON literals.

## 2026-04-01T09:36:19Z

- Added TypeScript SDK replay helpers so clients can sort, validate, and project vault event streams with the same sequence-gap, mixed-collection, delete-item, and head-alignment semantics as the Go sync package.
- Added `buildPutItemMutation` and `buildDeleteItemMutation` to the TypeScript SDK so TS clients can generate deterministic event IDs and next-head records without reimplementing the sync mutation rules.
- Extended the TS SDK Node test suite with replay, replay-against-head, and mutation-builder coverage to keep the new sync helpers aligned with Go behavior.

## 2026-04-01T09:39:42Z

- Added shared sync-mutation fixtures under `packages/test-vectors` for the deterministic starter put mutation item/event/head and the corresponding delete-mutation head.
- Exported parsed and canonicalized mutation fixtures from the test-vectors package and extended its Node tests to validate those new vectors.
- Updated Go sync tests to consume the shared mutation fixtures for replay-delete and mutation-builder expectations instead of hardcoded event and head literals.

## 2026-04-01T09:46:48Z

- Added a consumer-facing `safe secret code <item-id>` CLI command for TOTP items so the starter runtime can now generate 6-digit authenticator codes locally instead of only listing authenticator metadata.
- Added reusable SHA1 TOTP generation under `internal/crypto` plus local secret-material storage helpers under `internal/storage`, which gives the CLI a real seed-resolution path that can later move behind encrypted local persistence instead of hardcoded demo logic.
- Added Go tests covering RFC-style TOTP generation, secret-material storage, human and JSON code output, and failure cases for missing secret material or non-TOTP items.

## 2026-04-01T09:49:55Z

- Added a consumer-facing `safe secret add-totp <title> <issuer> <account-name> <secret-base32>` command so users can now create authenticator entries through the CLI instead of relying on starter fixtures or raw import payloads.
- Wired `add-totp` through the same local secret-material storage path and event-log mutation flow as the existing vault commands, so the new authenticator setup path remains aligned with the planned local-vault runtime rather than introducing demo-only state.
- Added CLI tests covering human and JSON `add-totp` output, immediate `secret show` and `secret code` use of newly added authenticators, and invalid-secret rejection.

## 2026-04-01T10:02:31Z

- Replaced the web app's placeholder bootstrap object with a real replay-backed `createVaultWorkspace` model that exposes overview stats, searchable vault items, authenticator cards, and recent activity derived from account config, collection head, and event records.
- Added durable consumer-facing web behaviors for text, kind, and tag filtering plus authenticator-to-login linking, so the primary web surface can start gathering product feedback without inventing throw-away demo state outside the existing protocol and sync model.
- Added a lightweight `@safe/web` Node test suite covering the starter workspace, search and filter behavior, delete-event activity rendering, and collection-head consistency checks.

## 2026-04-01T10:14:12Z

- Added browser-compatible TOTP generation to the TypeScript SDK via Web Crypto, including RFC-style code generation and window metadata helpers for `totp` vault items so the web stack can produce local authenticator codes instead of only listing authenticator records.
- Exported shared starter secret material from `@safe/test-vectors` and wired the web workspace through `unlockVaultWorkspace` and `createUnlockedVaultWorkspace`, which upgrades authenticator cards from locked metadata to live local codes when secret material is available without introducing server-side or demo-only state.
- Extended the TS SDK, test-vectors, and web Node test suites with coverage for TOTP generation, starter secret material, unlocked authenticator cards, and locked-without-secret behavior.

## 2026-04-01T10:21:37Z

- Added replay-backed web mutation helpers for local-first vault writes: `addLoginToVaultWorkspace`, `addTotpToVaultWorkspace`, and `deleteItemFromVaultWorkspace`, all of which generate normal event-log mutations and rebuild the workspace from the updated head and events instead of mutating ad hoc UI state.
- Extended the web workspace model to retain the trusted replay inputs it was built from, which gives future UI code a stable path for local writes, filtering continuity, and authenticator refresh after mutations.
- Added web tests covering login creation, authenticator creation with secret-material carry-forward and immediate code generation, and delete flows that remove TOTP secret material alongside the deleted item.

## 2026-04-01T10:33:52Z

- Added consumer-facing web item recovery helpers: `getVaultItemDetail`, `listDeletedVaultItems`, and `restoreItemToVaultWorkspace`, so the web client can inspect per-item history, surface deleted records, and replay restores through the normal event log instead of relying on out-of-band state.
- Extended the web workspace model with reusable item-history and latest-record helpers, which keeps detail views, deleted-item listings, and restore behavior aligned with the same replayed event source that powers the active vault view.
- Added web tests covering active item detail history, deleted-item discovery, successful restore after delete, and rejection paths for restoring active or unknown items.

## 2026-04-01T10:43:16Z

- Added replay-backed web edit helpers for active items: `updateLoginInVaultWorkspace` and `updateTotpInVaultWorkspace`, so the local-first web client can modify existing logins and authenticators through normal put-item events instead of replacing state out of band.
- Refactored web write handling through a shared `persistUpdatedItem` helper, which keeps add and update flows aligned on the same mutation, rebuild, and authenticator-refresh path.
- Added web tests covering login edits, TOTP metadata and secret rotation, and rejection paths for attempting to update the wrong item kind.

## 2026-04-01T10:52:48Z

- Added consumer-facing web export and import helpers: `exportVaultWorkspace`, `serializeVaultExportPayload`, and `importVaultWorkspace`, so the local-first client can produce deterministic vault payloads and replay them back through normal put-item mutations instead of bypassing the event log.
- Included TOTP secret-material carry-through in web export and import payloads for matching authenticator items, which keeps portability aligned with the existing built-in authenticator behavior instead of exporting incomplete records.
- Added web tests covering full-vault export ordering, single-item export, and importing an exported item back into a fresh workspace through replay-backed sequence advancement.
