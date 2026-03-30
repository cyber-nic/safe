# Cipher

Cipher is a zero-knowledge secret manager for individuals and small trusted groups.

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
- TypeScript for desktop, extension, and web-facing client code
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

## Immediate Next Steps

1. Scaffold the monorepo structure described in [IMPLEMENTATION_PLAN.md](./instructions/IMPLEMENTATION_PLAN.md).
2. Freeze canonical schemas and signing fixtures from [PROTOCOL.md](./instructions/PROTOCOL.md).
3. Create the Go and TypeScript test-vector harnesses.
4. Stand up the initial containerized local development environment.
