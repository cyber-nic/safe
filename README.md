# Safe

Safe is a zero-knowledge secret manager for individuals and small trusted groups.

## Current Docs

- [COMMUNICATION.md](./COMMUNICATION.md)
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
- durable file-backed local runtime wiring in the CLI, including password-derived unlock and encrypted secret-material storage
- a local server-rendered web client that identifies, unlocks, saves one login secret, locks, and reopens against the same account-local runtime concepts
- full Go test coverage for the current local-runtime path passes on `main`

The repository still has important follow-up gaps after M3:

- real third-party OAuth redirect integration is still missing; local demo flows still use the explicit dev-token path
- the localhost demo is now runnable end to end, but it is still a developer smoke path rather than a polished product environment
- broader multi-user sharing, revocation, and production deployment work remain out of scope for the shipped milestone

## Immediate Next Steps

1. Replace the HS256 dev-token login path with a real OAuth provider redirect flow.
2. Keep the localhost demo path simple enough that a new developer can validate the product loop quickly.
3. Continue tightening control-plane and sync hardening without reopening the shipped single-user milestone boundary.
4. Keep browser-native adapter and multi-user scope explicit follow-up work instead of backfilling it into the current product slice.

## Local Development

The local development stack is Compose-first:

- `localstack` for the local AWS and S3-compatible endpoint
- `control-plane` as the first Go service running in a dev container with Compose watch support
- `web` as the server-rendered local client on port `3000`

Quick start:

1. Copy `.env.example` to `.env`.
2. Optionally copy `compose.override.yaml.example` to `compose.override.yaml` for local-only stack naming overrides.
3. Run `make up`.
4. Run `make token` and copy the printed JWT into `.env` as `SAFE_OAUTH_ACCESS_TOKEN`.
5. Run `make up` again if you changed `.env` after the first boot so the web and control-plane containers pick up the token.
6. Open `http://localhost:3000`, click `Sign In`, create a secret, and confirm the vault flow in the browser.
7. Run `make cli ARGS="secret list"` to confirm the same account-local data from the CLI.
8. Run `make cli ARGS="sync push"` and then open a second browser profile to exercise the sync path against the same localstack-backed control plane.

The web service now starts inside Compose, so `npm start --prefix apps/web` is optional and mainly useful if you intentionally want to run the web client outside the container.

The `make token` target uses the values already in `.env`:

- `SAFE_OAUTH_HS256_SECRET`
- `SAFE_OAUTH_ISSUER`
- `SAFE_OAUTH_AUDIENCE`
- `SAFE_OAUTH_ACCOUNT_ID`

It prints a valid dev JWT for the current `.env`; it does not write back to `.env` automatically.

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
- `make token`
- `make shell-control-plane`
- `make shell-localstack`
- `make s3-ls`
- `make test-go`
- `make cli ARGS="secret list"`
- `make cli ARGS="sync push"`
- `make down`

## Engineer Coordination

Use the local switchboard for short coordination messages with other engineers and agents, especially around active branches and PRs.

Expected CLI:

```sh
switchboard-cli send -sender codex -role engineer -text "ready"
switchboard-cli history -n 10
switchboard-cli watch
```

If the binary is not on `PATH`, invoke it by absolute path. In this sandbox, the binary was provided at `/tmp/switchboard`.

Minimum expectation:

- check history when you start work
- send short updates when scope changes or blockers appear
- communicate frequently with PRs, including open, update, and handoff moments

See [COMMUNICATION.md](./COMMUNICATION.md) for the working agreement and validation notes.
