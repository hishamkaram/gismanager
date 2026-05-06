// Package publish is gismanager's publish-pipeline subpackage —
// directory walk, OGR-based PostGIS load, and GeoServer feature-type
// registration via the [hishamkaram/geoserver/v2] REST client.
//
// Entry points:
//
//   - [New] / [FromConfig] construct a *Manager.
//   - [(*Manager).Walk] yields layers from the configured source.
//   - [(*Manager).PublishAll] runs the full pipeline (walk → load →
//     publish) with bounded GeoServer-publish concurrency.
//   - [(*Manager).OpenSource] / [(*Manager).GetDriver] are read-only
//     inspection helpers for use in CLIs / scripts.
//   - [(*Manager).Validate] checks required fields against the
//     publish-pipeline contract (ErrConfigInvalid envelope).
//
// Configuration:
//
//   - [Manager] holds the merged configuration; constructed once via
//     [New] (functional options) or [FromConfig] (YAML, with ${VAR}
//     env-var interpolation).
//   - [GeoserverConfig], [DatastoreConfig], [SourceConfig] are the
//     three sub-config struct types YAML decodes into.
//
// Errors wrap the sentinels in [github.com/hishamkaram/gismanager/internal/errs]:
// ErrConfigInvalid (Validate failures), ErrUnsupportedFormat (GetDriver),
// ErrInvalidLayer / ErrInvalidDatasource (LayerToPostgis), ErrPostGISConnect
// (DBIsAlive), ErrGeoServerPublish (the publish flow). Recover the
// underlying *geoserver.APIError or *pq.Error via [errors.As] into
// [*errs.GISError].
//
// History: this package split out of the root gismanager package as
// part of the v2 restructure (Phase 3). v1.x callers can continue to
// use the v1 names (gismanager.ManagerConfig → publish.Manager,
// gismanager.GdalLayer → publish.Layer, gismanager.New / FromConfig
// / WithLogger / etc.) — those are now thin Deprecated wrappers in
// the root package that delegate here. v2 (Phase 4) drops the
// wrappers; v2 callers import this package directly.
package publish
