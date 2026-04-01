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
