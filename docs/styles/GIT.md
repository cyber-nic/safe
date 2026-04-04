# Git Conventions

## Purpose

This file defines preferred branch naming and commit message formats for this repository.

Follow these conventions consistently so that history is readable across human and agent contributors.

## Branch Names

### Format

```
<milestone>-<short-description>
```

- `milestone` — the workboard milestone this branch targets (e.g. `w1`, `w2`, `m3`)
- `short-description` — kebab-case summary of the work, 3–5 words

### Examples

```
w2-vault-unlock-flow
w1-local-runtime-contract
w2-local-persistence
```

### Rules

- do not include agent names, engineer handles, or any personal identifiers in branch names
- always include the milestone — it anchors the branch to planned work
- use kebab-case only, no underscores or slashes
- keep the description short and specific to the task, not the milestone

## Commit Messages

### Format

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <short summary>
```

- `type` — what kind of change (see below)
- `scope` — optional, the affected subsystem or package
- `short summary` — imperative mood, lowercase, no trailing period, under 72 characters

### Types

| Type       | When to use                                          |
|------------|------------------------------------------------------|
| `feat`     | a new capability visible to users or other systems   |
| `fix`      | corrects a bug or broken behavior                    |
| `docs`     | documentation only, no code change                   |
| `refactor` | code restructure with no behavior change             |
| `test`     | adds or corrects tests                               |
| `chore`    | tooling, deps, config — nothing that ships           |
| `build`    | changes to build system, CI, or container setup      |

### Scopes

Use the name of the package or subsystem being changed:

- `cli` — command-line interface
- `web` — web client
- `api` — control-plane API
- `proto` — shared protocol models
- `crypto` — cryptographic primitives or key handling
- `infra` — infrastructure, compose, deployment
- `plan` — planning documents under `docs/project/`

Omit the scope when the change is truly cross-cutting.

### Examples

```
feat(cli): add totp authenticator setup command
feat(web): add replay-backed vault export and import
fix(crypto): correct key derivation round count
docs: add process communication guide for engineers
docs(plan): freeze local runtime contract for M1
refactor(api): extract session middleware into separate package
chore: upgrade Go toolchain to 1.24
```

### Rules

- use imperative mood: "add", not "added" or "adds"
- do not capitalize the summary or end it with a period
- keep the subject line under 72 characters
- add a body if the commit needs context beyond the summary — separate it from the subject with a blank line
- do not reference issue numbers in the subject; use the body or PR description instead

## Pull Requests

- PR title should match the commit message format of the squashed or primary commit
- PR description should reference the relevant GitHub issue and workboard milestone
- one PR per milestone task — do not mix unrelated changes
- merge target is always `main`
