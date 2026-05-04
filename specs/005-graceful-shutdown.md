# Graceful Shutdown

## Status
Implemented

## Motivation

When ship-hook receives a termination signal (SIGTERM/SIGINT), it must stop accepting new webhook events but allow in-flight sub-plugin handlers to finish their work. Without this, handlers that are mid-way through GitHub API calls (e.g., adding labels, posting comments) may be interrupted, leaving the system in an inconsistent state.

The existing shutdown code calls `eventServer.GracefulShutdown()` and `eventServer.Shutdown(ctx)`, which stops accepting new HTTP requests but does not wait for dispatched sub-plugin goroutines to complete. The Dispatcher fires goroutines with `go` but has no mechanism to track or signal them.

## Design

### Problem Constraints

Prow's `githubeventserver` handler type signatures do not include `context.Context`:

```go
type PullRequestHandler func(*logrus.Entry, github.PullRequestEvent)
type IssueCommentEventHandler func(*logrus.Entry, github.IssueCommentEvent)
type ReviewEventHandler func(*logrus.Entry, github.ReviewEvent)
```

This means the Dispatcher's public handler methods (which must match these signatures) cannot receive a context from the event server. Context must be introduced at the Dispatcher level.

### Approach

The Dispatcher stores a `context.Context` and a `sync.WaitGroup` as struct fields:

- **Context**: A cancellable context created at construction time. Passed to all sub-plugin handler invocations. When shutdown begins, the context is cancelled to signal in-flight handlers.
- **WaitGroup**: Tracks all in-flight sub-plugin goroutines. Incremented before launching each goroutine, decremented when it returns.

Storing a context in a struct is generally discouraged in Go because contexts are meant to flow through call chains. Here it is a deliberate trade-off: the Prow event server does not propagate context through its handler signatures, so the Dispatcher must bridge this gap. The stored context represents the Dispatcher's lifecycle, not a per-request context.

### SubPlugin Interface Change

All handler methods gain a `context.Context` parameter:

```go
type SubPlugin interface {
    Name() string
    HandlePullRequestEvent(context.Context, *logrus.Entry, github.PullRequestEvent)
    HandleIssueCommentEvent(context.Context, *logrus.Entry, github.IssueCommentEvent)
    HandleReviewEvent(context.Context, *logrus.Entry, github.ReviewEvent)
}
```

Sub-plugins should pass this context to any downstream calls (GitHub API, HTTP requests, etc.) so that work is abandoned promptly on shutdown.

### Dispatcher Shutdown

The Dispatcher exposes a `Shutdown(ctx context.Context) error` method:

1. Cancels the internal context, signaling all in-flight handlers
2. Waits for all tracked goroutines to finish (via WaitGroup)
3. Respects the provided context's deadline — if it expires before all handlers finish, returns the context error

### Shutdown Sequence in main.go

```
SIGTERM received
  → eventServer.GracefulShutdown()     // stop accepting new HTTP requests
  → dispatcher.Shutdown(ctx)           // cancel context + wait for in-flight handlers
  → eventServer.Shutdown(ctx)          // close HTTP server
```

## Verification

- Unit test: dispatcher shutdown waits for in-flight handlers
- Unit test: context cancellation is visible to sub-plugin handlers
- Existing dispatcher and sub-plugin tests updated for new interface
- `make verify` passes
