# M1 Acceptance Criteria

## Purpose

Define a single, unambiguous test for whether the product is real.

---

## The Test

A developer must be able to:

1. Run CLI
2. Initialize account
3. Set password
4. Save a secret
5. Exit process
6. Restart CLI
7. Unlock
8. Read the same secret

---

## Failure Cases (REQUIRED)

- Wrong password → fails
- Corrupted storage → fails safely
- Partial write → does not expose broken state

---

## Non-Negotiable Rules

- No in-memory fallback
- No starter fixtures
- No hidden bootstrap

---

## Definition of Done

- This flow works every time
- Covered by automated test
- Reproducible by any engineer

---

## Anti-Goal

If this does not work, no other feature matters.
