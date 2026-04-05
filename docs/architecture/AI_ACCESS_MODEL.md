# AI Access Model

## Purpose

This document defines the first-class AI access model for Safe.

Safe is meant to serve both humans and AI, but AI access is not a shortcut around the core trust model. The same design principle still applies:

- decryption happens locally
- humans remain the authorization source
- the service does not get plaintext secrets in normal operation

## Product Position

Safe is not only a password manager. It is a local-first, zero-knowledge secret runtime for humans and AI.

That means AI access is a core product requirement, not a future add-on.

## Core Rules

- AI never gets broad implicit access to the vault.
- AI access must be explicitly approved by a human.
- AI access must be scoped.
- AI access should be time-bound where practical.
- AI access must be auditable.
- The CLI is the primary v1 surface for AI access.

## Mental Model

The vault is not handed to the AI.

Instead, a human-authorized local runtime may release a narrow answer to an AI request, such as:

- one secret value
- one OTP/TOTP seed-derived code
- one small set of secrets in a named scope
- one bounded collection read

The runtime should reveal the minimum needed for the approved action.

## v1 Approach

The v1 AI path should stay simple.

1. AI requests access through the CLI or a CLI-mediated local runtime.
2. The local runtime presents the request to the user.
3. The user approves or rejects the request.
4. If approved, the runtime decrypts locally.
5. The runtime returns only the approved secret material.

## Scope

At minimum, an AI access request should declare:

- requester identity or local process context
- requested secret or collection scope
- intended action
- requested duration if the access is not immediate one-shot

For v1, one-shot access is preferable to reusable delegated access.

## Out of Scope for v1

- background unattended vault access
- long-lived delegated AI tokens
- broad vault export for model context
- silent approval flows
- server-side secret brokering

## Relationship to Later Work

Future versions may add:

- reusable approval policies
- short-lived delegated machine credentials
- richer audit logs
- scoped automation collections

But v1 should prove the narrow, explicit, human-approved model first.
