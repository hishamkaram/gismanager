package convert

import (
	"fmt"

	"github.com/lukeroth/gdal"
)

// validateGDALDriver checks that the named GDAL driver is registered
// in the running GDAL build. Returns nil for an empty name (the user
// wants format inference from the destination path) and a non-nil
// error otherwise.
//
// The lookup goes through the unified GDAL driver registry, which
// covers both vector (OGR) and raster drivers since GDAL 2.0 — so
// this validator works for both modalities.
//
// This closes a gap documented in v1.2's CHANGELOG known limitations:
// the lukeroth/gdal utility wrappers (VectorTranslate / Translate /
// Warp / Rasterize / DEMProcessing / BuildVRT) silently succeed
// (cerr=0, NULL dataset) when the user hands them an unknown driver
// name — they fail at C-level option parsing, log to stderr, and
// return without setting cerr. Pre-validating on the Go side surfaces
// the bad-driver case as a clean errs.ErrConvertFailed before the C call.
//
// Caller wraps the returned error with errs.NewGISError(...).
func validateGDALDriver(name string) error {
	if name == "" {
		return nil
	}
	if _, err := gdal.GetDriverByName(name); err != nil {
		return fmt.Errorf("driver %q not registered: %w", name, err)
	}
	return nil
}
