# Communication

Use the local switchboard for fast engineer-to-engineer coordination.

## Peer Identity

- Read [`.whoami`](.whoami) before starting work.
- Current peer engineer: Codex

## Switchboard CLI

Use the installed `switchboard` command from `PATH`.

- Preferred invocation: `switchboard ...`
- Current binary location on this machine: `/usr/local/bin/switchboard`
- `/usr/local/bin/switchboard` is a symlink to `/code/switchboard/bin/switchboard`
- Do not assume `/tmp/switchboard` exists

Expected commands:

```sh
switchboard send -sender "agent" -role engineer -text "ready"
switchboard history -n 10
switchboard watch
```

## Stand-up Format

Every switchboard message should follow this template:

```
Status: <starting|in-progress|blocked|pr-open|pr-updated|handoff|complete>
Task: W<n> / #<issue>
Done: <what was accomplished since last update — omit if starting>
Next: <what comes next>
Blocker: <concrete description — omit if none>
```

Keep it short. One sentence per field. Include branch or PR number when relevant.

## Trigger Points

Send a stand-up message at each of these moments — do not skip:

1. **Session start** — after reading `.whoami` and history; before touching any code
2. **Plan change** — when scope, approach, or dependency assumptions shift
3. **Blocker hit** — immediately; do not wait to resolve it first
4. **PR opened** — include branch name and issue number
5. **PR updated** (new commits, rebase, or review response)
6. **Handoff** — before stepping away; include next action for the other engineer
7. **Task complete** — after the PR merges or the deliverable lands

## Working Agreement

- Always read the last 10 messages before sending — avoid duplicating a message the peer just sent.
- Include concrete references: issue number, branch name, PR number, or interface name.
- Frequent short messages beat infrequent long ones — the peer should never have to poll GitHub to know your state.

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
2. Run `switchboard history -n 10` — read what the peer sent.
3. Send a **session start** stand-up before touching any code.
4. Send a stand-up at every trigger point listed above.
5. Send a **task complete** or **handoff** stand-up before ending the session.
