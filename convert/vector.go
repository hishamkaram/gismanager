package convert

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/internal/errs"
	"github.com/hishamkaram/gismanager/internal/slogx"
)

// VectorOption configures a [ConvertVector] call. Construct via
// the [WithVector*] helpers below; pass any number of options in any
// order. Each helper mutates a private config struct; nil options are
// tolerated.
//
// The pattern matches v1.1's manager [Option] precedent — functional
// options for additive growth without breaking the public surface.
type VectorOption func(*vectorConfig)

type vectorConfig struct {
	logger       *slog.Logger
	format       string
	sourceSRS    string
	targetSRS    string
	bbox         *vectorBBox
	where        string
	simplifyTol  float64
	selectFields []string
	layerName    string
	overwrite    bool
	rawOptions   []string
}

type vectorBBox struct {
	MinX, MinY, MaxX, MaxY float64
}

func newVectorConfig(opts []VectorOption) *vectorConfig {
	c := &vectorConfig{logger: slogx.Default()}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// WithVectorLogger sets the structured logger used for diagnostic output
// during conversion. Passing nil falls back to the default logger from
// [slogx.Default]. Mirrors v1.1's [WithLogger] semantics for the manager.
func WithVectorLogger(l *slog.Logger) VectorOption {
	return func(c *vectorConfig) {
		if l == nil {
			c.logger = slogx.Default()
			return
		}
		c.logger = l
	}
}

// WithVectorFormat selects the OGR output driver by name (e.g. "GPKG",
// "GeoJSON", "FlatGeobuf", "ESRI Shapefile", "KML"). Maps to ogr2ogr's
// `-f` flag. When empty, GDAL infers the driver from the destination
// path's extension.
func WithVectorFormat(driverName string) VectorOption {
	return func(c *vectorConfig) { c.format = driverName }
}

// WithVectorSourceSRS overrides the input CRS. Accepts any GDAL
// SetFromUserInput-friendly form ("EPSG:4326", "+proj=longlat ...",
// well-known WKT). Use only when the source has a missing or wrong CRS.
// Maps to ogr2ogr's `-s_srs`.
func WithVectorSourceSRS(srs string) VectorOption {
	return func(c *vectorConfig) { c.sourceSRS = srs }
}

// WithVectorTargetSRS reprojects features to the given CRS during the
// conversion. The most common use is `WithVectorTargetSRS("EPSG:3857")`
// for web tile pipelines. Maps to ogr2ogr's `-t_srs`.
func WithVectorTargetSRS(srs string) VectorOption {
	return func(c *vectorConfig) { c.targetSRS = srs }
}

// WithVectorBoundingBox restricts output to features intersecting the
// given bounding box. Coordinates are in the *source* CRS unless
// [WithVectorSourceSRS] also reinterprets them. Maps to ogr2ogr's
// `-spat`.
func WithVectorBoundingBox(minX, minY, maxX, maxY float64) VectorOption {
	return func(c *vectorConfig) {
		c.bbox = &vectorBBox{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
	}
}

// WithVectorWhere restricts output to features matching the given OGR
// SQL WHERE expression (without the `WHERE` keyword), e.g.
// `"CONTINENT = 'Africa'"`. Maps to ogr2ogr's `-where`.
func WithVectorWhere(sql string) VectorOption {
	return func(c *vectorConfig) { c.where = sql }
}

// WithVectorSimplify applies Douglas-Peucker simplification with the
// given tolerance (in target-CRS units). Useful for thinning vector
// data before web delivery. Maps to ogr2ogr's `-simplify`.
func WithVectorSimplify(tolerance float64) VectorOption {
	return func(c *vectorConfig) { c.simplifyTol = tolerance }
}

// WithVectorSelectFields restricts the output to the named attribute
// fields. The geometry column is always preserved. Maps to ogr2ogr's
// `-select`.
func WithVectorSelectFields(fields ...string) VectorOption {
	return func(c *vectorConfig) { c.selectFields = append([]string(nil), fields...) }
}

// WithVectorLayerName renames the destination layer (e.g. forces a
// different table name when copying to PostGIS or a different layer
// name in GPKG). Maps to ogr2ogr's `-nln`.
func WithVectorLayerName(name string) VectorOption {
	return func(c *vectorConfig) { c.layerName = name }
}

// WithVectorOverwrite replaces an existing destination layer instead of
// failing with a "layer already exists" error. Maps to ogr2ogr's
// `-overwrite`.
func WithVectorOverwrite() VectorOption {
	return func(c *vectorConfig) { c.overwrite = true }
}

// WithVectorRawOptions appends raw `ogr2ogr`-style flags to the
// generated argument list, after every other option. Use this to reach
// flags the typed helpers do not yet expose (e.g. `-lco`, `-skipfailures`,
// `-nlt`). Each entry is one CLI token.
func WithVectorRawOptions(args ...string) VectorOption {
	return func(c *vectorConfig) { c.rawOptions = append(c.rawOptions, args...) }
}

// ConvertVector converts the vector source at src into dst, applying any
// of the provided options. It is a thin wrapper around the GDAL C entry
// point behind `ogr2ogr` ([gdal.VectorTranslate]) — every option maps
// 1:1 to an ogr2ogr CLI flag.
//
// Behavior:
//   - The destination driver is chosen by [WithVectorFormat] when set,
//     else inferred from dst's extension (GDAL behavior).
//   - Reprojection happens during the copy when [WithVectorTargetSRS] is
//     set; chain with [WithVectorBoundingBox] / [WithVectorWhere] /
//     [WithVectorSimplify] for the typical "extract a region" pipeline.
//   - dst and src paths can use any GDAL Virtual File System prefix
//     (`/vsimem/`, `/vsis3/`, `/vsicurl/`, `/vsigs/`, `/vsizip/`); the
//     function passes paths through to GDAL unchanged. Bare `.zip`
//     Shapefile bundles do NOT auto-prefix — pass `/vsizip/<path>`
//     explicitly, or extract first via [GetGISFiles].
//   - ctx is honored at the function boundary (checked before opening
//     the source); the underlying CGo call is synchronous and
//     uninterruptible — long conversions cannot be cancelled mid-way.
//
// Errors are wrapped with [errs.ErrConvertFailed]; recover the underlying
// GDAL error via [errors.As] into [*GISError].
//
// Driver names supplied via [WithVectorFormat] are validated against
// the running GDAL build before the C call, so an unknown driver
// surfaces as a clean errs.ErrConvertFailed rather than the upstream
// silent-fail-with-stderr-warning behavior.
func ConvertVector(ctx context.Context, src, dst string, opts ...VectorOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg := newVectorConfig(opts)

	if err := validateGDALDriver(cfg.format); err != nil {
		cfg.logger.Error("ConvertVector: invalid format", "format", cfg.format, "err", err)
		return errs.NewGISError("ConvertVector", src, errs.ErrConvertFailed, err)
	}

	srcDS, err := gdal.OpenEx(src, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		cfg.logger.Error("ConvertVector: open source", "src", src, "err", err)
		return errs.NewGISError("ConvertVector", src, errs.ErrConvertFailed, err)
	}
	defer srcDS.Close()

	args := buildVectorTranslateArgs(cfg)
	cfg.logger.Debug("ConvertVector: invoking VectorTranslate",
		"src", src, "dst", dst, "args", args)

	out, vErr := gdal.VectorTranslate(dst, []gdal.Dataset{srcDS}, args)
	if vErr != nil {
		return errs.NewGISError("ConvertVector", fmt.Sprintf("%s -> %s", src, dst),
			errs.ErrConvertFailed, vErr)
	}
	defer out.Close()
	return nil
}

// buildVectorTranslateArgs renders cfg into the []string ogr2ogr-style
// arg list GDALVectorTranslate expects. Unit-tested separately so the
// option->arg mapping is locked in without needing CGo.
func buildVectorTranslateArgs(cfg *vectorConfig) []string {
	var args []string
	if cfg.format != "" {
		args = append(args, "-f", cfg.format)
	}
	if cfg.overwrite {
		args = append(args, "-overwrite")
	}
	if cfg.sourceSRS != "" {
		args = append(args, "-s_srs", cfg.sourceSRS)
	}
	if cfg.targetSRS != "" {
		args = append(args, "-t_srs", cfg.targetSRS)
	}
	if cfg.bbox != nil {
		args = append(args, "-spat",
			formatFloat(cfg.bbox.MinX),
			formatFloat(cfg.bbox.MinY),
			formatFloat(cfg.bbox.MaxX),
			formatFloat(cfg.bbox.MaxY))
	}
	if cfg.where != "" {
		args = append(args, "-where", cfg.where)
	}
	if cfg.simplifyTol > 0 {
		args = append(args, "-simplify", formatFloat(cfg.simplifyTol))
	}
	if len(cfg.selectFields) > 0 {
		args = append(args, "-select", strings.Join(cfg.selectFields, ","))
	}
	if cfg.layerName != "" {
		args = append(args, "-nln", cfg.layerName)
	}
	args = append(args, cfg.rawOptions...)
	return args
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
