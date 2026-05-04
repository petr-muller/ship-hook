# Observability

## Status
Proposed

## Motivation

SHIP Hook currently has no metrics exposure, no health endpoints, and limited logging conventions. During live testing on ota-stage, the lack of structured observability made it impossible to diagnose why the ready-for-humans plugin was silently not acting on events. While the HandlerResult contract (spec 006) addressed the immediate logging gap, ship-hook needs a coherent observability foundation that operators can rely on and sub-plugin authors can follow without bolting things on ad-hoc.

## Principles

### Logging

1. **The dispatcher owns lifecycle logging.** Event receipt, routing decisions, and handler outcomes are logged by the dispatcher, not by sub-plugins. Sub-plugins log domain-specific decisions only (e.g., "label already exists", "API call failed").

2. **Structured fields are injected, not duplicated.** The dispatcher injects `event_type`, `org`, `repo`, `pr`, `plugin` into the logrus entry passed to handlers. Sub-plugins add domain-specific fields but never re-log these base fields.

3. **Three log levels, no Warn.** Info for actions taken and errors that need operator attention. Debug for skipped events, internal decisions, and diagnostic detail. Error for failures that need investigation. Warn is not used — a message is either actionable (Error) or informational (Info/Debug).

4. **HandlerResult is the primary sub-plugin observability channel.** Returning `Irrelevant("reason")` or `Handled("reason")` from handlers is how sub-plugins communicate their decisions to the dispatcher and the operator. Meaningful reasons are required — empty reasons are acceptable only for plugins that always act on every event (like the example plugin).

### Metrics

5. **Expose what Prow already collects.** Prow's GitHub client automatically tracks API call counts, rate limits, request durations, and cache behavior. The event server tracks webhook counts and response codes. SHIP Hook must expose these by wiring up `metrics.ExposeMetrics()`. This is zero-cost, high-value.

6. **One dispatcher-level histogram.** The dispatcher records handler duration per plugin per event type, using the `took_action` label from HandlerResult. This lets operators spot slow or misbehaving plugins without each plugin managing its own metrics.

7. **Sub-plugins do not register metrics.** Sub-plugins must not call `prometheus.MustRegister()` or create their own collectors. If a plugin needs business-specific metrics in the future, the dispatcher will provide a metrics registry interface. This prevents metric name collisions and ensures consistent labeling.

### Health

8. **Standard Prow health endpoints.** SHIP Hook exposes `/healthz` (liveness) and `/healthz/ready` (readiness) via `pjutil.NewHealthOnPort()`, following the same pattern as hook and other Prow components. Operators can configure the port via `--health-port`.

### Tracing

9. **No distributed tracing for now.** Prow does not use OpenTelemetry. SHIP Hook's request flow (webhook → dispatcher → sub-plugin → GitHub API) is linear and short-lived. The GitHub event GUID propagated through logrus entries provides sufficient correlation. Revisit when ship-hook gains cross-service communication or long-running workflows.

### Guidelines for Sub-Plugin Authors

- **Use the provided `*logrus.Entry`** — never create your own logger instance.
- **Return meaningful HandlerResult reasons** — this is how the dispatcher reports your plugin's decisions.
- **Log domain decisions at Debug** — e.g., "config resolved to label=X". Log errors at Error.
- **Do not register Prometheus metrics** — the dispatcher tracks handler duration for you.
- **Do not log event lifecycle** — no "received event" or "handler complete" messages. The dispatcher handles this.

### Guidelines for Operators

- **Expose metrics** by ensuring `--metrics-port` is reachable (default 9090). All Prow GitHub client metrics and dispatcher metrics are available at `/metrics`.
- **Use `--log-level=debug`** for diagnosing sub-plugin behavior. At debug level, every event shows which plugins were skipped, dispatched to, and why each plugin considered the event relevant or irrelevant.
- **Health probes** are on `--health-port` (default 8081). Use `/healthz` for liveness and `/healthz/ready` for readiness in Kubernetes pod specs.

## Implementation

### Wire up Prow infrastructure in `cmd/ship-hook/main.go`

1. Add `flagutil.InstrumentationOptions` to the options struct and bind its flags.
2. Call `metrics.ExposeMetrics("ship-hook", config.PushGateway{}, o.instrumentationOptions.MetricsPort)` at startup.
3. Call `pjutil.NewHealthOnPort(o.instrumentationOptions.HealthPort)` with `health.ServeReady()`.

### Add handler duration histogram in `pkg/dispatch/dispatch.go`

Register a Prometheus histogram:
```
shiphook_plugin_handle_duration_seconds{event_type, plugin, took_action}
```

Record duration in each dispatcher handler method, using `HandlerResult.Relevant` to set `took_action`.

### Update deployment manifests

Add metrics and health ports to the container spec. Add liveness and readiness probes.

## Verification

- `make verify` passes
- `curl :9090/metrics` shows Prow GitHub client metrics, webhook metrics, and ship-hook dispatcher metrics
- `curl :8081/healthz` returns OK
- `curl :8081/healthz/ready` returns OK
- Dev server test: send events and verify `shiphook_plugin_handle_duration_seconds` histogram has entries
