package gismanager

// Compatibility shim for the v1.x publish-pipeline API.
//
// The publish pipeline (Manager constructor, Walk + PublishAll
// orchestration, GdalLayer wrapper, the connection helpers, the
// configuration types) moved to the [publish] subpackage as part of
// the v2 restructure groundwork (see Phase 3 in
// ~/.claude/plans/how-can-we-improve-steady-emerson.md).
//
// This file re-exports every public symbol from the publish subpackage
// at its v1.x import path. Type aliases preserve full identity —
// errors.As, errors.Is, method sets, slice element types all
// interoperate transparently across the boundary. Function wrappers
// are pure pass-throughs with no behavior change.
//
// Notable v1 → v2 type renames preserved here as aliases:
//
//   - gismanager.ManagerConfig → publish.Manager
//   - gismanager.GdalLayer → publish.Layer
//
// The "Config" suffix on Manager was a misnomer — the type IS the
// manager, not its config. Same for the redundant "Gdal" prefix on
// Layer (it's clear from package context). v2 (Phase 4) drops this
// file entirely; new code should import the publish subpackage
// directly.

import (
	"context"
	"log/slog"

	"github.com/hishamkaram/gismanager/publish"
)

// =============================================================================
// Type aliases
// =============================================================================

// ManagerConfig is preserved as a v1.x alias of [publish.Manager].
//
// Deprecated: use [publish.Manager] directly via
// `import "github.com/hishamkaram/gismanager/publish"`.
type ManagerConfig = publish.Manager

// GdalLayer is preserved as a v1.x alias of [publish.Layer].
//
// Deprecated: use [publish.Layer] directly.
type GdalLayer = publish.Layer

// GeoserverConfig is preserved as a v1.x alias of [publish.GeoserverConfig].
//
// Deprecated: use [publish.GeoserverConfig] directly.
type GeoserverConfig = publish.GeoserverConfig

// SourceConfig is preserved as a v1.x alias of [publish.SourceConfig].
//
// Deprecated: use [publish.SourceConfig] directly.
type SourceConfig = publish.SourceConfig

// DatastoreConfig is preserved as a v1.x alias of [publish.DatastoreConfig].
//
// Deprecated: use [publish.DatastoreConfig] directly.
type DatastoreConfig = publish.DatastoreConfig

// Option is preserved as a v1.x alias of [publish.Option].
//
// Deprecated: use [publish.Option] directly.
type Option = publish.Option

// WalkItem is preserved as a v1.x alias of [publish.WalkItem].
//
// Deprecated: use [publish.WalkItem] directly.
type WalkItem = publish.WalkItem

// LayerField is preserved as a v1.x alias of [publish.LayerField].
//
// Deprecated: use [publish.LayerField] directly.
type LayerField = publish.LayerField

// =============================================================================
// Top-level function wrappers — all forward to the publish subpackage.
// All Deprecated.
// =============================================================================

// New forwards to [publish.New].
//
// Deprecated: use [publish.New] directly.
func New(opts ...Option) (*ManagerConfig, error) {
	return publish.New(opts...)
}

// FromConfig forwards to [publish.FromConfig].
//
// Deprecated: use [publish.FromConfig] directly.
func FromConfig(configFile string) (*ManagerConfig, error) {
	return publish.FromConfig(configFile)
}

// WithLogger forwards to [publish.WithLogger].
//
// Deprecated: use [publish.WithLogger] directly.
func WithLogger(l *slog.Logger) Option {
	return publish.WithLogger(l)
}

// WithGeoserver forwards to [publish.WithGeoserver].
//
// Deprecated: use [publish.WithGeoserver] directly.
func WithGeoserver(cfg GeoserverConfig) Option {
	return publish.WithGeoserver(cfg)
}

// WithDatastore forwards to [publish.WithDatastore].
//
// Deprecated: use [publish.WithDatastore] directly.
func WithDatastore(cfg DatastoreConfig) Option {
	return publish.WithDatastore(cfg)
}

// WithSource forwards to [publish.WithSource].
//
// Deprecated: use [publish.WithSource] directly.
func WithSource(cfg SourceConfig) Option {
	return publish.WithSource(cfg)
}

// GetGISFiles forwards to [publish.GetGISFiles].
//
// Deprecated: use [publish.GetGISFiles] directly.
func GetGISFiles(root string) ([]string, error) {
	return publish.GetGISFiles(root)
}

// DBIsAlive forwards to [publish.DBIsAlive].
//
// Deprecated: use [publish.DBIsAlive] directly.
func DBIsAlive(dbType, connectionStr string) error {
	return publish.DBIsAlive(dbType, connectionStr)
}

// DBIsAliveContext forwards to [publish.DBIsAliveContext].
//
// Deprecated: use [publish.DBIsAliveContext] directly.
func DBIsAliveContext(ctx context.Context, dbType, connectionStr string) error {
	return publish.DBIsAliveContext(ctx, dbType, connectionStr)
}
