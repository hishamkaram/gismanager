package gismanager

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/lukeroth/gdal"
)

// RasterConvertOption configures [ConvertRaster], [ToCOG], and
// [ReprojectRaster]. Construct via the [WithRaster*] helpers.
type RasterConvertOption func(*rasterConvertConfig)

type rasterConvertConfig struct {
	logger          *slog.Logger
	format          string
	creationOptions []string // raw "KEY=VAL" pairs for -co
	outputBounds    *rasterBounds
	bands           []int
	resamplingAlg   string
	targetResX      float64
	targetResY      float64
	hasTargetRes    bool
	cutlineDS       string
	cutlineLayer    string
	rawOptions      []string
}

type rasterBounds struct {
	MinX, MinY, MaxX, MaxY float64
}

func newRasterConvertConfig(opts []RasterConvertOption) *rasterConvertConfig {
	c := &rasterConvertConfig{logger: GetLogger()}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// WithRasterLogger sets the structured logger used during conversion.
// nil falls back to [GetLogger].
func WithRasterLogger(l *slog.Logger) RasterConvertOption {
	return func(c *rasterConvertConfig) {
		if l == nil {
			c.logger = GetLogger()
			return
		}
		c.logger = l
	}
}

// WithRasterFormat selects the GDAL output driver by name (e.g.
// "GTiff", "COG", "PNG", "JPEG"). Maps to gdal_translate's `-of` and
// gdalwarp's `-of`. When empty, GDAL infers from the destination path.
func WithRasterFormat(driver string) RasterConvertOption {
	return func(c *rasterConvertConfig) { c.format = driver }
}

// WithRasterCreationOption appends a single driver-specific creation
// option (e.g. "COMPRESS=DEFLATE", "TILING_SCHEME=GoogleMapsCompatible",
// "BLOCKSIZE=512"). May be called multiple times to set several options.
// Maps to `-co KEY=VAL`.
func WithRasterCreationOption(key, val string) RasterConvertOption {
	return func(c *rasterConvertConfig) {
		c.creationOptions = append(c.creationOptions, fmt.Sprintf("%s=%s", key, val))
	}
}

// WithRasterOutputBounds restricts the output to the given bounding box,
// in the source CRS units (for [ConvertRaster]'s `-projwin`) or target
// CRS units (for [ReprojectRaster]'s `-te`). The wrapper picks the right
// flag based on which entry point invoked it.
func WithRasterOutputBounds(minX, minY, maxX, maxY float64) RasterConvertOption {
	return func(c *rasterConvertConfig) {
		c.outputBounds = &rasterBounds{MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY}
	}
}

// WithRasterBands selects a subset of source bands to copy, in the
// given order. Maps to `-b 1 -b 2 ...`. Empty means all bands.
func WithRasterBands(bands ...int) RasterConvertOption {
	return func(c *rasterConvertConfig) { c.bands = append([]int(nil), bands...) }
}

// WithRasterResamplingAlg sets the resampling algorithm for warp /
// translate. Common values: "near", "bilinear", "cubic", "cubicspline",
// "lanczos", "average", "mode", "max", "min". Maps to `-r`.
func WithRasterResamplingAlg(alg string) RasterConvertOption {
	return func(c *rasterConvertConfig) { c.resamplingAlg = alg }
}

// WithRasterTargetResolution sets the output pixel size in target-CRS
// units. Pass non-zero values for both axes; zero means "use source
// resolution". Maps to `-tr xRes yRes`.
func WithRasterTargetResolution(xRes, yRes float64) RasterConvertOption {
	return func(c *rasterConvertConfig) {
		c.targetResX = xRes
		c.targetResY = yRes
		c.hasTargetRes = true
	}
}

// WithRasterCutline points at a vector dataset (and an optional layer
// within it) to mask the output by. Pixels outside the cutline polygon
// are written as NoData. Maps to `-cutline <ds>` (and `-cl <layer>` if
// non-empty). Useful for cookie-cutter clipping of imagery.
func WithRasterCutline(ds, layer string) RasterConvertOption {
	return func(c *rasterConvertConfig) {
		c.cutlineDS = ds
		c.cutlineLayer = layer
	}
}

// WithRasterRawOptions appends raw flag tokens to the generated
// argument list. Use this to reach gdal_translate / gdalwarp flags the
// typed helpers don't expose.
func WithRasterRawOptions(args ...string) RasterConvertOption {
	return func(c *rasterConvertConfig) { c.rawOptions = append(c.rawOptions, args...) }
}

// ConvertRaster converts the raster source at src into dst, applying any
// of the provided options. It is a thin wrapper around the C entry
// point behind `gdal_translate` ([gdal.Translate]) — every option maps
// 1:1 to a gdal_translate CLI flag.
//
// Use cases:
//   - GeoTIFF → COG (prefer [ToCOG] for sane defaults).
//   - GeoTIFF → PNG / JPEG for web previews.
//   - Subset bands or extract a window.
//
// For reprojection use [ReprojectRaster] instead — gdal_translate does
// NOT reproject.
//
// Errors are wrapped with [ErrConvertFailed]; recover the underlying
// GDAL error via [errors.As] into [*GISError].
func ConvertRaster(ctx context.Context, src, dst string, opts ...RasterConvertOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg := newRasterConvertConfig(opts)

	srcDS, err := gdal.OpenEx(src, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		cfg.logger.Error("ConvertRaster: open source", "src", src, "err", err)
		return newGISError("ConvertRaster", src, ErrConvertFailed, err)
	}
	defer srcDS.Close()

	args := buildTranslateArgs(cfg)
	cfg.logger.Debug("ConvertRaster: invoking Translate",
		"src", src, "dst", dst, "args", args)

	out, tErr := gdal.Translate(dst, srcDS, args)
	if tErr != nil {
		return newGISError("ConvertRaster", fmt.Sprintf("%s -> %s", src, dst),
			ErrConvertFailed, tErr)
	}
	out.Close()
	return nil
}

// ToCOG converts the raster source at src into a Cloud-Optimized GeoTIFF
// at dst. Equivalent to:
//
//	ConvertRaster(ctx, src, dst,
//	    WithRasterFormat("COG"),
//	    WithRasterCreationOption("COMPRESS", "DEFLATE"),
//	    WithRasterCreationOption("BLOCKSIZE", "512"),
//	    WithRasterCreationOption("OVERVIEW_RESAMPLING", "NEAREST"),
//	    opts...)
//
// Caller-supplied opts override any of the defaults — pass
// `WithRasterCreationOption("COMPRESS", "ZSTD")` to swap codecs, etc.
func ToCOG(ctx context.Context, src, dst string, opts ...RasterConvertOption) error {
	defaults := []RasterConvertOption{
		WithRasterFormat("COG"),
		WithRasterCreationOption("COMPRESS", "DEFLATE"),
		WithRasterCreationOption("BLOCKSIZE", "512"),
		WithRasterCreationOption("OVERVIEW_RESAMPLING", "NEAREST"),
	}
	merged := make([]RasterConvertOption, 0, len(defaults)+len(opts))
	merged = append(merged, defaults...)
	merged = append(merged, opts...)
	return ConvertRaster(ctx, src, dst, merged...)
}

// buildTranslateArgs renders cfg into gdal_translate-style args.
// Unit-tested separately so the mapping is locked in without CGo.
func buildTranslateArgs(cfg *rasterConvertConfig) []string {
	var args []string
	if cfg.format != "" {
		args = append(args, "-of", cfg.format)
	}
	for _, co := range cfg.creationOptions {
		args = append(args, "-co", co)
	}
	for _, b := range cfg.bands {
		args = append(args, "-b", strconv.Itoa(b))
	}
	if cfg.resamplingAlg != "" {
		args = append(args, "-r", cfg.resamplingAlg)
	}
	if cfg.outputBounds != nil {
		// gdal_translate uses -projwin (ulx uly lrx lry — note y-flip).
		args = append(args, "-projwin",
			formatFloat(cfg.outputBounds.MinX),
			formatFloat(cfg.outputBounds.MaxY),
			formatFloat(cfg.outputBounds.MaxX),
			formatFloat(cfg.outputBounds.MinY))
	}
	if cfg.hasTargetRes {
		args = append(args, "-tr",
			formatFloat(cfg.targetResX),
			formatFloat(cfg.targetResY))
	}
	args = append(args, cfg.rawOptions...)
	return args
}
