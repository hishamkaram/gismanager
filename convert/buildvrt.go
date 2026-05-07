package convert

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/v2/errs"
	"github.com/hishamkaram/gismanager/v2/internal/slogx"
)

// VRTOption configures a [BuildVRT] call. Construct via the
// [WithVRT*] helpers below; pass any number of options in any order.
type VRTOption func(*vrtConfig)

type vrtConfig struct {
	logger        *slog.Logger
	resolution    string  // -resolution highest|lowest|average|user
	resX          float64 // -tr xRes yRes (only when resolution=user)
	resY          float64
	hasRes        bool
	separate      bool   // -separate (each input becomes a band)
	addAlpha      bool   // -addalpha
	resampleAlg   string // -r
	srcNoData     string // -srcnodata "VAL[,VAL2,...]"
	vrtNoData     string // -vrtnodata
	hideNoData    bool   // -hidenodata
	bandList      []int  // -b
	allowProjMisc bool   // -allow_projection_difference
	rawOptions    []string
}

func newVRTConfig(opts []VRTOption) *vrtConfig {
	c := &vrtConfig{logger: slogx.Default()}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// WithVRTLogger sets the structured logger used during VRT build.
// nil falls back to [slogx.Default].
func WithVRTLogger(l *slog.Logger) VRTOption {
	return func(c *vrtConfig) {
		if l == nil {
			c.logger = slogx.Default()
			return
		}
		c.logger = l
	}
}

// WithVRTResolution selects how the VRT picks output resolution from
// inputs of differing pixel sizes. Valid values: "highest", "lowest",
// "average", "user". When "user", call [WithVRTUserResolution] too.
// Maps to `-resolution`. Default GDAL behavior is "average".
func WithVRTResolution(mode string) VRTOption {
	return func(c *vrtConfig) { c.resolution = mode }
}

// WithVRTUserResolution sets the explicit output pixel size. Only
// effective when [WithVRTResolution] is "user". Maps to `-tr`.
func WithVRTUserResolution(xRes, yRes float64) VRTOption {
	return func(c *vrtConfig) {
		c.resX = xRes
		c.resY = yRes
		c.hasRes = true
	}
}

// WithVRTSeparate emits one VRT band per input dataset (instead of
// stacking them into a tile mosaic). Useful for assembling RGBA from
// single-band inputs. Maps to `-separate`.
func WithVRTSeparate() VRTOption {
	return func(c *vrtConfig) { c.separate = true }
}

// WithVRTAddAlpha appends an alpha band to the output indicating which
// pixels are sourced from any input vs. NoData. Maps to `-addalpha`.
func WithVRTAddAlpha() VRTOption {
	return func(c *vrtConfig) { c.addAlpha = true }
}

// WithVRTResamplingAlg sets the resampling algorithm used when input
// resolutions differ from the output. "near", "bilinear", "cubic", etc.
// Maps to `-r`.
func WithVRTResamplingAlg(alg string) VRTOption {
	return func(c *vrtConfig) { c.resampleAlg = alg }
}

// WithVRTSrcNoData sets the NoData value(s) interpreted on the source
// side, comma-separated for multi-band inputs. Maps to `-srcnodata`.
func WithVRTSrcNoData(values string) VRTOption {
	return func(c *vrtConfig) { c.srcNoData = values }
}

// WithVRTNoData sets the NoData value(s) emitted on the output VRT.
// Maps to `-vrtnodata`.
func WithVRTNoData(values string) VRTOption {
	return func(c *vrtConfig) { c.vrtNoData = values }
}

// WithVRTHideNoData omits the NoData value from the output VRT
// metadata so downstream readers treat those pixels as valid. Maps
// to `-hidenodata`.
func WithVRTHideNoData() VRTOption {
	return func(c *vrtConfig) { c.hideNoData = true }
}

// WithVRTBands selects a subset of bands from each input. Maps to
// repeated `-b N`.
func WithVRTBands(bands ...int) VRTOption {
	return func(c *vrtConfig) { c.bandList = append([]int(nil), bands...) }
}

// WithVRTAllowProjectionDifference relaxes the SRS-must-match check;
// useful when assembling near-equivalent SRSes (different EPSG codes
// resolving to identical projections). Maps to
// `-allow_projection_difference`.
func WithVRTAllowProjectionDifference() VRTOption {
	return func(c *vrtConfig) { c.allowProjMisc = true }
}

// WithVRTRawOptions appends raw `gdalbuildvrt`-style flag tokens to the
// generated argument list.
func WithVRTRawOptions(args ...string) VRTOption {
	return func(c *vrtConfig) { c.rawOptions = append(c.rawOptions, args...) }
}

// BuildVRT composes one or more raster inputs into a Virtual Raster
// (VRT) at dst. Thin wrapper around the GDAL C entry point behind
// `gdalbuildvrt` ([gdal.BuildVRT]).
//
// Use cases: assemble a tile pyramid from many individual GeoTIFFs;
// stack single-band inputs into RGBA via [WithVRTSeparate]; build a
// multi-resolution mosaic for downstream `Warp` / `Translate` calls.
//
// At least one source path is required. Each path is opened with
// `gdal.OpenEx(OFRaster|OFReadOnly)` and closed when [BuildVRT]
// returns.
//
// Errors are wrapped with [errs.ErrConvertFailed]; recover via
// [errors.As] into [*GISError]. The Op field is "BuildVRT".
func BuildVRT(ctx context.Context, dst string, srcs []string, opts ...VRTOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(srcs) == 0 {
		return errs.NewGISError("BuildVRT", dst, errs.ErrConvertFailed,
			fmt.Errorf("at least one source path is required"))
	}

	cfg := newVRTConfig(opts)

	// Pre-validate every source by opening it. GDALBuildVRT in path-mode
	// silently skips unreadable paths with a stderr warning rather than
	// returning a non-zero cerr — so without this loop, callers passing
	// a typo-d path would get a non-nil result with a malformed VRT
	// instead of a useful error. We close immediately; GDAL re-opens via
	// its path-mode API. Acceptable cost for the much better error
	// envelope.
	for _, src := range srcs {
		ds, err := gdal.OpenEx(src, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
		if err != nil {
			cfg.logger.Error("BuildVRT: open source", "src", src, "err", err)
			return errs.NewGISError("BuildVRT", src, errs.ErrConvertFailed, err)
		}
		ds.Close()
	}

	// lukeroth/gdal's GDALBuildVRT wrapper has a quirk: it always
	// indexes &sourceDS[0] (panics on empty slice) AND passes both the
	// dataset list and the file-path list to the C function. The C API
	// uses paths when papszSrcDSNames is non-NULL, ignoring the dataset
	// list — so we let GDAL do the opens itself by passing srcs as the
	// file-path arg, with a zero-value dataset slice of matching length
	// to satisfy the binding's index guard.
	zeroDS := make([]gdal.Dataset, len(srcs))

	args := buildVRTArgs(cfg)
	cfg.logger.Debug("BuildVRT: invoking",
		"dst", dst, "srcs", srcs, "args", args)

	out, vErr := gdal.BuildVRT(dst, zeroDS, srcs, args)
	if vErr != nil {
		return errs.NewGISError("BuildVRT", dst, errs.ErrConvertFailed, vErr)
	}
	defer out.Close()
	return nil
}

// buildVRTArgs renders cfg into the []string gdalbuildvrt-style
// argument list. Unit-tested separately so the option->arg mapping is
// locked in without CGo.
func buildVRTArgs(cfg *vrtConfig) []string {
	var args []string
	if cfg.resolution != "" {
		args = append(args, "-resolution", cfg.resolution)
	}
	if cfg.hasRes {
		args = append(args, "-tr",
			formatFloat(cfg.resX),
			formatFloat(cfg.resY))
	}
	if cfg.separate {
		args = append(args, "-separate")
	}
	if cfg.addAlpha {
		args = append(args, "-addalpha")
	}
	if cfg.resampleAlg != "" {
		args = append(args, "-r", cfg.resampleAlg)
	}
	if cfg.srcNoData != "" {
		args = append(args, "-srcnodata", cfg.srcNoData)
	}
	if cfg.vrtNoData != "" {
		args = append(args, "-vrtnodata", cfg.vrtNoData)
	}
	if cfg.hideNoData {
		args = append(args, "-hidenodata")
	}
	for _, b := range cfg.bandList {
		args = append(args, "-b", fmt.Sprintf("%d", b))
	}
	if cfg.allowProjMisc {
		args = append(args, "-allow_projection_difference")
	}
	args = append(args, cfg.rawOptions...)
	return args
}
