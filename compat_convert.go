package gismanager

// Compatibility shim for the v1.x conversion subsystem API.
//
// The conversion subsystem (ConvertVector / ConvertRaster / ToCOG /
// ReprojectRaster / Rasterize / BuildVRT / DEMProcessing / ToPMTiles
// plus all their With* options) moved to the [convert] subpackage as
// part of the v2 restructure groundwork (see Phase 2 in
// ~/.claude/plans/how-can-we-improve-steady-emerson.md).
//
// This file re-exports every public symbol from the convert subpackage
// at its v1.x import path so existing v1.x callers compile unchanged.
// Type aliases preserve full identity — errors.As, errors.Is, pointer
// equality, slice element types all interoperate transparently across
// the boundary. Function wrappers are pure pass-throughs with no
// behavior change.
//
// All declarations are marked Deprecated; new code should import
// "github.com/hishamkaram/gismanager/convert" directly. The v2 module
// bump (Phase 4) deletes this file.

import (
	"context"
	"log/slog"

	"github.com/hishamkaram/gismanager/convert"
)

// =============================================================================
// Type aliases
// =============================================================================

// VectorConvertOption is preserved as a v1.x alias of [convert.VectorOption].
//
// Deprecated: use [convert.VectorOption] directly via
// `import "github.com/hishamkaram/gismanager/convert"`.
type VectorConvertOption = convert.VectorOption

// RasterConvertOption is preserved as a v1.x alias of [convert.RasterOption].
//
// Deprecated: use [convert.RasterOption] directly.
type RasterConvertOption = convert.RasterOption

// RasterizeOption is preserved as a v1.x alias of [convert.RasterizeOption].
//
// Deprecated: use [convert.RasterizeOption] directly.
type RasterizeOption = convert.RasterizeOption

// VRTOption is preserved as a v1.x alias of [convert.VRTOption].
//
// Deprecated: use [convert.VRTOption] directly.
type VRTOption = convert.VRTOption

// DEMOption is preserved as a v1.x alias of [convert.DEMOption].
//
// Deprecated: use [convert.DEMOption] directly.
type DEMOption = convert.DEMOption

// PMTilesOption is preserved as a v1.x alias of [convert.PMTilesOption].
//
// Deprecated: use [convert.PMTilesOption] directly.
type PMTilesOption = convert.PMTilesOption

// =============================================================================
// Entry points
// =============================================================================

// ConvertVector forwards to [convert.ConvertVector].
//
// Deprecated: use [convert.ConvertVector] directly.
func ConvertVector(ctx context.Context, src, dst string, opts ...VectorConvertOption) error {
	return convert.ConvertVector(ctx, src, dst, opts...)
}

// ConvertRaster forwards to [convert.ConvertRaster].
//
// Deprecated: use [convert.ConvertRaster] directly.
func ConvertRaster(ctx context.Context, src, dst string, opts ...RasterConvertOption) error {
	return convert.ConvertRaster(ctx, src, dst, opts...)
}

// ToCOG forwards to [convert.ToCOG].
//
// Deprecated: use [convert.ToCOG] directly.
func ToCOG(ctx context.Context, src, dst string, opts ...RasterConvertOption) error {
	return convert.ToCOG(ctx, src, dst, opts...)
}

// ReprojectRaster forwards to [convert.ReprojectRaster].
//
// Deprecated: use [convert.ReprojectRaster] directly.
func ReprojectRaster(ctx context.Context, src, dst, srcSRS, dstSRS string, opts ...RasterConvertOption) error {
	return convert.ReprojectRaster(ctx, src, dst, srcSRS, dstSRS, opts...)
}

// Rasterize forwards to [convert.Rasterize].
//
// Deprecated: use [convert.Rasterize] directly.
func Rasterize(ctx context.Context, vectorSrc, rasterDst string, opts ...RasterizeOption) error {
	return convert.Rasterize(ctx, vectorSrc, rasterDst, opts...)
}

// BuildVRT forwards to [convert.BuildVRT].
//
// Deprecated: use [convert.BuildVRT] directly.
func BuildVRT(ctx context.Context, dst string, srcs []string, opts ...VRTOption) error {
	return convert.BuildVRT(ctx, dst, srcs, opts...)
}

// DEMProcessing forwards to [convert.DEMProcessing].
//
// Deprecated: use [convert.DEMProcessing] directly.
func DEMProcessing(ctx context.Context, src, dst, mode string, opts ...DEMOption) error {
	return convert.DEMProcessing(ctx, src, dst, mode, opts...)
}

// ToPMTiles forwards to [convert.ToPMTiles].
//
// Deprecated: use [convert.ToPMTiles] directly.
func ToPMTiles(ctx context.Context, src, dst string, opts ...PMTilesOption) error {
	return convert.ToPMTiles(ctx, src, dst, opts...)
}

// =============================================================================
// Vector options — all forward to convert.WithVector*; all Deprecated.
// =============================================================================

// WithVectorLogger forwards to [convert.WithVectorLogger]. Deprecated.
func WithVectorLogger(l *slog.Logger) VectorConvertOption { return convert.WithVectorLogger(l) }

// WithVectorFormat forwards to [convert.WithVectorFormat]. Deprecated.
func WithVectorFormat(driverName string) VectorConvertOption {
	return convert.WithVectorFormat(driverName)
}

// WithVectorSourceSRS forwards to [convert.WithVectorSourceSRS]. Deprecated.
func WithVectorSourceSRS(srs string) VectorConvertOption { return convert.WithVectorSourceSRS(srs) }

// WithVectorTargetSRS forwards to [convert.WithVectorTargetSRS]. Deprecated.
func WithVectorTargetSRS(srs string) VectorConvertOption { return convert.WithVectorTargetSRS(srs) }

// WithVectorBoundingBox forwards to [convert.WithVectorBoundingBox]. Deprecated.
func WithVectorBoundingBox(minX, minY, maxX, maxY float64) VectorConvertOption {
	return convert.WithVectorBoundingBox(minX, minY, maxX, maxY)
}

// WithVectorWhere forwards to [convert.WithVectorWhere]. Deprecated.
func WithVectorWhere(sql string) VectorConvertOption { return convert.WithVectorWhere(sql) }

// WithVectorSimplify forwards to [convert.WithVectorSimplify]. Deprecated.
func WithVectorSimplify(tolerance float64) VectorConvertOption {
	return convert.WithVectorSimplify(tolerance)
}

// WithVectorSelectFields forwards to [convert.WithVectorSelectFields]. Deprecated.
func WithVectorSelectFields(fields ...string) VectorConvertOption {
	return convert.WithVectorSelectFields(fields...)
}

// WithVectorLayerName forwards to [convert.WithVectorLayerName]. Deprecated.
func WithVectorLayerName(name string) VectorConvertOption { return convert.WithVectorLayerName(name) }

// WithVectorOverwrite forwards to [convert.WithVectorOverwrite]. Deprecated.
func WithVectorOverwrite() VectorConvertOption { return convert.WithVectorOverwrite() }

// WithVectorRawOptions forwards to [convert.WithVectorRawOptions]. Deprecated.
func WithVectorRawOptions(args ...string) VectorConvertOption {
	return convert.WithVectorRawOptions(args...)
}

// =============================================================================
// Raster options — all forward to convert.WithRaster*; all Deprecated.
// =============================================================================

// WithRasterLogger forwards to [convert.WithRasterLogger]. Deprecated.
func WithRasterLogger(l *slog.Logger) RasterConvertOption { return convert.WithRasterLogger(l) }

// WithRasterFormat forwards to [convert.WithRasterFormat]. Deprecated.
func WithRasterFormat(driver string) RasterConvertOption { return convert.WithRasterFormat(driver) }

// WithRasterCreationOption forwards to [convert.WithRasterCreationOption]. Deprecated.
func WithRasterCreationOption(key, val string) RasterConvertOption {
	return convert.WithRasterCreationOption(key, val)
}

// WithRasterOutputBounds forwards to [convert.WithRasterOutputBounds]. Deprecated.
func WithRasterOutputBounds(minX, minY, maxX, maxY float64) RasterConvertOption {
	return convert.WithRasterOutputBounds(minX, minY, maxX, maxY)
}

// WithRasterBands forwards to [convert.WithRasterBands]. Deprecated.
func WithRasterBands(bands ...int) RasterConvertOption { return convert.WithRasterBands(bands...) }

// WithRasterResamplingAlg forwards to [convert.WithRasterResamplingAlg]. Deprecated.
func WithRasterResamplingAlg(alg string) RasterConvertOption {
	return convert.WithRasterResamplingAlg(alg)
}

// WithRasterTargetResolution forwards to [convert.WithRasterTargetResolution]. Deprecated.
func WithRasterTargetResolution(xRes, yRes float64) RasterConvertOption {
	return convert.WithRasterTargetResolution(xRes, yRes)
}

// WithRasterCutline forwards to [convert.WithRasterCutline]. Deprecated.
func WithRasterCutline(ds, layer string) RasterConvertOption {
	return convert.WithRasterCutline(ds, layer)
}

// WithRasterRawOptions forwards to [convert.WithRasterRawOptions]. Deprecated.
func WithRasterRawOptions(args ...string) RasterConvertOption {
	return convert.WithRasterRawOptions(args...)
}

// =============================================================================
// Rasterize options — all forward to convert.WithRasterize*; all Deprecated.
// =============================================================================

// WithRasterizeLogger forwards to [convert.WithRasterizeLogger]. Deprecated.
func WithRasterizeLogger(l *slog.Logger) RasterizeOption { return convert.WithRasterizeLogger(l) }

// WithRasterizeFormat forwards to [convert.WithRasterizeFormat]. Deprecated.
func WithRasterizeFormat(driver string) RasterizeOption { return convert.WithRasterizeFormat(driver) }

// WithRasterizeBurnValues forwards to [convert.WithRasterizeBurnValues]. Deprecated.
func WithRasterizeBurnValues(values ...float64) RasterizeOption {
	return convert.WithRasterizeBurnValues(values...)
}

// WithRasterizeAttribute forwards to [convert.WithRasterizeAttribute]. Deprecated.
func WithRasterizeAttribute(name string) RasterizeOption { return convert.WithRasterizeAttribute(name) }

// WithRasterizeTargetResolution forwards to [convert.WithRasterizeTargetResolution]. Deprecated.
func WithRasterizeTargetResolution(xRes, yRes float64) RasterizeOption {
	return convert.WithRasterizeTargetResolution(xRes, yRes)
}

// WithRasterizeOutputSize forwards to [convert.WithRasterizeOutputSize]. Deprecated.
func WithRasterizeOutputSize(xSize, ySize int) RasterizeOption {
	return convert.WithRasterizeOutputSize(xSize, ySize)
}

// WithRasterizeOutputBounds forwards to [convert.WithRasterizeOutputBounds]. Deprecated.
func WithRasterizeOutputBounds(minX, minY, maxX, maxY float64) RasterizeOption {
	return convert.WithRasterizeOutputBounds(minX, minY, maxX, maxY)
}

// WithRasterizeLayer forwards to [convert.WithRasterizeLayer]. Deprecated.
func WithRasterizeLayer(name string) RasterizeOption { return convert.WithRasterizeLayer(name) }

// WithRasterizeWhere forwards to [convert.WithRasterizeWhere]. Deprecated.
func WithRasterizeWhere(sql string) RasterizeOption { return convert.WithRasterizeWhere(sql) }

// WithRasterizeCreationOption forwards to [convert.WithRasterizeCreationOption]. Deprecated.
func WithRasterizeCreationOption(key, val string) RasterizeOption {
	return convert.WithRasterizeCreationOption(key, val)
}

// WithRasterizeOutputType forwards to [convert.WithRasterizeOutputType]. Deprecated.
func WithRasterizeOutputType(t string) RasterizeOption { return convert.WithRasterizeOutputType(t) }

// WithRasterizeRawOptions forwards to [convert.WithRasterizeRawOptions]. Deprecated.
func WithRasterizeRawOptions(args ...string) RasterizeOption {
	return convert.WithRasterizeRawOptions(args...)
}

// =============================================================================
// VRT options — all forward to convert.WithVRT*; all Deprecated.
// =============================================================================

// WithVRTLogger forwards to [convert.WithVRTLogger]. Deprecated.
func WithVRTLogger(l *slog.Logger) VRTOption { return convert.WithVRTLogger(l) }

// WithVRTResolution forwards to [convert.WithVRTResolution]. Deprecated.
func WithVRTResolution(mode string) VRTOption { return convert.WithVRTResolution(mode) }

// WithVRTUserResolution forwards to [convert.WithVRTUserResolution]. Deprecated.
func WithVRTUserResolution(xRes, yRes float64) VRTOption {
	return convert.WithVRTUserResolution(xRes, yRes)
}

// WithVRTSeparate forwards to [convert.WithVRTSeparate]. Deprecated.
func WithVRTSeparate() VRTOption { return convert.WithVRTSeparate() }

// WithVRTAddAlpha forwards to [convert.WithVRTAddAlpha]. Deprecated.
func WithVRTAddAlpha() VRTOption { return convert.WithVRTAddAlpha() }

// WithVRTResamplingAlg forwards to [convert.WithVRTResamplingAlg]. Deprecated.
func WithVRTResamplingAlg(alg string) VRTOption { return convert.WithVRTResamplingAlg(alg) }

// WithVRTSrcNoData forwards to [convert.WithVRTSrcNoData]. Deprecated.
func WithVRTSrcNoData(values string) VRTOption { return convert.WithVRTSrcNoData(values) }

// WithVRTNoData forwards to [convert.WithVRTNoData]. Deprecated.
func WithVRTNoData(values string) VRTOption { return convert.WithVRTNoData(values) }

// WithVRTHideNoData forwards to [convert.WithVRTHideNoData]. Deprecated.
func WithVRTHideNoData() VRTOption { return convert.WithVRTHideNoData() }

// WithVRTBands forwards to [convert.WithVRTBands]. Deprecated.
func WithVRTBands(bands ...int) VRTOption { return convert.WithVRTBands(bands...) }

// WithVRTAllowProjectionDifference forwards to [convert.WithVRTAllowProjectionDifference]. Deprecated.
func WithVRTAllowProjectionDifference() VRTOption { return convert.WithVRTAllowProjectionDifference() }

// WithVRTRawOptions forwards to [convert.WithVRTRawOptions]. Deprecated.
func WithVRTRawOptions(args ...string) VRTOption { return convert.WithVRTRawOptions(args...) }

// =============================================================================
// DEM options — all forward to convert.WithDEM*; all Deprecated.
// =============================================================================

// WithDEMLogger forwards to [convert.WithDEMLogger]. Deprecated.
func WithDEMLogger(l *slog.Logger) DEMOption { return convert.WithDEMLogger(l) }

// WithDEMFormat forwards to [convert.WithDEMFormat]. Deprecated.
func WithDEMFormat(driver string) DEMOption { return convert.WithDEMFormat(driver) }

// WithDEMColorFile forwards to [convert.WithDEMColorFile]. Deprecated.
func WithDEMColorFile(path string) DEMOption { return convert.WithDEMColorFile(path) }

// WithDEMZFactor forwards to [convert.WithDEMZFactor]. Deprecated.
func WithDEMZFactor(z float64) DEMOption { return convert.WithDEMZFactor(z) }

// WithDEMScale forwards to [convert.WithDEMScale]. Deprecated.
func WithDEMScale(s float64) DEMOption { return convert.WithDEMScale(s) }

// WithDEMAzimuth forwards to [convert.WithDEMAzimuth]. Deprecated.
func WithDEMAzimuth(degrees float64) DEMOption { return convert.WithDEMAzimuth(degrees) }

// WithDEMAltitude forwards to [convert.WithDEMAltitude]. Deprecated.
func WithDEMAltitude(degrees float64) DEMOption { return convert.WithDEMAltitude(degrees) }

// WithDEMCombined forwards to [convert.WithDEMCombined]. Deprecated.
func WithDEMCombined() DEMOption { return convert.WithDEMCombined() }

// WithDEMMultidirectional forwards to [convert.WithDEMMultidirectional]. Deprecated.
func WithDEMMultidirectional() DEMOption { return convert.WithDEMMultidirectional() }

// WithDEMAlgorithm forwards to [convert.WithDEMAlgorithm]. Deprecated.
func WithDEMAlgorithm(alg string) DEMOption { return convert.WithDEMAlgorithm(alg) }

// WithDEMCreationOption forwards to [convert.WithDEMCreationOption]. Deprecated.
func WithDEMCreationOption(key, val string) DEMOption {
	return convert.WithDEMCreationOption(key, val)
}

// WithDEMOutputType forwards to [convert.WithDEMOutputType]. Deprecated.
func WithDEMOutputType(t string) DEMOption { return convert.WithDEMOutputType(t) }

// WithDEMRawOptions forwards to [convert.WithDEMRawOptions]. Deprecated.
func WithDEMRawOptions(args ...string) DEMOption { return convert.WithDEMRawOptions(args...) }

// =============================================================================
// PMTiles options — all forward to convert.WithPMTiles*; all Deprecated.
// =============================================================================

// WithPMTilesLogger forwards to [convert.WithPMTilesLogger]. Deprecated.
func WithPMTilesLogger(l *slog.Logger) PMTilesOption { return convert.WithPMTilesLogger(l) }

// WithPMTilesDeduplicate forwards to [convert.WithPMTilesDeduplicate]. Deprecated.
func WithPMTilesDeduplicate(dedupe bool) PMTilesOption { return convert.WithPMTilesDeduplicate(dedupe) }
