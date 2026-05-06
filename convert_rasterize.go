package gismanager

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/lukeroth/gdal"
)

// RasterizeOption configures a [Rasterize] call. Construct via the
// [WithRasterize*] helpers below; pass any number of options in any
// order. Each helper mutates a private config struct; nil options are
// tolerated.
type RasterizeOption func(*rasterizeConfig)

type rasterizeConfig struct {
	logger          *slog.Logger
	format          string
	burnValues      []float64
	attribute       string
	targetResX      float64
	targetResY      float64
	hasTargetRes    bool
	extent          *rasterBounds
	sizeX           int
	sizeY           int
	hasSize         bool
	layer           string
	where           string
	creationOptions []string
	outputType      string
	rawOptions      []string
}

func newRasterizeConfig(opts []RasterizeOption) *rasterizeConfig {
	c := &rasterizeConfig{logger: GetLogger()}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// WithRasterizeLogger sets the structured logger used during rasterization.
// nil falls back to [GetLogger].
func WithRasterizeLogger(l *slog.Logger) RasterizeOption {
	return func(c *rasterizeConfig) {
		if l == nil {
			c.logger = GetLogger()
			return
		}
		c.logger = l
	}
}

// WithRasterizeFormat selects the output GDAL driver by name (e.g.
// "GTiff", "COG", "PNG"). Maps to gdal_rasterize's `-of`. When empty,
// GDAL infers from the destination path.
func WithRasterizeFormat(driver string) RasterizeOption {
	return func(c *rasterizeConfig) { c.format = driver }
}

// WithRasterizeBurnValues sets per-band burn values. One value per
// output band. Mutually exclusive with [WithRasterizeAttribute] —
// callers should use one or the other. Maps to repeated `-burn`.
func WithRasterizeBurnValues(values ...float64) RasterizeOption {
	return func(c *rasterizeConfig) { c.burnValues = append([]float64(nil), values...) }
}

// WithRasterizeAttribute burns the named attribute field's value into
// the raster (numeric attributes only). Mutually exclusive with
// [WithRasterizeBurnValues]. Maps to `-a`.
func WithRasterizeAttribute(name string) RasterizeOption {
	return func(c *rasterizeConfig) { c.attribute = name }
}

// WithRasterizeTargetResolution sets the output pixel size in target-CRS
// units. Maps to `-tr xRes yRes`. Mutually exclusive with
// [WithRasterizeOutputSize].
func WithRasterizeTargetResolution(xRes, yRes float64) RasterizeOption {
	return func(c *rasterizeConfig) {
		c.targetResX = xRes
		c.targetResY = yRes
		c.hasTargetRes = true
	}
}

// WithRasterizeOutputSize sets the output dimensions in pixels. Maps
// to `-ts`. Mutually exclusive with [WithRasterizeTargetResolution].
func WithRasterizeOutputSize(xSize, ySize int) RasterizeOption {
	return func(c *rasterizeConfig) {
		c.sizeX = xSize
		c.sizeY = ySize
		c.hasSize = true
	}
}

// WithRasterizeOutputBounds restricts the output to the given bounding
// box in the *source* CRS units. Maps to `-te xmin ymin xmax ymax`.
// When omitted, GDAL uses the source layer's extent.
func WithRasterizeOutputBounds(minX, minY, maxX, maxY float64) RasterizeOption {
	return func(c *rasterizeConfig) {
		c.extent = &rasterBounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
	}
}

// WithRasterizeLayer picks one named layer from a multi-layer source
// (e.g. a GeoPackage). Default is to rasterize all layers, which can
// produce unexpected output when the source has more than one layer.
// Maps to `-l`.
func WithRasterizeLayer(name string) RasterizeOption {
	return func(c *rasterizeConfig) { c.layer = name }
}

// WithRasterizeWhere applies an OGR SQL WHERE filter to the source
// features (without the WHERE keyword). Maps to `-where`.
func WithRasterizeWhere(sql string) RasterizeOption {
	return func(c *rasterizeConfig) { c.where = sql }
}

// WithRasterizeCreationOption appends a single driver-specific
// creation option (e.g. "COMPRESS=DEFLATE"). May be called multiple
// times. Maps to repeated `-co KEY=VAL`.
func WithRasterizeCreationOption(key, val string) RasterizeOption {
	return func(c *rasterizeConfig) {
		c.creationOptions = append(c.creationOptions, fmt.Sprintf("%s=%s", key, val))
	}
}

// WithRasterizeOutputType sets the output pixel data type (e.g.
// "Byte", "UInt16", "Float32"). Maps to `-ot`.
func WithRasterizeOutputType(t string) RasterizeOption {
	return func(c *rasterizeConfig) { c.outputType = t }
}

// WithRasterizeRawOptions appends raw `gdal_rasterize`-style flag
// tokens to the generated argument list. Use this to reach flags
// the typed helpers do not expose (e.g. `-init`, `-a_nodata`,
// `-add`).
func WithRasterizeRawOptions(args ...string) RasterizeOption {
	return func(c *rasterizeConfig) { c.rawOptions = append(c.rawOptions, args...) }
}

// Rasterize burns the vector geometries from vectorSrc into the raster
// destination at rasterDst. Thin wrapper around the GDAL C entry
// point behind `gdal_rasterize` ([gdal.Rasterize]).
//
// Pick one of [WithRasterizeBurnValues] (constant value per band) or
// [WithRasterizeAttribute] (per-feature value from an attribute field).
// Without either, GDAL defaults to burning value 255 into a single
// band — almost certainly not what callers want; the binding does
// not error in that case.
//
// Pick one of [WithRasterizeTargetResolution] (pixel size in CRS
// units) or [WithRasterizeOutputSize] (pixel dimensions). Without
// either, GDAL chooses an arbitrary 256x256 default.
//
// Errors are wrapped with [ErrConvertFailed]; recover via
// [errors.As] into [*GISError]. The Op field is "Rasterize".
//
// Same binding gaps documented on [ConvertVector] apply: ctx is
// honored only at the function boundary, and the underlying CGo call
// is synchronous.
func Rasterize(ctx context.Context, vectorSrc, rasterDst string, opts ...RasterizeOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg := newRasterizeConfig(opts)

	if err := validateGDALDriver(cfg.format); err != nil {
		cfg.logger.Error("Rasterize: invalid format", "format", cfg.format, "err", err)
		return newGISError("Rasterize", vectorSrc, ErrConvertFailed, err)
	}

	srcDS, err := gdal.OpenEx(vectorSrc, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		cfg.logger.Error("Rasterize: open source", "src", vectorSrc, "err", err)
		return newGISError("Rasterize", vectorSrc, ErrConvertFailed, err)
	}
	defer srcDS.Close()

	args := buildRasterizeArgs(cfg)
	cfg.logger.Debug("Rasterize: invoking",
		"src", vectorSrc, "dst", rasterDst, "args", args)

	out, rErr := gdal.Rasterize(rasterDst, srcDS, args)
	if rErr != nil {
		return newGISError("Rasterize", fmt.Sprintf("%s -> %s", vectorSrc, rasterDst),
			ErrConvertFailed, rErr)
	}
	defer out.Close()
	return nil
}

// buildRasterizeArgs renders cfg into the []string gdal_rasterize-style
// argument list. Unit-tested separately so the option->arg mapping is
// locked in without CGo.
func buildRasterizeArgs(cfg *rasterizeConfig) []string {
	var args []string
	if cfg.format != "" {
		args = append(args, "-of", cfg.format)
	}
	if cfg.outputType != "" {
		args = append(args, "-ot", cfg.outputType)
	}
	for _, v := range cfg.burnValues {
		args = append(args, "-burn", formatFloat(v))
	}
	if cfg.attribute != "" {
		args = append(args, "-a", cfg.attribute)
	}
	if cfg.layer != "" {
		args = append(args, "-l", cfg.layer)
	}
	if cfg.where != "" {
		args = append(args, "-where", cfg.where)
	}
	if cfg.hasTargetRes {
		args = append(args, "-tr",
			formatFloat(cfg.targetResX),
			formatFloat(cfg.targetResY))
	}
	if cfg.hasSize {
		args = append(args, "-ts",
			strconv.Itoa(cfg.sizeX),
			strconv.Itoa(cfg.sizeY))
	}
	if cfg.extent != nil {
		args = append(args, "-te",
			formatFloat(cfg.extent.MinX),
			formatFloat(cfg.extent.MinY),
			formatFloat(cfg.extent.MaxX),
			formatFloat(cfg.extent.MaxY))
	}
	for _, co := range cfg.creationOptions {
		args = append(args, "-co", co)
	}
	args = append(args, cfg.rawOptions...)
	return args
}
