# Product Clarity (PM Override)

## What Safe IS

Safe is NOT primarily a password manager.

Safe is:

> A local-first, zero-knowledge secret runtime for humans and AI.

Core properties:

- All secrets are encrypted client-side
- The runtime (CLI first) is the trust boundary
- Object storage is dumb persistence
- Control plane is authorization only
- AI access is explicit, scoped, and human-approved

---

## First Users

1. Single user (developer)
2. User + spouse (trusted pair)
3. Public users

---

## Core Differentiation

- Local-first trust model
- Portable encrypted state (S3-backed)
- CLI-native access for AI
- Explicit authorization model for non-human actors

---

## What Safe is NOT (v1)

- Not a full UI-first password manager
- Not browser-autofill-first
- Not enterprise secrets platform

---

## The Only Loop That Matters (M1)

1. Initialize account
2. Create unlock metadata
3. Save encrypted secret
4. Restart process
5. Unlock
6. Read secret

If this loop is not rock-solid, the product is not real.
