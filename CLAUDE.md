# Boxship

External Prow plugin that receives GitHub webhook events from Prow Hook and dispatches them to internal sub-plugins.

## Build

```
make build     # compile binary to _output/boxship
make test      # run all tests
make vet       # run go vet
make verify    # vet + test
make image     # build container image (requires build first)
```

## Architecture

- `cmd/boxship/` - Server entrypoint. Uses `sigs.k8s.io/prow/pkg/githubeventserver` for HTTP event handling.
- `pkg/dispatch/` - Event dispatcher that multiplexes events to registered sub-plugins.
- `pkg/subplugins/` - Sub-plugin implementations. Each sub-package implements `dispatch.SubPlugin`.
- `images/boxship/` - Container image definition.
- `specs/` - Lightweight feature specifications.

## Adding a Sub-Plugin

1. Create a new package under `pkg/subplugins/<name>/`
2. Implement the `dispatch.SubPlugin` interface
3. Register the plugin in `cmd/boxship/main.go`

See `pkg/subplugins/example/` for a reference implementation.

## Dependencies

- `sigs.k8s.io/prow` - Prow libraries for event server, GitHub types, and utilities
- `github.com/openshift/ci-tools` - May be added for shared CI tooling code
