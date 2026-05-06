// Example: wire gismanager's *slog.Logger to OpenTelemetry via the
// otelslog bridge so every library log record automatically carries
// the trace_id / span_id of whatever span is active when the log is
// emitted. Library logs become first-class telemetry alongside spans
// in your observability backend (Jaeger, Tempo, Honeycomb, Dash0, …).
//
// gismanager's library code already takes a *slog.Logger via
// WithLogger; this example just constructs that logger from an
// OpenTelemetry log provider rather than from the default text
// handler. No library-side changes are needed — the bridge is fully
// caller-driven.
//
// Run with an OTLP/HTTP collector reachable at OTEL_EXPORTER_OTLP_ENDPOINT
// (default http://localhost:4318):
//
//	go run . -src ./gis-data \
//	    -gs-url http://localhost:8080/geoserver -gs-user admin -gs-pass geoserver \
//	    -pg-host localhost -pg-port 5432 -pg-db gis -pg-user postgres -pg-pass postgres
//
// Pipe the records into any backend that speaks OTLP. See
// docs/observability.md for collector setup.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hishamkaram/gismanager/v2/publish"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func main() {
	srcPath := flag.String("src", "", "source directory to publish")
	gsURL := flag.String("gs-url", "", "GeoServer URL")
	gsUser := flag.String("gs-user", "admin", "GeoServer user")
	gsPass := flag.String("gs-pass", "geoserver", "GeoServer password")
	pgHost := flag.String("pg-host", "localhost", "PostGIS host")
	pgPort := flag.Uint("pg-port", 5432, "PostGIS port")
	pgDB := flag.String("pg-db", "gis", "PostGIS database")
	pgUser := flag.String("pg-user", "postgres", "PostGIS user")
	pgPass := flag.String("pg-pass", "postgres", "PostGIS password")
	flag.Parse()

	if *srcPath == "" || *gsURL == "" {
		slog.Error("usage: at minimum -src and -gs-url are required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Wire the OpenTelemetry log provider. Reads the standard
	// OTEL_EXPORTER_OTLP_* env vars for endpoint + headers so this
	// works against any OTLP-speaking collector unchanged.
	exp, err := otlploghttp.New(ctx)
	if err != nil {
		slog.Error("otlploghttp.New", "err", err)
		os.Exit(1)
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("gismanager"),
		semconv.ServiceVersion("dev"),
	))
	if err != nil {
		slog.Error("resource.Merge", "err", err)
		os.Exit(1)
	}

	provider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(exp)),
		log.WithResource(res),
	)
	defer func() {
		// Flush + shutdown in the deferred path so exit-on-error in
		// publish still drains the OTLP buffer.
		if err := provider.Shutdown(context.Background()); err != nil {
			slog.Error("provider.Shutdown", "err", err)
		}
	}()

	// Construct the *slog.Logger that gismanager will use. Every
	// library Debug/Info/Warn/Error becomes an OTLP log record with
	// the active span's trace_id/span_id automatically attached.
	// Passing the provider explicitly via WithLoggerProvider means we
	// don't need to call any global setter — the bridge wires straight
	// through. (If a future caller wants the global, they can add
	// `global.SetLoggerProvider(provider)` from the
	// go.opentelemetry.io/otel/log/global package.)
	logger := otelslog.NewLogger("gismanager",
		otelslog.WithLoggerProvider(provider))

	mgr, err := publish.New(
		publish.WithLogger(logger),
		publish.WithGeoserver(publish.GeoserverConfig{
			ServerURL:     *gsURL,
			Username:      *gsUser,
			Password:      *gsPass,
			WorkspaceName: "demo",
		}),
		publish.WithDatastore(publish.DatastoreConfig{
			Host:   *pgHost,
			Port:   *pgPort,
			DBName: *pgDB,
			DBUser: *pgUser,
			DBPass: *pgPass,
			Name:   "gismanager_demo",
		}),
		publish.WithSource(publish.SourceConfig{Path: *srcPath}),
	)
	if err != nil {
		logger.Error("construct manager", "err", err)
		os.Exit(1)
	}

	if err := mgr.PublishAll(ctx); err != nil {
		logger.Error("publish", "err", err)
		os.Exit(1)
	}
	logger.Info("publish ok")
}
