package publish

import (
	"log/slog"

	"github.com/hishamkaram/gismanager/v2/internal/slogx"
)

// Option configures a [Manager] at construction time. Pass options to
// [New] in any order.
//
// Each option is a small function that mutates the partially-constructed
// manager. The functional-options pattern matches stdlib precedent
// ([net/http.Server], [log/slog.HandlerOptions]) and keeps the public
// surface small while leaving room for additive growth — new
// configuration knobs can ship as new With* helpers without breaking
// existing callers.
type Option func(*Manager)

// WithLogger sets the structured logger the manager and its sub-operations
// use for diagnostic output. Passing nil falls back to the default logger
// returned by [slogx.Default] so callers can opt into the default explicitly.
func WithLogger(l *slog.Logger) Option {
	return func(m *Manager) {
		if l == nil {
			m.logger = slogx.Default()
			return
		}
		m.logger = l
	}
}

// WithGeoserver sets the GeoServer endpoint, credentials, and workspace
// name the manager publishes feature types into.
func WithGeoserver(cfg GeoserverConfig) Option {
	return func(m *Manager) { m.Geoserver = cfg }
}

// WithDatastore sets the PostGIS connection parameters and GeoServer
// datastore name the manager loads layers into.
func WithDatastore(cfg DatastoreConfig) Option {
	return func(m *Manager) { m.Datastore = cfg }
}

// WithSource sets the source-directory configuration the manager scans for
// supported GIS files.
func WithSource(cfg SourceConfig) Option {
	return func(m *Manager) { m.Source = cfg }
}

// New constructs a [Manager] from the given options. With no options,
// the returned manager has zero-value GeoServer / Datastore / Source
// configs and the default logger from [slogx.Default]. The error return is
// reserved — today [New] always returns nil — so future validation
// (e.g. checking required fields) is a non-breaking addition.
//
// Programmatic callers should prefer [New] over [FromConfig]; YAML-driven
// callers can still use [FromConfig], which now delegates here.
func New(opts ...Option) (*Manager, error) {
	m := &Manager{
		logger: slogx.Default(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	return m, nil
}
