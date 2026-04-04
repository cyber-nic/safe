# Safe Decisions

## Purpose

This file records decisions that should not be rediscovered in code review.

Each decision should include:

- status
- date
- owner
- decision
- rationale
- downstream impact

## Active Decisions

### D1 - Coordination lives in the repo

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- the repo is the primary coordination system for technical execution

Rationale:

- the team is still small
- the critical path is mostly code and interfaces
- keeping planning next to code reduces drift

Impact:

- `docs/WORKBOARD.md` is the active work queue
- `docs/INTERFACES.md` is the contract source of truth
- external PM tools are optional mirrors, not canonical state
- GitHub issues and the GitHub Project are the shared discussion and visibility layer
- `docs/GITHUB_PROJECTS.md` is the CLI access guide

### D2 - Optimize for the first trustworthy local loop

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- current execution priority is the first real local save/read loop, not broader workspace richness

Rationale:

- this matches the implementation plan and current repo gap
- the project has enough fixture-backed surface area already

Impact:

- new work must justify itself against the local-runtime milestone
- non-critical surface expansion should be deferred

### D3 - Ownership is directory-first

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- contributors default to one owned directory tree per task

Rationale:

- this lowers merge conflicts
- it keeps commits small and attributable

Impact:

- cross-boundary edits need an explicit handoff note
- shared contracts should land before consumer wiring

### D4 - Contract-first merge order

Status:

- accepted

Date:

- 2026-04-04

Owner:

- Engineer1

Decision:

- shared contract docs and interface locks land before downstream implementation branches

Rationale:

- Engineer2 and Engineer3 need stable boundaries
- this avoids parallel guessing in `cmd/safe`, `apps/web`, and `internal/storage`

Impact:

- `docs/INTERFACES.md` should be updated before implementation starts on a new shared seam

## Pending Decisions

### P1 - Local persistence backend format

Status:

- pending

Owner:

- Engineer1

Question:

- use a simple file-backed format first or introduce SQLite immediately for the CLI local runtime

Decision driver:

- fastest path to a trustworthy restart-safe loop with low migration cost

### P2 - Local encryption payload format

Status:

- pending

Owner:

- Engineer1

Question:

- exact local envelope format for password-derived encryption, key wrapping, and versioning

Decision driver:

- must support wrong-password and corrupted-payload tests cleanly

### P3 - Web local runtime storage boundary

Status:

- pending

Owner:

- Engineer1

Question:

- how much of the CLI local runtime contract can be shared with the web surface before browser-specific storage adapters diverge

Decision driver:

- avoid inventing a second runtime model
