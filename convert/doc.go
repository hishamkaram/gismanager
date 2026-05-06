// Package convert is gismanager's stateless conversion subsystem —
// thin Go wrappers over GDAL's `ogr2ogr` / `gdal_translate` / `gdalwarp`
// / `gdal_rasterize` / `gdalbuildvrt` / `gdaldem` C entry points, plus
// PMTiles archive conversion via [protomaps/go-pmtiles].
//
// Entry points:
//
//   - [ConvertVector] (the `ogr2ogr` equivalent)
//   - [ConvertRaster] / [ToCOG] (the `gdal_translate` equivalent + COG helper)
//   - [ReprojectRaster] (the `gdalwarp` equivalent)
//   - [Rasterize] (the `gdal_rasterize` equivalent: vector → raster)
//   - [BuildVRT] (the `gdalbuildvrt` equivalent: multi-raster mosaic)
//   - [DEMProcessing] (the `gdaldem` equivalent: hillshade / slope / aspect / etc.)
//   - [ToPMTiles] (MBTiles → PMTiles archive)
//
// Each entry point has a matching `*Option` type and `WithX*` functional
// option helpers ([VectorOption] / [RasterOption] / [RasterizeOption] /
// [VRTOption] / [DEMOption] / [PMTilesOption]). Pass any number of
// options in any order; nil options are tolerated.
//
// Errors wrap [github.com/hishamkaram/gismanager/internal/errs.ErrConvertFailed];
// recover the underlying GDAL or filesystem error via [errors.As] into
// [*errs.GISError]. The Op field on the returned *GISError disambiguates
// which entry point produced the failure.
//
// History: this package split out of the root gismanager package as part
// of the v2 restructure (Phase 2). v1.x callers can continue to use the
// v1 names (gismanager.ConvertVector etc.) — those are now thin
// [Deprecated] wrappers that delegate here. v2 (Phase 4) drops the
// wrappers; v2 callers import this package directly.
package convert
