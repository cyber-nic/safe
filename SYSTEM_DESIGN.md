# Secret Manager — System Design Plan (v1)

## 1. Product Goal

Build a **zero-knowledge secret manager** for personal/family use that:

- Stores secrets securely
- Works across desktop, mobile, browser, CLI
- Supports simple encrypted sharing
- Is fast and offline-friendly
- Uses **S3-compatible object storage as backend**
- Avoids a traditional database (v1)

---

## 2. Core Principles

### Security
- Client-side encryption only
- Server never sees plaintext or raw keys
- End-to-end encrypted sharing
- No dependency on cloud KMS for decryption

### Simplicity
- No backend DB
- Immutable object model
- Minimal concepts:
  - account, device, collection, item, invite

### Performance
- Decrypt only what’s needed
- Local cache + search index
- Incremental sync (events + snapshots)

### Resilience
- Object storage = source of truth
- Immutable writes + versioning
- Offline-first clients

---

## 3. Scope (v1)

### Included
- Personal vault
- Shared collections (family)
- Passwords, notes, API keys, SSH keys
- Browser extension (save/fill)
- Desktop app + CLI
- Local search
- OAuth (Google, Apple, Microsoft)
- Master password unlock
- Device registration
- Export/import
- S3 / R2 / GCS support

### Excluded
- SaaS multi-tenant features
- Server-side search
- Enterprise RBAC / SSO
- Real-time collaboration
- Azure support

---

## 4. Architecture

### Backend
Thin control plane:
- Auth
- Device registration
- Invite handling
- Signed URL issuance
- Policy enforcement

### Storage
- S3 / R2 / GCS only
- Encrypted blobs

### Client
- Local encrypted DB (SQLite / IndexedDB)
- Local search index
- Sync cursor
- Device key storage

### Truth Model
- Immutable objects
- Append-only event log
- Periodic snapshots
- Small mutable pointers

---

## 5. Authentication Model

### Identity
- OAuth (Google, Apple, Microsoft)

### Vault Unlock
- Master password required

### Principle
OAuth = identity  
Password = cryptographic root

---

## 6. Threat Model

Defend against:
- Server compromise → ciphertext only
- Object store breach → ciphertext only
- Device theft → encrypted cache + lock
- Network replay → signed + short-lived requests
- Malicious browser pages → strict extension isolation
- Sharing misuse → clear ownership + revocation

---

## 7. Cryptography

### Pattern
Per-item envelope encryption

### Key Hierarchy
Master Password → Argon2id → Master Key
Master Key → Account Key
Account Key → Collection Keys
Collection Key → Item Keys
Item Key → Item Content

### Properties
- Per-item isolation
- Efficient updates
- Clean sharing/revocation

---

## 8. Metadata Model

### Encrypted
- Title, username, URL, tags, notes
- Item content

### Minimal cleartext
- Object ID
- Version
- Timestamp (if needed)
- Hash

---

## 9. Data Model

- Account config
- Device
- Collection
- Membership
- Item
- Item index
- Event
- Snapshot
- Invite

---

## 10. Storage Layout
/accounts/{id}/config.json
/accounts/{id}/heads/latest.json
/accounts/{id}/devices/{device_id}.json
/accounts/{id}/snapshots/{snapshot_id}.json
/accounts/{id}/events/YYYY/MM/{event}.json
/accounts/{id}/collections/{collection_id}/…
/accounts/{id}/invites/{invite_id}.json

- Immutable writes preferred
- Versioning enabled

---

## 11. Sync Model

### Initial Sync
1. Fetch config
2. Load snapshot
3. Replay events
4. Build local state

### Incremental
1. Fetch head
2. Pull new events
3. Apply locally

### Snapshots
- Periodic state checkpoints
- Prevent infinite replay

---

## 12. Concurrency

- Optimistic concurrency
- Version-based conflict detection

v1 strategy:
- Manual conflict resolution if needed

---

## 13. Client Design

### Local Storage
- Encrypted SQLite / IndexedDB

### Stores
- Ciphertext cache
- Decrypted index
- Search index
- Sync state

### Benefits
- Fast
- Offline
- Low latency

---

## 14. Search

- Fully local
- No server-side search

Search fields:
- Title
- Username
- URL
- Tags
- Notes

---

## 15. Sharing

### Unit
- Collection-based

### Roles (v1)
- Owner
- Member

### Flow
1. Create invite
2. Wrap collection key
3. Accept invite
4. Sync data

### Revocation
- Update membership
- Rotate collection key if needed

---

## 16. Agent-Native Secrets (future)

- Special collections for agents
- Scoped, short-lived access
- CLI/API usage
- Least privilege model

---

## 17. Browser Extension

### Security
- Crypto isolated from page
- Origin-bound autofill
- No plaintext exposure to DOM

### Performance
- Local lookup first
- Decrypt on demand

---

## 18. Recovery

### v1 Model
- Master password
- Recovery key

No reset if both lost.

---

## 19. Backend Responsibilities

- OAuth handling
- Device registration
- Signed URL generation
- Invite coordination
- Rate limiting

Not responsible for:
- decrypting secrets
- indexing data

---

## 20. Infrastructure

### Providers
- AWS S3
- Cloudflare R2
- GCS

### Terraform
- Buckets
- IAM
- Functions
- DNS

### Bucket Settings
- Versioning ON
- Lifecycle rules
- Encryption at rest

---

## 21. Regional / GDPR

### Rule
- Account = single region

### Per-region stack
- Storage
- Backend
- Logs

### Avoid
- Global shared state

---

## 22. Cost Model

### Costs
- Storage
- Requests
- Egress
- Logs

### Risks
- Too many small writes
- Excess listing
- Version sprawl

### Mitigation
- Snapshots
- Efficient prefixes
- Log limits

---

## 23. Risks

### Product
- Recovery failure
- Sharing complexity
- Sync bugs

### Security
- Local compromise
- Metadata leakage
- OAuth linking issues

### Architecture
- Client complexity
- Object-store limitations

---

## 24. Differentiation

- Agent-native secrets
- User-owned storage
- Simple sharing
- Unified human + machine model

---

## 25. Implementation Plan

### Phase 1
- Single-user vault
- Sync engine
- Desktop + CLI

### Phase 2
- Browser extension

### Phase 3
- Sharing

### Phase 4
- Recovery + export

### Phase 5
- Agent access

---

## 26. Final Decisions

- OAuth + master password
- No DB
- Object storage only
- Local-first clients
- Collection-based sharing
- Recovery key required

---

## 27. Summary

A **client-side encrypted, local-first secret manager** using:

- Per-item encryption
- Collection-based sharing
- Append-only sync
- Snapshots
- S3-compatible storage

This yields:

- Strong security
- Low cost
- High portability
- Simple infrastructure

The hardest parts are:

- Sync correctness
- Sharing correctness
- Extension security
- Recovery UX