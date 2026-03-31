# Change Log

## 2026-03-31T12:04:45Z

- Added `safe secret search <query>` to the CLI with case-insensitive matching across IDs, titles, usernames, tags, URLs, issuer, and account fields.
- Added CLI tests covering search matches by title and tag, no-result output, and the empty-query matcher edge case.
- Continued the repo's CLI iteration toward the implementation-plan local search milestone.

## 2026-03-31T12:07:01Z

- Added `safe secret update <item-id> <title> <username>` for login items so the CLI now covers a concrete update flow in addition to create, read, and search.
- Refactored CLI write persistence through a shared item-mutation helper used by both `secret add` and `secret update`.
- Added tests for successful updates, missing-item handling, and rejection of non-login updates.
