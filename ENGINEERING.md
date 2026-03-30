# Development Principles

## Core Stack

- Use Go for backend and systems components.
- Use TypeScript for frontend, shared client logic, and web-facing application code.
- Use Material UI for the primary application UI layer.
- Prefer SSR for web applications where it improves performance, security posture, and delivery of core flows.
- Use containers for local development, CI, and deployment consistency.
- Use S3-compatible object storage as the durable storage layer.
- Use AWS as the primary cloud platform unless there is a clear reason to abstract or substitute.

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
