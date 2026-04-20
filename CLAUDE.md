# Boxship

External Prow plugin that receives GitHub webhook events from Prow Hook and dispatches them to internal sub-plugins.

## Build

```
make build     # compile binary to _output/boxship
make test      # run all unit tests
make vet       # run go vet
make verify    # vet + test
make image     # build container image (requires build first)
```

## Architecture

- `cmd/boxship/` - Server entrypoint. Uses `sigs.k8s.io/prow/pkg/githubeventserver` for HTTP event handling.
- `pkg/config/` - Configuration loading, merging, and resolution. Supports layered config (top/org/repo) with supplemental config files.
- `pkg/dispatch/` - Event dispatcher that multiplexes events to registered sub-plugins. Uses `config.Resolver` for enable/disable gating.
- `pkg/subplugins/` - Sub-plugin implementations. Each sub-package implements `dispatch.SubPlugin`.
- `images/boxship/` - Container image definition.
- `specs/` - Lightweight feature specifications.

## Adding a Sub-Plugin

1. Create a new package under `pkg/subplugins/<name>/`
2. Define a narrow `githubClient` interface for the GitHub API methods the plugin needs
3. Accept the client and resolver via constructor: `func New(ghc githubClient, resolver *config.Resolver) *Plugin`
4. Implement the `dispatch.SubPlugin` interface
5. Define a `PluginConfig` struct and `defaultConfig()` if the plugin needs configuration
6. Use `config.ResolvePluginConfig[T](resolver, name, defaultCfg, org, repo)` in handlers to get typed config
7. Register the plugin in `cmd/boxship/main.go`
6. Add unit tests in `pkg/subplugins/<name>/<name>_test.go` using `fakegithub.NewFakeClient()`

See `pkg/subplugins/example/` for a reference implementation.

## Testing

Three layers of testing are available:

### Unit Tests

```
make test      # runs go test ./...
```

All sub-plugin handlers must have unit tests. Use `fakegithub.NewFakeClient()` from `sigs.k8s.io/prow/pkg/github/fakegithub` and inject it via the plugin constructor. Assert mutations via FakeClient tracking fields (`IssueCommentsAdded`, `IssueLabelsAdded`, etc.). Use `pkg/testhelpers/` for event construction helpers.

### Integration Tests

```
make integration-test   # runs tests with //go:build integration tag
```

Located in `test/integration/plugins/<name>/`. Use realistic JSON webhook payloads from `test/integration/testdata/`. Load events with `integration.LoadTestEvent[T](t, filename)`. Integration tests are excluded from `make test`.

### Interactive Dev Server

```
make dev-server    # start boxship with fakegithub (port 8888 webhook, 8889 state)
make dev-webhook EVENT=pull_request PAYLOAD=test/integration/testdata/pull_request_opened.json
make dev-state     # inspect fakegithub mutations
make dev-reset     # clear state between tests
make dev-watch     # auto-restart on code changes (requires watchexec)
```

The dev server uses an in-memory fakegithub client. No real GitHub API calls are made. Use the `/dev-test` Claude Code skill for agent-driven testing.

## Dependencies

- `sigs.k8s.io/prow` - Prow libraries for event server, GitHub types, and utilities
- `github.com/openshift/ci-tools` - May be added for shared CI tooling code
