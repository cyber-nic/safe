# Communication

## Purpose

This file defines how engineers should raise project-management feedback, coordination issues, and process problems.

This is for process communication, not implementation details.

Use task issues for task work.
Use this process when the problem is about planning, ownership, sequencing, handoffs, scope, or collaboration quality.

## Who Receives PM Feedback

Project manager:

- Engineer1
- current agent identity is defined in `.whoami`

At the moment, that is the PM contact for:

- unclear ownership
- conflicting write scopes
- blocked handoffs
- missing contracts
- wrong milestone priority
- GitHub Project confusion
- documentation drift between `docs/` and GitHub
- process improvements

## Preferred Channels

Use these in order:

1. Comment on the milestone issue or the relevant planning issue in GitHub.
2. Add a short note to `docs/HANDOFFS.md` if the issue affects ownership or handoff flow.
3. Update `docs/WORKBOARD.md` only if the PM has agreed the plan should change.

Do not use `LOG.md` for coordination feedback.

## Comment Format

When sending PM or process feedback through GitHub, use this structure:

```text
[Agent: <your name>]
From: <your name>
To: <PM name from .whoami>
Via: cyber-nic GitHub issue comment
Type: Process
Subject: <short topic>

Context:
- what happened

Impact:
- what is blocked, confused, duplicated, or risky

Request:
- what decision, clarification, or change is needed
```

Keep it short and specific.

## When To Escalate

Raise PM feedback immediately if:

- two engineers need the same files
- a task depends on an undefined interface
- the GitHub issue and `docs/WORKBOARD.md` disagree
- ownership is ambiguous
- the current milestone no longer matches reality
- a task should be split, merged, or reassigned
- communication with another engineer is stalled

## Expected PM Response

The PM should respond by doing one or more of these:

- clarify ownership
- update `docs/WORKBOARD.md`
- update `docs/INTERFACES.md`
- record a decision in `docs/DECISIONS.md`
- add a handoff note in `docs/HANDOFFS.md`
- change the GitHub Project or issue status

## Scope Boundary

Use this file for:

- process
- planning
- coordination
- handoffs
- escalation

Do not use this file for:

- code design discussion inside a task
- implementation status that belongs on the task issue
- low-level debugging notes
