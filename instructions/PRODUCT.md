# Cipher

Cipher is a zero-knowledge secret manager for individuals and small trusted groups. It keeps encryption and key access on the client, uses a thin control plane for identity and authorization, and stores durable encrypted state in S3-compatible object storage. The product is designed to be local-first, offline-capable, and portable across desktop, browser extension, CLI, and future mobile clients.

## MVP

The v1 product is a secure personal vault with small-group sharing.

Primary user outcomes:

- Create and manage secrets on a primary desktop client
- Unlock locally with a master password
- Sync encrypted state across devices through S3-backed storage
- Use a browser extension for conservative lookup and autofill
- Share collections with a small number of trusted users
- Revoke members for future access through key rotation
- Recover access using a recovery key without server-side escrow

## Primary Users

- Individuals managing personal credentials, API keys, notes, and SSH keys
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
- Server-side search
- Fine-grained per-field permissions
- Recovery without a recovery key
- Broad automation or agent-native secret access
