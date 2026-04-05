# CLI AI Interaction Flow (W9)

## Purpose

Define the exact CLI behavior for AI-driven secret access.

This makes the AI access model concrete and testable.

---

## Core Principle

The CLI is the enforcement point.

All AI access flows through:
- local runtime
- explicit human approval
- minimal data exposure

---

## Example Flow

### Step 1 — AI Request

AI (or local agent) requests:

safe get secret github_token

---

### Step 2 — CLI Intercepts

CLI does NOT immediately return the secret.

Instead it evaluates:
- is runtime unlocked?
- is this interactive?
- is this request pre-approved?

---

### Step 3 — User Prompt

CLI prompts:

"AI is requesting access to:
  - secret: github_token
  - scope: single item
Approve? (y/n)"

---

### Step 4 — User Decision

If denied:
- return error
- no data exposed

If approved:
- decrypt locally
- return only requested value

---

### Step 5 — Output

Return minimal response:

{
  "value": "..."
}

---

## Constraints

- no full vault dumps
- no wildcard access
- no background polling
- no silent approvals

---

## Future Extensions

- approval policies
- short-lived approvals
- audit logs
- session-scoped permissions

---

## v1 Rule

If a request cannot be explained clearly to a human, it must be denied.
