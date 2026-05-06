# OpenTelemetry × gismanager — runnable example

Wires gismanager's `*slog.Logger` to OpenTelemetry via the
[`otelslog` bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog)
so every library log record automatically carries `trace_id` /
`span_id` of the active span. Library logs become first-class
telemetry alongside spans in any OTLP-speaking backend (Jaeger,
Tempo, Honeycomb, Dash0, …).

No gismanager-side changes are required — the project already takes
a `*slog.Logger` via `WithLogger`, and the bridge is fully
caller-driven.

## Why a separate go.mod?

Per the project's `CLAUDE.md`, the gismanager runtime dependency
surface is intentionally small (`lukeroth/gdal`, `lib/pq`,
`hishamkaram/geoserver/v2`, `gopkg.in/yaml.v3`, stdlib). The
OpenTelemetry SDK + the OTLP exporter + their transitive deps would
roughly double that surface, so this example lives in its own
sub-module. The `replace github.com/hishamkaram/gismanager => ../..`
directive pins it to the parent checkout so the example always
builds against current master.

## Running

Stand up a local OTLP/HTTP collector. The OpenTelemetry Collector
demo image is the simplest path:

```sh
docker run --rm -p 4318:4318 -p 4317:4317 \
    otel/opentelemetry-collector:latest
```

Then run the example from the repo root:

```sh
cd examples/otel_pipeline
go run . \
    -src ../../testdata \
    -gs-url http://localhost:8080/geoserver \
    -pg-host localhost -pg-port 5432 -pg-db gis \
    -pg-user postgres -pg-pass postgres
```

You should see log records flowing into the collector with
`service.name=gismanager` and (when invoked from inside an active
span) `trace_id` / `span_id` attributes attached.

## OTLP endpoint configuration

The example reads the standard OpenTelemetry env vars:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` | Collector base URL |
| `OTEL_EXPORTER_OTLP_HEADERS` | (empty) | Auth headers (e.g. `api-key=…`) |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | OTLP protocol variant |

See [OpenTelemetry's env-var reference](https://opentelemetry.io/docs/specs/otel/protocol/exporter/)
for the full list.

## Production wiring

For production deploys, replace `otlploghttp.New(ctx)` with the
gRPC variant `otlploggrpc.New(ctx)` if your collector / backend
prefers gRPC, and consider attaching a service.instance.id resource
attribute (e.g. derived from `HOSTNAME` in a Kubernetes pod). See
[`docs/observability.md`](../../docs/observability.md) for the
architectural pattern and a Kubernetes-flavored deployment recipe.
