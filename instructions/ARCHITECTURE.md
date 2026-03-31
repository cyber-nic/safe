# Safe Architecture

## Purpose

This document is the top-level architecture map for Safe. It explains the major runtime components, the implementation split between Go and TypeScript, the core trust boundaries, the main data flows, and where to go next for deeper detail.

It is intentionally shorter than [SYSTEM_DESIGN.md](./instructions/SYSTEM_DESIGN.md). The system design document is the deep specification. This document is the entry point.

## System Shape

Safe is a zero-knowledge secret manager with three primary runtime parts:

- client applications that perform unlock, cryptography, local storage, and sync
- a thin control plane that handles identity, policy, and scoped storage access
- S3-compatible object storage that holds encrypted durable state

The system is designed so the service can coordinate access without being able to decrypt vault contents in normal operation.

## Runtime Components

### Clients

Safe has four client surfaces:

- desktop app as the primary v1 user client
- browser extension for lookup and conservative autofill
- CLI for power users and automation-friendly flows
- web app for SSR onboarding and account-management flows

Client responsibilities:

- master-password unlock
- local key derivation and cryptographic operations
- encrypted local persistence
- local search over decrypted metadata
- sync and replay
- device enrollment, recovery, and sharing flows

### Control Plane

The control plane is intentionally narrow.

Responsibilities:

- OAuth-backed identity
- device registration and revocation
- invite coordination
- policy decisions
- short-lived signed URLs or scoped storage credentials
- rate limiting, audit events, and region routing

Non-responsibilities:

- decryption
- plaintext secret storage
- server-side search
- primary vault-content storage

### Object Storage

S3-compatible object storage is the durable data plane.

It stores:

- immutable encrypted objects
- append-only event records
- encrypted snapshots
- small mutable pointers and account metadata
- invite and membership artifacts

## Technology Split

The v1 implementation split is:

- Go for the control plane, CLI, and backend or storage-facing system components
- TypeScript for the desktop app, browser extension, and SSR web-facing client code
- Material UI for the primary UI component layer
- containers for local development and deployment consistency
- AWS as the primary cloud target

This split keeps operational services simple in Go while keeping user-facing product surfaces efficient in TypeScript.

## Trust Boundaries

The most important security boundary is between the unlocked client and everything else.

Trusted for confidentiality:

- unlocked client runtime on a healthy device
- local cryptographic implementations
- local possession of master password and recovery key

Not trusted for confidentiality:

- control plane
- object storage
- network

High-risk boundaries:

- browser extension interaction with web pages
- local decrypted search index and cached derivatives
- mutable metadata used for sync, authz, and recovery flows

## Main Data Flows

### Unlock Flow

1. User authenticates with OAuth.
2. Client fetches account metadata.
3. User enters master password.
4. Client derives keys locally and unwraps the account master key.
5. Client opens local state and begins sync.

### Sync Flow

1. Client fetches authenticated mutable metadata.
2. Client validates freshness, signatures, and rollback markers.
3. Client downloads new immutable objects and events.
4. Client applies replay deterministically.
5. Client advances its local cursor only after durable local commit.

### Write Flow

1. Client creates new immutable encrypted objects.
2. Client creates and signs a canonical event.
3. Client attempts compare-and-swap on the head pointer.
4. If successful, the write commits.
5. If not, the client retries from new head state using idempotency rules.

### Sharing Flow

1. Owner creates a collection invite.
2. Recipient authenticates and presents device trust material.
3. Owner wraps the collection key for the recipient.
4. Recipient accepts and syncs collection state through collection-scoped authorization.

## Domain Areas

The main bounded contexts in the system are:

- Account
- Device
- Vault
- Collection Sharing
- Sync
- Authorization

Each of these should own its own rules, state transitions, and invariants. Backend code should be organized by domain first, not by transport or framework.

## Repository Direction

The intended repository shape is:

- `/apps` for desktop, extension, and web
- `/cmd` for Go binaries such as CLI and control plane
- `/internal` for Go domain and infrastructure packages
- `/packages` for TypeScript shared packages, UI kit, and test vectors
- `/infra` for Terraform

Shared protocol schemas, canonical serialization rules, and fixture vectors must remain language-neutral so Go and TypeScript implementations stay interoperable.

## Read Next

- Read [PRODUCT.md](./instructions/PRODUCT.md) for scope and user outcomes.
- Read [SYSTEM_DESIGN.md](./instructions/SYSTEM_DESIGN.md) for the full architecture and object model.
- Read [PROTOCOL.md](./instructions/PROTOCOL.md) for canonical serialization, signing, rollback, and capability rules.
- Read [SECURITY.md](./instructions/SECURITY.md) for the design assessment and release blockers.
- Read [IMPLEMENTATION_PLAN.md](./instructions/IMPLEMENTATION_PLAN.md) for sequencing and workstreams.
