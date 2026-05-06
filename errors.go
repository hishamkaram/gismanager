package gismanager

import "github.com/hishamkaram/gismanager/internal/errs"

// This file is a v1.x compatibility shim. The actual implementation of
// the typed-error machinery lives in [internal/errs] as of the v2
// restructure groundwork (Phase 1). External callers continue to use
// the gismanager-prefixed names (gismanager.GISError,
// gismanager.ErrConvertFailed, etc.) — those work via the type
// aliases and var aliases below. Internal callers within the root
// package call the package-private newGISError wrapper, which
// delegates to errs.NewGISError. v2 will drop this file when the
// root package is emptied of all symbols.

// GISError aliases [internal/errs.GISError]. See that type for full
// documentation. v1.x users should keep importing [GISError] from
// this package; the alias means errors.As / errors.Is / pointer
// equality all work transparently across the boundary.
type GISError = errs.GISError

// Sentinel errors aliased from [internal/errs]. The underlying values
// are the SAME error instances — `errors.Is(err, gismanager.ErrConvertFailed)`
// and `errors.Is(err, errs.ErrConvertFailed)` both work because they
// reference the same underlying *errors.errorString.
var (
	ErrConfigInvalid     = errs.ErrConfigInvalid
	ErrUnsupportedFormat = errs.ErrUnsupportedFormat
	ErrInvalidLayer      = errs.ErrInvalidLayer
	ErrInvalidDatasource = errs.ErrInvalidDatasource
	ErrPostGISConnect    = errs.ErrPostGISConnect
	ErrGeoServerPublish  = errs.ErrGeoServerPublish
	ErrNoSourcesFound    = errs.ErrNoSourcesFound
	ErrConvertFailed     = errs.ErrConvertFailed
)

// newGISError is the internal call-site wrapper around
// [internal/errs.NewGISError]. Keeping it here means every existing
// caller in the root package (manager.go / layer.go / walk.go /
// utils.go / convert_*.go) is unchanged in Phase 1 — they continue to
// call newGISError as before, which now delegates to the moved impl.
//
// Phase 2+ moves the convert_*.go callers to a separate subpackage
// where they call errs.NewGISError directly; same for Phase 3 with
// publish/. Phase 4 drops this wrapper when no root caller remains.
func newGISError(op, source string, sentinel, cause error) *GISError {
	return errs.NewGISError(op, source, sentinel, cause)
}
