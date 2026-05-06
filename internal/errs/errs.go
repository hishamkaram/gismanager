// Package errs is gismanager's typed-error machinery — the *GISError
// envelope and the package-level Err* sentinels. Moved here from the
// root gismanager package as part of the v2 restructure groundwork
// (Phase 1) so the future convert/ and publish/ subpackages can
// reference these without a circular import through the root.
//
// External callers continue to use gismanager.GISError /
// gismanager.ErrConvertFailed / etc. via the type aliases and var
// aliases the root package re-exports. The package name `errs` is
// not part of the public v1.x API surface; it is only directly
// imported by other packages inside this module.
//
// In v2, the convert/ and publish/ subpackages import this package
// directly (e.g. errs.NewGISError) rather than the root package's
// alias.
package errs

import (
	"errors"
	"fmt"
)

// Sentinel errors. Match via [errors.Is] — never compare error strings.
//
// These wrap higher-level failure categories. The library wraps them around
// the underlying cause (GDAL CGo error, *geoserver.APIError, lib/pq error,
// etc.) so callers can branch on the category while still recovering the
// original via [errors.As].
var (
	// ErrConfigInvalid signals that the YAML config is syntactically valid
	// but semantically wrong (missing fields, malformed values, etc.).
	ErrConfigInvalid = errors.New("gismanager: config invalid")

	// ErrUnsupportedFormat signals that a path's extension does not map to
	// any GDAL OGR driver gismanager knows about.
	ErrUnsupportedFormat = errors.New("gismanager: unsupported format")

	// ErrInvalidLayer signals that a *GdalLayer was nil or had a nil
	// embedded *gdal.Layer pointer when an operation required a real layer.
	ErrInvalidLayer = errors.New("gismanager: invalid layer")

	// ErrInvalidDatasource signals that a *gdal.DataSource passed in was
	// nil or unusable.
	ErrInvalidDatasource = errors.New("gismanager: invalid datasource")

	// ErrPostGISConnect signals that the PostGIS ping/handshake failed.
	// The underlying *pq.Error (or driver-level error) is recoverable via
	// [errors.As].
	ErrPostGISConnect = errors.New("gismanager: postgis connect")

	// ErrGeoServerPublish signals that the GeoServer publish flow failed
	// at workspace, datastore, or feature-type creation. The underlying
	// *geoserver.APIError is recoverable via [errors.As].
	ErrGeoServerPublish = errors.New("gismanager: geoserver publish")

	// ErrNoSourcesFound signals that the configured source directory
	// contained no supported GIS files.
	ErrNoSourcesFound = errors.New("gismanager: no sources found")

	// ErrConvertFailed signals that a vector or raster conversion
	// (ConvertVector / ConvertRaster / ToCOG / ReprojectRaster) failed.
	// The underlying GDAL error is recoverable via [errors.As] into
	// [*GISError]; the GISError.Op field disambiguates which entry
	// point produced the error.
	ErrConvertFailed = errors.New("gismanager: convert failed")
)

// GISError is the typed error gismanager returns for higher-level
// operations. It carries the operation name, the source path (file, layer
// name, or workspace as appropriate), the sentinel category, and the
// underlying cause.
//
// Match by sentinel:
//
//	if errors.Is(err, gismanager.ErrUnsupportedFormat) { ... }
//
// Inspect for fields:
//
//	var gerr *gismanager.GISError
//	if errors.As(err, &gerr) {
//	    log.Println(gerr.Op, gerr.Source)
//	}
type GISError struct {
	// Op is the operation name (e.g. "OpenSource", "PublishGeoserverLayer",
	// "LoadToPostGIS"). Useful for logging and triage.
	Op string

	// Source identifies what the operation acted on — typically a file
	// path, a layer name, or a workspace name. Empty if not applicable.
	Source string

	// Sentinel is one of the package-level Err* values. errors.Is(e,
	// Sentinel) returns true.
	Sentinel error

	// Cause is the underlying error. May be nil if the failure is
	// purely a sentinel (e.g. ErrInvalidLayer with no nested cause).
	Cause error
}

// Error returns a stable, parseable message of the form
//
//	gismanager: <Op> <Source>: <sentinel>: <cause>
//
// Source is omitted if empty; Cause is omitted if nil.
func (e *GISError) Error() string {
	switch {
	case e.Source == "" && e.Cause == nil:
		return fmt.Sprintf("gismanager: %s: %v", e.Op, e.Sentinel)
	case e.Source == "":
		return fmt.Sprintf("gismanager: %s: %v: %v", e.Op, e.Sentinel, e.Cause)
	case e.Cause == nil:
		return fmt.Sprintf("gismanager: %s %q: %v", e.Op, e.Source, e.Sentinel)
	default:
		return fmt.Sprintf("gismanager: %s %q: %v: %v", e.Op, e.Source, e.Sentinel, e.Cause)
	}
}

// Unwrap returns the underlying cause for [errors.Is] / [errors.As]
// traversal. It walks past Sentinel — Sentinel is matched separately by
// [GISError.Is].
func (e *GISError) Unwrap() error { return e.Cause }

// Is reports whether the error matches the given target. It returns true
// for the GISError's Sentinel so callers can match by category:
//
//	errors.Is(err, gismanager.ErrUnsupportedFormat)
func (e *GISError) Is(target error) bool {
	return target != nil && e.Sentinel == target
}

// NewGISError constructs a *GISError. Callers wrap underlying failures
// with a sentinel category so the public surface is consistent. The
// root gismanager package keeps a `newGISError` package-private wrapper
// around this for the duration of the v1.x line; convert/ and publish/
// subpackages call this directly.
func NewGISError(op, source string, sentinel, cause error) *GISError {
	return &GISError{Op: op, Source: source, Sentinel: sentinel, Cause: cause}
}
