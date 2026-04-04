# Safe Handoffs

## Purpose

This file is the handoff log between engineers.

GitHub companion workflow:

- each tracked task also has a GitHub issue for discussion
- add a matching issue comment when a handoff happens
- use `docs/GITHUB_PROJECTS.md` for `gh` commands
- directed GitHub comments must include explicit `From`, `To`, and `Via` headers

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

- `docs/WORKBOARD.md`
- `docs/INTERFACES.md`

Blocker policy:

- if the persistence contract is insufficient, record the missing contract detail here and stop

### 2026-04-04 - Engineer1 internal note

Task:

- `W1 - Freeze local-runtime contract`

Status:

- in progress

Next action:

- finalize the first persistence and unlock decisions before wiring `cmd/safe`
