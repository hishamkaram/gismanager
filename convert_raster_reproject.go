package gismanager

import (
	"context"
	"fmt"

	"github.com/lukeroth/gdal"
)

// ReprojectRaster reprojects the raster source at src into dst, going
// from srcSRS to dstSRS. It is a thin wrapper around the C entry point
// behind `gdalwarp` ([gdal.Warp]).
//
// srcSRS and dstSRS accept any GDAL SetFromUserInput-friendly form
// ("EPSG:4326", "EPSG:3857", "+proj=longlat +datum=WGS84 +no_defs",
// well-known WKT). The most common pair in 2025–26 web pipelines is
// EPSG:32618 / EPSG:4326 → EPSG:3857.
//
// Supports the same [WithRaster*] options as [ConvertRaster] /
// [ToCOG] — chain with [WithRasterResamplingAlg],
// [WithRasterTargetResolution], [WithRasterCutline] for the typical
// "warp + clip" pipeline.
//
// Output bounds, when supplied via [WithRasterOutputBounds], are
// interpreted in the *target* CRS (gdalwarp's `-te` semantics), not
// the source CRS (gdal_translate's `-projwin`).
//
// Errors are wrapped with [ErrConvertFailed]; recover via
// [errors.As] into [*GISError]. The Op field is "ReprojectRaster".
//
// Known gap: same as [ConvertVector] — silent failures when GDAL's C
// option-parsing rejects an input (e.g. an unknown CRS string) without
// setting cerr. Pre-validate inputs on the caller side if needed.
func ReprojectRaster(ctx context.Context, src, dst, srcSRS, dstSRS string, opts ...RasterConvertOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if srcSRS == "" || dstSRS == "" {
		return newGISError("ReprojectRaster", fmt.Sprintf("%s -> %s", src, dst),
			ErrConvertFailed,
			fmt.Errorf("srcSRS and dstSRS must both be non-empty"))
	}

	cfg := newRasterConvertConfig(opts)

	srcDS, err := gdal.OpenEx(src, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		cfg.logger.Error("ReprojectRaster: open source", "src", src, "err", err)
		return newGISError("ReprojectRaster", src, ErrConvertFailed, err)
	}
	defer srcDS.Close()

	args := buildWarpArgs(cfg, srcSRS, dstSRS)
	cfg.logger.Debug("ReprojectRaster: invoking Warp",
		"src", src, "dst", dst, "args", args)

	out, wErr := gdal.Warp(dst, nil, []gdal.Dataset{srcDS}, args)
	if wErr != nil {
		return newGISError("ReprojectRaster", fmt.Sprintf("%s -> %s", src, dst),
			ErrConvertFailed, wErr)
	}
	out.Close()
	return nil
}

// buildWarpArgs renders cfg + (srcSRS, dstSRS) into gdalwarp-style args.
// Differs from [buildTranslateArgs] in two ways:
//   - Uses `-s_srs` / `-t_srs` for CRS pair (mandatory for warp).
//   - Uses `-te` for output bounds (target CRS) rather than `-projwin`
//     (source CRS) — this is the gdalwarp convention.
//   - Adds `-cutline` / `-cl` when the cutline option is set (cutline is
//     a warp-only flag; gdal_translate does not support it).
//
// Unit-tested separately so the mapping is locked in without CGo.
func buildWarpArgs(cfg *rasterConvertConfig, srcSRS, dstSRS string) []string {
	var args []string
	if cfg.format != "" {
		args = append(args, "-of", cfg.format)
	}
	args = append(args, "-s_srs", srcSRS, "-t_srs", dstSRS)
	for _, co := range cfg.creationOptions {
		args = append(args, "-co", co)
	}
	if cfg.resamplingAlg != "" {
		args = append(args, "-r", cfg.resamplingAlg)
	}
	if cfg.outputBounds != nil {
		// gdalwarp -te uses (xmin ymin xmax ymax) in target CRS.
		args = append(args, "-te",
			formatFloat(cfg.outputBounds.MinX),
			formatFloat(cfg.outputBounds.MinY),
			formatFloat(cfg.outputBounds.MaxX),
			formatFloat(cfg.outputBounds.MaxY))
	}
	if cfg.hasTargetRes {
		args = append(args, "-tr",
			formatFloat(cfg.targetResX),
			formatFloat(cfg.targetResY))
	}
	if cfg.cutlineDS != "" {
		args = append(args, "-cutline", cfg.cutlineDS)
		if cfg.cutlineLayer != "" {
			args = append(args, "-cl", cfg.cutlineLayer)
		}
		// -crop_to_cutline is the typical companion (the alternative is
		// to keep the original raster extent and only mask pixels). We
		// default to crop because the cutline use case is almost always
		// cookie-cutter clipping for tile pipelines. Users who want
		// preserve-extent semantics can pass -overwrite via raw options.
		args = append(args, "-crop_to_cutline")
	}
	args = append(args, cfg.rawOptions...)
	return args
}
