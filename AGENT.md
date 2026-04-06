Always rebase off latest, review changes, fix conflicts, update your PR. Read `.whoami`, `docs/` and your assigned GitHub Project item/issue first. Periodically and frequently communicate with your peers using switchboard and github. Follow the current milestone, write scope, and agent-identity conventions. Do the assigned work, update repo docs if contracts change, use GitHub issues/project for progress and handoffs.

CoordinationDocs: docs/WORKBOARD.md,docs/INTERFACES.md,docs/DECISIONS.md,docs/HANDOFFS.md,docs/GITHUB_PROJECTS.md

# Local docker identity. Keep these values unique per engineer or agent on a shared machine.

SafeStackName: safe-codex
ControlPlanePort: 18080
LocalstackPort: 14566

# You are a `Safe` engineer agent.

Before doing anything:

1. Read `.whoami` to confirm your agent identity and coordination conventions.
2. Check the GitHub Project board and the linked GitHub issue for your assigned task: `https://github.com/users/cyber-nic/projects/1`
3. Read `docs/project/WORKBOARD.md`, `docs/project/INTERFACES.md`, `docs/project/DECISIONS.md`, `docs/project/HANDOFFS.md`, and `docs/project/GITHUB_PROJECTS.md`.
4. Read the relevant files in `docs/architecture/` to understand product, architecture, protocol, security, and implementation priorities.
5. Read `docs/styles/` for conventions on git, code style, and other project standards.
6. Run `git pull` to make sure you have the latest, then create a branch for your new work; commit as needed to your branch and push changes regularly.
7. Update and run unit tests. Update and run integration tests using your local docker compose stack.
8. Pull latest from main and fix conflicts. Once complete we'll create a PR.

Execution rules:

- Follow the current milestone and write-scope boundaries in `docs/WORKBOARD.md`.
- Treat repo docs as the technical source of truth.
- Use GitHub issues/project for progress updates, handoffs, blockers, and discussion.
- Use the `.whoami` identity convention when commenting through GitHub.
- Do not use `LOG.md` for new progress tracking; it is being sunset as coordination moves to GitHub Projects plus `docs/`.

Deliverables:

- implement the assigned work within the allowed write scope
- update repo docs if contracts or ownership change
- comment on the matching GitHub issue with progress, blockers, or handoff notes
- summarize what changed, what was verified, and any remaining risks
