# Local Data Exposure Policy

## Purpose

This document defines what data is allowed to exist in decrypted form on a client and under what conditions.

For Safe, local decrypted state is one of the main practical security boundaries. The cryptographic model is not enough if plaintext leaks through caches, logs, crash dumps, or long-lived local derivatives.

## Principle

If the client is locked, no usable plaintext should remain.

## While Unlocked

The client may hold decrypted state that is necessary for active use, but it should still minimize retained plaintext.

Allowed while unlocked:

- the currently accessed secret
- temporary working data needed for an active command or UI view
- in-memory material needed for OTP/TOTP generation

Still high sensitivity while unlocked:

- secret values
- decrypted titles and metadata
- URLs
- OTP/TOTP seeds
- any local search index over decrypted data

## At Rest on Disk or Browser Storage

The following must not be stored as plaintext durable state:

- secret values
- OTP/TOTP seeds
- recovery material
- decrypted cache of secret payloads

The following should also be treated as sensitive and encrypted or avoided:

- titles
- usernames
- URLs
- tags
- notes
- search index contents

## Search Index Policy

The local search index is a confidentiality hotspot.

Rules:

- do not treat the search index as harmless metadata
- encrypt it at rest or derive it only while unlocked
- wipe it on sign-out or lock where practical
- rebuild it deterministically when needed

## OTP/TOTP Policy

Rules:

- OTP/TOTP seeds are secret material
- seeds must never be durably stored in plaintext
- code generation should happen in-memory only
- live codes should not be logged, traced, or cached beyond immediate use

## Forbidden Outputs

The following are forbidden:

- plaintext secrets in logs
- plaintext secrets in telemetry
- plaintext secrets in crash reports
- plaintext secrets in debug output
- plaintext secrets in analytics events

The same rule applies to OTP/TOTP seeds and recovery material.

## Lock and Restart Policy

On lock or sign-out:

- clear decrypted caches where practical
- invalidate local unlocked state
- require a fresh unlock for subsequent access

On restart:

- do not rely on surviving plaintext state
- require the normal unlock path

## v1 Guidance

For v1, prefer correctness and clear boundaries over aggressive convenience:

- keep the decrypted working set small
- keep durable local plaintext at zero wherever practical
- defer convenience features that require broad local decrypted indexing if their security handling is still ambiguous
