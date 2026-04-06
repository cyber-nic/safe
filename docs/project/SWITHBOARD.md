# Communication

Use the local switchboard for fast engineer-to-engineer coordination.

## Peer Identity

- Read [`.whoami`](/code/safe/.whoami) before starting work.
- Current peer engineer: Claude
- Current local identity: `sender=codex`, `role=engineer`

## Switchboard CLI

Use the installed `switchboard` command from `PATH`.

- Preferred invocation: `switchboard ...`
- Current binary location on this machine: `/usr/local/bin/switchboard`
- `/usr/local/bin/switchboard` is a symlink to `/code/switchboard/bin/switchboard`
- Do not assume `/tmp/switchboard` exists


Expected commands:

```sh
switchboard send -sender codex -role engineer -text "ready"
switchboard history -n 10
switchboard watch
```

## Working Agreement

- Check switchboard history when you start work and before you open or update a PR.
- Send short progress notes when your plan changes, when you hit a blocker, and when a branch or PR is ready for review.
- Include concrete references in messages when relevant: issue number, branch name, PR number, or interface dependency.
- Communicate frequently around PRs so other engineers can coordinate dependent work without polling GitHub.

## Local Validation

The switchboard service in this repo is reachable at `http://127.0.0.1:3017`.

Useful fallback checks:

```sh
curl -s http://127.0.0.1:3017/api/history
curl -s http://127.0.0.1:3017/api/history | jq .
```

Validated on 2026-04-06:

- `switchboard history -n 10` showed Claude's inbound message.
- `switchboard send ...` posted a Codex reply that appeared in history.
- `curl -s http://127.0.0.1:3017/api/history` returned the same conversation via HTTP.

## Minimum Routine

1. Read `.whoami`.
2. Check `switchboard history -n 10`.
3. Send a short status note when starting or changing direction.
4. Send another note when opening, updating, or handing off a PR.
