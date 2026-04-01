# Safe

Safe is a zero-knowledge secret manager for individuals and small trusted groups. It keeps encryption and key access on the client, uses a thin control plane for identity and authorization, and stores durable encrypted state in S3-compatible object storage. The product is designed to be local-first, offline-capable, and portable across a web app, CLI, and future client surfaces.

## MVP

The v1 product is a secure personal vault with small-group sharing.

The first product loop that matters is narrower than the full v1 scope:

- sign in
- save a secret
- read that secret back safely

If that loop is not working in a real client, additional surface area should be treated as secondary.

Primary user outcomes:

- Create and manage secrets in a primary web app
- Unlock locally with a master password
- Sync encrypted state across devices through S3-backed storage
- Generate 6-digit OTP/TOTP codes locally for websites and apps that require 2FA
- Share collections with a small number of trusted users
- Revoke members for future access through key rotation
- Recover access using a recovery key without server-side escrow

## Primary Users

- Individuals managing personal credentials, OTP/TOTP authenticators, API keys, notes, and SSH keys
- Small trusted groups such as families or tiny teams sharing a limited set of secrets

## Must Be True in v1

- The service cannot decrypt vault contents in normal operation
- A locked client does not expose usable plaintext state
- Sync is deterministic and resilient to partial failure
- Device addition is explicit and auditable
- Sharing is collection-based, not fine-grained
- Revocation protects future data, not historical plaintext already seen

## Not in v1

- Enterprise administration
- Real-time collaboration
- Browser extension autofill
- Server-side search
- Fine-grained per-field permissions
- Recovery without a recovery key
- Broad automation or agent-native secret access
