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
