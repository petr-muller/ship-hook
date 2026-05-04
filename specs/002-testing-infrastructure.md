# Testing Infrastructure

## Status
Draft

## Motivation

SHIP Hook needs a layered testing strategy that supports fast iteration for both humans and agents. Sub-plugins need to actuate against GitHub (via injected clients), and each layer of testing requires different trade-offs between speed, realism, and isolation.

## Design

### Layer 1: Unit Tests

Standard Go tests (`make test`). Each sub-plugin defines a narrow `githubClient` interface for the GitHub API methods it uses, accepts it via constructor injection, and tests against `fakegithub.FakeClient`. Test helpers in `pkg/testhelpers/` reduce event construction boilerplate.

### Layer 2: Integration Tests

Tests with `//go:build integration` tag in `test/integration/`. Run via `make integration-test`. Use realistic JSON webhook payloads from `test/integration/testdata/` and verify full handler chains against FakeClient state.

### Layer 3: Interactive Dev Server

A local dev server (`cmd/devserver/`) runs ship-hook with an embedded `fakegithub.FakeClient`. Webhooks are sent via `cmd/devwebhook/` (wraps Prow's phony). State inspection via `/state` and `/reset` HTTP endpoints. Controlled via `make dev-server`, `make dev-webhook`, `make dev-state`, `make dev-reset`.

A Claude Code skill (`.claude/skills/dev-test.md`) exposes the dev server workflow to both humans and agents.

## Verification

1. `make verify` passes (includes unit tests, excludes integration tests)
2. `make integration-test` passes
3. `make dev-server` starts and accepts webhooks via `make dev-webhook`
4. `make dev-state` shows plugin mutations; `make dev-reset` clears them
