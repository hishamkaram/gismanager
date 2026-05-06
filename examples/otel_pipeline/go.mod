// This example lives in its own Go module so that adding OpenTelemetry
// runtime dependencies (the trace SDK, the OTLP HTTP exporter, the
// otelslog bridge, and their transitive deps) does NOT leak into the
// main gismanager module — per CLAUDE.md, the project's runtime
// dependency surface is intentionally small (lukeroth/gdal, lib/pq,
// hishamkaram/geoserver/v2, gopkg.in/yaml.v3, stdlib).
//
// To run from the repo root:
//
//	cd examples/otel_pipeline
//	go run .  -src ./gis-data
//
// The `replace github.com/hishamkaram/gismanager => ../..` directive
// pins this example to the parent checkout's gismanager source so it
// always builds against current master without needing a published
// version.

module github.com/hishamkaram/gismanager/examples/otel_pipeline

go 1.25

require (
	github.com/hishamkaram/gismanager v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/contrib/bridges/otelslog v0.7.0
	go.opentelemetry.io/otel v1.32.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp v0.8.0
	go.opentelemetry.io/otel/sdk v1.32.0
	go.opentelemetry.io/otel/sdk/log v0.8.0
)

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.23.0 // indirect
	github.com/hishamkaram/geoserver/v2 v2.0.0 // indirect
	github.com/lib/pq v1.12.3 // indirect
	github.com/lukeroth/gdal v0.0.0-20251112192847-aa5e8dc032a2 // indirect
	go.opentelemetry.io/otel/log v0.8.0 // indirect
	go.opentelemetry.io/otel/metric v1.32.0 // indirect
	go.opentelemetry.io/otel/trace v1.32.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.27.0 // indirect
	golang.org/x/text v0.20.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241104194629-dd2ea8efbc28 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241104194629-dd2ea8efbc28 // indirect
	google.golang.org/grpc v1.67.1 // indirect
	google.golang.org/protobuf v1.35.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/hishamkaram/gismanager => ../..
