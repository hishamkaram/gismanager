# Observability

gismanager's library code emits structured logs via `*slog.Logger`
throughout the publish pipeline and the conversion subsystem. Every
public method takes — or threads — a `*slog.Logger` configured at
construction time, so callers control log handling end-to-end.

This doc covers two integration patterns:

1. **Plain `slog`** — the default. Stderr text or JSON; no extra deps.
2. **OpenTelemetry** — log records with trace correlation, suitable
   for production observability backends (Jaeger / Tempo / Honeycomb
   / Dash0 / etc.) via the `otelslog` bridge.

## 1. Plain slog

The library default (`slogx.Default()`) writes text-handler
records to stderr with default level filtering. To customize, build
your own `*slog.Logger` and pass it on construction:

```go
import (
    "log/slog"
    "os"
)

logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

mgr, _ := publish.New(publish.WithLogger(logger), /* ... */)
```

Library log calls follow standard Go conventions — `key`-`value`
pairs, lower_snake or single-word keys, `err` for the error field.
Sample lines:

```
time=2026-05-06T12:00:00Z level=INFO msg="read features" count=148
time=2026-05-06T12:00:01Z level=ERROR msg="open source" path=/data/x.shp err="..."
time=2026-05-06T12:00:02Z level=ERROR msg="ensure workspace" workspace=demo err="..."
```

## 2. OpenTelemetry (recommended for prod)

The library does **not** depend on OpenTelemetry directly — keeping
the runtime dep surface small is a project policy (see CLAUDE.md).
But because the library accepts an arbitrary `*slog.Logger`, an
OpenTelemetry-backed logger drops in cleanly via the
[`otelslog` bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog).

### Architecture

```
gismanager library code
       │
       ▼ slog.Logger.Debug/Info/Warn/Error
otelslog handler  ──►  OpenTelemetry log provider
                           │
                           ▼
                    OTLP/HTTP or OTLP/gRPC
                           │
                           ▼
                  collector / backend (Jaeger, Tempo, Honeycomb, …)
```

Every library log record is wrapped with the active span's
`trace_id` / `span_id` (when one is active), so logs and traces
correlate in the backend without any further library changes.

### Runnable example

See [`examples/otel_pipeline/`](../examples/otel_pipeline/) — a
~120-line `main.go` that wires `otelslog.NewLogger`, the
`otlploghttp` exporter, and a `*publish.Manager.PublishAll` call. It
lives in its own Go submodule so the OTel SDK + exporter
dependencies don't leak into gismanager's top-level `go.mod`.

### Kubernetes-flavored deployment

A typical production deploy:

1. Run the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector-contrib)
   as a `DaemonSet` (one per node) or `Deployment`. Configure it
   with the OTLP receiver and your backend's exporter.
2. Set on the gismanager Pod:

   ```yaml
   env:
     - name: OTEL_EXPORTER_OTLP_ENDPOINT
       value: "http://otel-collector.otel.svc.cluster.local:4318"
     - name: OTEL_RESOURCE_ATTRIBUTES
       value: "service.name=gismanager,service.instance.id=$(HOSTNAME)"
   ```

3. Adopt the `examples/otel_pipeline/main.go` wiring at process
   startup (or vendor the relevant ~30 lines into your own
   long-running publish loop).

The library code itself is unaware — every log it makes via the
plumbed-in logger flows out through the bridge with full trace
context.

## Log-key conventions (current state)

The library's existing log calls predate this doc and use keys that
mostly match Go conventions but don't strictly follow OpenTelemetry
semantic conventions. Examples:

| Library key | OTel-conventional equivalent | Status |
|-------------|------------------------------|--------|
| `path` | `file.path` | not yet renamed |
| `src` | `file.path` | not yet renamed |
| `dst` | `file.path` (output) | not yet renamed |
| `workspace`, `datastore`, `layer` | `geoserver.workspace`, `geoserver.datastore`, `geoserver.layer` | not yet renamed |
| `database` | `db.name` | not yet renamed |
| `err` | (Go convention; OTel uses `error`) | kept |

A future PR (tracked in the v1.4+ backlog) audits these and renames
where safe — non-breaking for callers since log-key names aren't
part of the public API contract. Until then, use OTel processor
rules in your collector to remap keys at ingest time:

```yaml
processors:
  attributes:
    actions:
      - key: file.path
        from_attribute: path
        action: upsert
      - key: path
        action: delete
```

## Performance notes

- The `otelslog` bridge has measured overhead of ~1 µs per log
  record on a modern x86 (per the OTel benchmarks). For the publish
  pipeline (which logs O(layers) records, not O(features)), this is
  imperceptible.
- Use a `BatchProcessor` (not `SimpleProcessor`) on the log provider
  in production — the example uses batch by default. Simple flushes
  every record synchronously and slows the publish flow noticeably
  on networks with non-zero RTT to the collector.
- The OTLP/HTTP exporter's default queue holds 2048 records; size
  it up via `WithMaxQueueSize` if you publish many small layers in a
  burst.
