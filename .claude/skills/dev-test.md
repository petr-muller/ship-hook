---
name: dev-test
description: Interactive development testing for ship-hook plugins. Start a dev server, send test webhooks, inspect plugin state.
---

# Dev Test Skill

Orchestrates the ship-hook dev server for interactive plugin testing. The dev server runs ship-hook with a fakegithub client so plugins can actuate without touching real GitHub.

## Sub-commands

Invoke as `/dev-test <command>` where command is one of:

### start

1. Run `make dev-server` in background (use Bash with `run_in_background`)
2. Wait for port 8888 to be listening: `curl -s --retry 10 --retry-delay 1 --retry-connrefused http://localhost:8888/hook > /dev/null 2>&1 || true`
3. Confirm the server is running by checking `curl -s http://localhost:8889/state`

### send

Usage: `/dev-test send <event-type> <payload-file>`

Run: `make dev-webhook EVENT=<event-type> PAYLOAD=<payload-file>`

Available test payloads in `test/integration/testdata/`:
- `pull_request_opened.json` - PR opened event
- `issue_comment_created.json` - Issue comment created event

Example: `/dev-test send pull_request test/integration/testdata/pull_request_opened.json`

### state

Fetch and display the current fakegithub state showing all mutations made by plugins:

```bash
curl -s http://localhost:8889/state | jq .
```

### reset

Clear all fakegithub state between test runs:

```bash
curl -s -XPOST http://localhost:8889/reset
```

### stop

Kill the running dev server process. Find and kill the process listening on port 8888.

```bash
kill $(lsof -t -i:8888) 2>/dev/null || kill $(ss -tlnp 'sport = :8888' | grep -oP 'pid=\K\d+') 2>/dev/null
```

## Safety

- The dev server uses an in-memory fakegithub client. No real GitHub tokens or API calls are involved.
- The HMAC secret is hardcoded to `devhmac` for development convenience.
- All "GitHub API calls" from plugins go to the fake client and are recorded for inspection via `/state`.
