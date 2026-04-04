# GitHub Project Access

## Purpose

This file explains how to access Safe planning artifacts in GitHub with the `gh` CLI.

Do not store credentials in this repo.

## Canonical Sources

Use both of these:

- repo-side technical source of truth:
  `docs/WORKBOARD.md`, `docs/INTERFACES.md`, `docs/DECISIONS.md`, `docs/HANDOFFS.md`
- GitHub-side collaboration surface:
  GitHub issues plus the GitHub Project board

Rule:

- repo docs define technical contracts and write scopes
- GitHub issues and project items are the shared discussion and visibility layer

## Prerequisites

You need:

- `gh` installed
- GitHub access to `cyber-nic/safe`
- `gh` authenticated with `repo` and `project` scope

Check auth:

```bash
gh auth status
```

If Project access is missing:

```bash
gh auth refresh -h github.com -s project
```

## Useful Targets

Repository:

```bash
gh repo view cyber-nic/safe
gh issue list --repo cyber-nic/safe
```

Project board:

```bash
gh project list --owner "@me"
gh project view 1 --owner "@me"
gh project field-list 1 --owner "@me"
```

## Working With Planning Issues

List planning issues:

```bash
gh issue list --repo cyber-nic/safe
```

Open an issue in the browser:

```bash
gh issue view <number> --repo cyber-nic/safe --web
```

Comment on an issue:

```bash
gh issue comment <number> --repo cyber-nic/safe --body "Update goes here"
```

Edit an issue:

```bash
gh issue edit <number> --repo cyber-nic/safe --title "New title"
```

Close an issue:

```bash
gh issue close <number> --repo cyber-nic/safe
```

Reopen an issue:

```bash
gh issue reopen <number> --repo cyber-nic/safe
```

## Working With The Project

List items:

```bash
gh project item-list 1 --owner "@me"
```

Add an existing repo issue to the project:

```bash
gh project item-add 1 --owner "@me" --url https://github.com/cyber-nic/safe/issues/<number>
```

Project field IDs and option IDs can be discovered with:

```bash
gh project field-list 1 --owner "@me" --format json
```

Set an item status:

```bash
gh project item-edit \
  --id <item-id> \
  --project-id <project-id> \
  --field-id <status-field-id> \
  --single-select-option-id <option-id>
```

## Recommended Workflow

1. Read `docs/WORKBOARD.md` before starting work.
2. Find the matching GitHub issue for your task.
3. Comment on the issue when you start, block, hand off, or finish work.
4. Keep technical contract changes in repo docs, not only in issue comments.
5. Update the GitHub Project status to match the repo workboard.

## Current Conventions

- milestone work is represented by a GitHub issue
- each tracked work item has its own GitHub issue
- comments belong on issues
- the GitHub Project is for status and visibility, not for storing technical contracts
