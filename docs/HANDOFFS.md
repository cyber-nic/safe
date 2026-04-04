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

- `docs/INTERFACES.md`
- `docs/DECISIONS.md`
- `docs/WORKBOARD.md`

Next action:

- implement the durable adapter and restart-survival tests against the frozen contract
