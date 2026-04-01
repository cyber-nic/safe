# Safe

Safe is a zero-knowledge secret manager for individuals and small trusted groups.

## Current Docs

- [PRODUCT.md](./instructions/PRODUCT.md)
- [SYSTEM_DESIGN.md](./instructions/SYSTEM_DESIGN.md)
- [IMPLEMENTATION_PLAN.md](./instructions/IMPLEMENTATION_PLAN.md)
- [SECURITY.md](./instructions/SECURITY.md)
- [ENGINEERING.md](./instructions/ENGINEERING.md)
- [PROTOCOL.md](./instructions/PROTOCOL.md)
- [ARCHITECTURE.md](./instructions/ARCHITECTURE.md)

## Stack

- Go for backend services and CLI
- TypeScript for web-facing client code
- Material UI for primary UI work
- SSR for web applications
- Containers for local development and deployment consistency
- AWS and S3-compatible storage for infrastructure and durable encrypted state

## Before Implementation

The following decisions are considered frozen enough to start scaffolding:

- zero-knowledge client-side encryption model
- Go plus TypeScript split
- S3-compatible object storage as the durable data plane
- explicit device enrollment
- collection-based sharing
- authenticated mutable metadata and rollback protection as release requirements

## Current Status

The repository is no longer at pure scaffold stage.

Useful progress already landed:

- monorepo structure and local Compose development workflow
- starter control-plane and CLI binaries
- shared Go and TypeScript protocol models plus canonical fixture coverage
- a script-friendly CLI prototype for local secret CRUD, history, import/export, and JSON output

But the project is still behind the plan's actual critical path:

- cryptographic key hierarchy and recovery flows are still missing
- local encrypted persistence and unlock or lock lifecycle are still missing
- signed mutable metadata and rollback detection are still missing
- object-store sync is still a starter model, not the intended v1 storage protocol

## Immediate Next Steps

1. Finish freezing signer ownership, authenticated mutable metadata, and rollback rules from [PROTOCOL.md](./instructions/PROTOCOL.md).
2. Implement the first real cryptographic key hierarchy slice: password derivation, KEK/AMK wrapping, and recovery-key fixtures.
3. Replace the CLI's starter in-memory vault model with durable encrypted local persistence aligned with the Phase 2 local-runtime plan.
4. Start the real object-storage sync path only after the local encrypted-state and signed-metadata groundwork is in place.

## Local Development

The initial local development stack is Compose-first:

- `localstack` for the local AWS and S3-compatible endpoint
- `control-plane` as the first Go service running in a dev container with Compose watch support

Quick start:

1. Copy `.env.example` to `.env` if you need to override defaults or set `LOCALSTACK_AUTH_TOKEN`.
2. Run `make up`.
3. Run `make logs` to follow service output.
4. Run `make watch` if you want Compose-managed restart behavior on Go file changes.

Useful targets:

- `make ps`
- `make shell-control-plane`
- `make shell-localstack`
- `make s3-ls`
- `make test-go`
- `make down`
