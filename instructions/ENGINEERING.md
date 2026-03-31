# Development Principles

## Core Stack

- Use Go for backend and systems components.
- Use TypeScript for frontend, shared client logic, and web-facing application code.
- Use Material UI for the primary application UI layer.
- Prefer SSR for web applications where it improves performance, security posture, and delivery of core flows.
- Use containers for local development, CI, and deployment consistency.
- Use S3-compatible object storage as the durable storage layer.
- Use AWS as the primary cloud platform unless there is a clear reason to abstract or substitute.

## Local Development Standard

- Use Docker Compose as the default local development orchestrator.
- Use Docker Compose `develop.watch` for the inner loop where containerized services benefit from file sync, rebuild, or restart behavior.
- Prefer Compose over local Kubernetes for day-to-day development because it is simpler, faster to understand, and closer to the actual v1 topology.
- Reserve `kind` for explicit Kubernetes validation work such as Helm charts, ingress behavior, service networking, and deployment smoke tests.
- Do not make `kind`, Tilt, or any Kubernetes control loop a prerequisite for routine feature development in v1.
- Use LocalStack as the default local AWS emulator, including the local S3 endpoint for development and integration testing.
- Configure local S3 clients against LocalStack in a way that matches AWS behavior as closely as practical, including versioning, pre-signed URLs, and explicit endpoint configuration.
- Keep LocalStack credentials or auth tokens out of the repository and source them from the developer environment.
- Maintain a small set of integration tests that can also run against real AWS S3 before release-critical changes, because no emulator is a substitute for the real platform.
- Treat any alternative S3-compatible server as optional secondary coverage, not the primary local target.

## Local Development Notes

- The default local stack should be a Compose project with the control plane, supporting services, and LocalStack.
- Desktop, web, extension, and CLI development may run partly on the host when that produces a faster feedback loop, but all backing dependencies should still be available through Compose.
- If a service has a strong native hot-reload workflow, prefer Compose-managed dependencies plus a host-run application process over forcing every edit through image rebuilds.
- Introduce `kind` only after we have Kubernetes manifests worth validating. It should not precede a stable Compose-based environment.
- We are not standardizing on MinIO for v1 local development. The primary goal is AWS S3 behavior, so LocalStack is the better default fit.

## Engineering Principles

- Write unit tests for core logic, security-sensitive code, domain rules, and regressions.
- Use domain-driven design to keep the code organized around core business concepts and bounded contexts.
- Keep the code DRY. Eliminate duplication when it improves clarity and maintainability.
- Keep the system lean. Prefer simple architecture, narrow interfaces, and minimal operational overhead.
- Favor clean code: readable names, small focused modules, explicit boundaries, and predictable behavior.

## Delivery Rules

- Choose straightforward solutions before clever ones.
- Optimize for maintainability over novelty.
- Keep dependencies intentional and limited.
- Build only what is needed for the current phase.
- Treat security, correctness, and testability as first-order concerns.
