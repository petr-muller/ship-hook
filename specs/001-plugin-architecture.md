# Plugin Architecture

## Status
Implemented (skeleton)

## Motivation

Boxship needs to host multiple independent behaviors ("sub-plugins") in a single deployable binary. Different sub-plugins may be owned by different people and should be developed independently without tight coupling. The architecture must integrate with Prow's external plugin protocol, where Prow Hook forwards GitHub webhook events over HTTP.

## Design

### Event Flow

```
GitHub → Prow Hook → boxship HTTP server → Dispatcher → SubPlugin handlers
```

1. Prow Hook receives a GitHub webhook and forwards it to boxship's HTTP endpoint
2. The `githubeventserver.GitHubEventServer` (from `sigs.k8s.io/prow/pkg/githubeventserver`) validates the HMAC signature and deserializes the event
3. The event server invokes registered handler functions, which are the Dispatcher's methods
4. The Dispatcher iterates registered sub-plugins and calls each one's handler in a separate goroutine

### Key Types

**`dispatch.SubPlugin` interface** (`pkg/dispatch/dispatch.go`): Every sub-plugin implements this.

```go
type SubPlugin interface {
    Name() string
    HandlePullRequestEvent(*logrus.Entry, github.PullRequestEvent)
    HandleIssueCommentEvent(*logrus.Entry, github.IssueCommentEvent)
}
```

New handler methods can be added to the interface as needed. The upstream event server supports: `PullRequest`, `IssueComment`, `Issue`, `Push`, `Status`, `ReviewComment`, `Review`, `WorkflowRun`, `RegistryPackage`.

**`dispatch.Dispatcher`** (`pkg/dispatch/dispatch.go`): Holds registered sub-plugins and implements the handler function signatures expected by `githubeventserver`. Each handler fans out to all registered sub-plugins concurrently.

### Sub-Plugin Package Layout

Each sub-plugin lives in its own package under `pkg/subplugins/<name>/`. A sub-plugin package must:

- Export a constructor (e.g. `New(...)`) returning a type that implements `dispatch.SubPlugin`
- Be registered in `cmd/boxship/main.go` via `dispatcher.Register(...)`

See `pkg/subplugins/example/` for the reference implementation.

### Server Entrypoint

`cmd/boxship/main.go` wires everything together:

1. Parses flags (`githubeventserver.Options`, `flagutil.GitHubOptions`, HMAC secret path, dry-run, log level)
2. Creates the `githubeventserver.GitHubEventServer`
3. Creates a `Dispatcher` and registers all sub-plugins
4. Registers the Dispatcher's handler methods on the event server
5. Starts the HTTP server with graceful shutdown

### Container Image

Built following the openshift/ci-tools pattern: the Go binary is compiled externally, and `images/boxship/Dockerfile` simply ADDs it into a UBI9 minimal base image.

## Verification

- `make build` compiles the binary
- `make test` runs all tests
- `make verify` runs vet + tests
- The binary starts and listens on port 8888 (default) at `/hook` (default)
