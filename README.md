# Safe

Safe is a zero-knowledge secret manager for individuals and small trusted groups.

## Current Docs

- [PRODUCT.md](./docs/architecture/PRODUCT.md)
- [SYSTEM_DESIGN.md](./docs/architecture/SYSTEM_DESIGN.md)
- [IMPLEMENTATION_PLAN.md](./docs/architecture/IMPLEMENTATION_PLAN.md)
- [SECURITY.md](./docs/architecture/SECURITY.md)
- [ENGINEERING.md](./docs/architecture/ENGINEERING.md)
- [PROTOCOL.md](./docs/architecture/PROTOCOL.md)
- [ARCHITECTURE.md](./docs/architecture/ARCHITECTURE.md)

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

1. Finish the first real user loop in a client surface: sign in, save a secret, and read it back.
2. Replace the current starter in-memory vault behavior with durable encrypted local persistence aligned with the local-runtime plan.
3. Implement the minimum unlock path needed to reopen and read a saved secret safely.
4. Continue signer, rollback, and sync-hardening work only insofar as it supports that first trustworthy save/read loop, then expand outward.

## Local Development

The initial local development stack is Compose-first:

- `localstack` for the local AWS and S3-compatible endpoint
- `control-plane` as the first Go service running in a dev container with Compose watch support

Quick start:

1. Copy `.env.example` to `.env` if you need to override defaults or set `LOCALSTACK_AUTH_TOKEN`.
2. Optionally copy `compose.override.yaml.example` to `compose.override.yaml` for local-only stack naming overrides.
3. Run `make up`.
4. Run `make logs` to follow service output.
5. Run `make watch` if you want Compose-managed restart behavior on Go file changes.

When multiple engineers or agents run the stack on the same machine, each one needs a unique Compose namespace as well as unique host ports. Set the shared identity in your local `.env`:

- `SAFE_STACK_NAME=safe-<engineer>`

Then, if you want the stack name to live outside shared repo config, copy `compose.override.yaml.example` to `compose.override.yaml` and edit the explicit local names there. Keep the actual port numbers in `.env`. The `make` targets now pass `.env` explicitly and automatically include `compose.override.yaml` when that file exists.

Example `.env`:

```env
SAFE_STACK_NAME=safe-codex
```

Example `compose.override.yaml`:

```yaml
name: safe-codex

volumes:
  localstack-data:
    name: safe-codex-localstack-data
```

Example matching local port values:

```env
CONTROL_PLANE_PORT=18080
LOCALSTACK_PORT=14566
```

Using a unique `SAFE_STACK_NAME` avoids collisions on the Compose project name and the LocalStack data volume; changing only the ports is not enough.
Keep the port values in `.env`: Compose merges `ports:` arrays across override files, so a `compose.override.yaml` is not a clean place to replace the base port mappings.

Useful targets:

- `make ps`
- `make shell-control-plane`
- `make shell-localstack`
- `make s3-ls`
- `make test-go`
- `make down`
