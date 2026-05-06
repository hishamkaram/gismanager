package convert

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/internal/errs"
	"github.com/hishamkaram/gismanager/internal/slogx"
)

// DEMOption configures a [DEMProcessing] call.
type DEMOption func(*demConfig)

type demConfig struct {
	logger          *slog.Logger
	format          string
	colorFile       string
	zFactor         float64
	hasZFactor      bool
	scale           float64
	hasScale        bool
	azimuth         float64
	hasAzimuth      bool
	altitude        float64
	hasAltitude     bool
	combined        bool
	multidirect     bool
	algorithm       string // -alg Horn|ZevenbergenThorne
	creationOptions []string
	outputType      string
	rawOptions      []string
}

func newDEMConfig(opts []DEMOption) *demConfig {
	c := &demConfig{logger: slogx.Default()}
	for _, o := range opts {
		if o != nil {
			o(c)
		}
	}
	return c
}

// WithDEMLogger sets the structured logger used during DEM processing.
// nil falls back to [slogx.Default].
func WithDEMLogger(l *slog.Logger) DEMOption {
	return func(c *demConfig) {
		if l == nil {
			c.logger = slogx.Default()
			return
		}
		c.logger = l
	}
}

// WithDEMFormat selects the output GDAL driver by name. Maps to `-of`.
// When empty, GDAL infers from the destination path.
func WithDEMFormat(driver string) DEMOption {
	return func(c *demConfig) { c.format = driver }
}

// WithDEMColorFile points at a text file mapping elevation values to
// RGBA colors. REQUIRED when the processing mode is "color-relief";
// ignored for other modes. Format documented at:
// https://gdal.org/programs/gdaldem.html#color-relief
func WithDEMColorFile(path string) DEMOption {
	return func(c *demConfig) { c.colorFile = path }
}

// WithDEMZFactor sets the vertical exaggeration applied before
// computing the slope/hillshade/etc. Useful when elevation units differ
// from horizontal units (e.g. lat-lon DEM with meters elevation —
// pass an `-s` arg via raw options or use WithDEMZFactor with a
// large number to compensate). Maps to `-z`.
func WithDEMZFactor(z float64) DEMOption {
	return func(c *demConfig) {
		c.zFactor = z
		c.hasZFactor = true
	}
}

// WithDEMScale sets the ratio of vertical units to horizontal units
// (e.g. 111120 for a lat-lon DEM with meter elevations). Maps to `-s`.
func WithDEMScale(s float64) DEMOption {
	return func(c *demConfig) {
		c.scale = s
		c.hasScale = true
	}
}

// WithDEMAzimuth sets the sun azimuth (degrees clockwise from north)
// for hillshade. Default 315 (NW). Maps to `-az`.
func WithDEMAzimuth(degrees float64) DEMOption {
	return func(c *demConfig) {
		c.azimuth = degrees
		c.hasAzimuth = true
	}
}

// WithDEMAltitude sets the sun altitude (degrees above horizon) for
// hillshade. Default 45. Maps to `-alt`.
func WithDEMAltitude(degrees float64) DEMOption {
	return func(c *demConfig) {
		c.altitude = degrees
		c.hasAltitude = true
	}
}

// WithDEMCombined enables combined shading (mixes shaded relief with
// elevation-tinted output) for hillshade mode. Maps to `-combined`.
func WithDEMCombined() DEMOption {
	return func(c *demConfig) { c.combined = true }
}

// WithDEMMultidirectional enables multidirectional hillshade — averages
// hillshades from four azimuths for fewer "dark face" artifacts in
// rugged terrain. Maps to `-multidirectional`.
func WithDEMMultidirectional() DEMOption {
	return func(c *demConfig) { c.multidirect = true }
}

// WithDEMAlgorithm picks the slope/aspect algorithm. Valid:
// "Horn" (default, GDAL's traditional formula) or "ZevenbergenThorne"
// (smoother on rugged terrain). Maps to `-alg`.
func WithDEMAlgorithm(alg string) DEMOption {
	return func(c *demConfig) { c.algorithm = alg }
}

// WithDEMCreationOption appends a single driver-specific creation
// option (e.g. "COMPRESS=DEFLATE"). May be called multiple times.
// Maps to repeated `-co KEY=VAL`.
func WithDEMCreationOption(key, val string) DEMOption {
	return func(c *demConfig) {
		c.creationOptions = append(c.creationOptions, fmt.Sprintf("%s=%s", key, val))
	}
}

// WithDEMOutputType sets the output pixel data type. Maps to `-ot`.
func WithDEMOutputType(t string) DEMOption {
	return func(c *demConfig) { c.outputType = t }
}

// WithDEMRawOptions appends raw `gdaldem`-style flag tokens.
func WithDEMRawOptions(args ...string) DEMOption {
	return func(c *demConfig) { c.rawOptions = append(c.rawOptions, args...) }
}

// DEMProcessing runs a `gdaldem`-style raster analysis on a DEM input.
// Thin wrapper around the C entry point behind `gdaldem`
// ([gdal.DEMProcessing]).
//
// mode is one of:
//   - "hillshade" — shaded relief from sun position
//   - "slope" — gradient magnitude in degrees or percent
//   - "aspect" — gradient direction (0-360°)
//   - "color-relief" — elevation-tinted RGB; REQUIRES [WithDEMColorFile]
//   - "TRI" — Terrain Ruggedness Index
//   - "TPI" — Topographic Position Index
//   - "roughness" — local elevation variability
//
// Errors are wrapped with [errs.ErrConvertFailed]; recover via [errors.As]
// into [*GISError]. The Op field is "DEMProcessing".
func DEMProcessing(ctx context.Context, src, dst, mode string, opts ...DEMOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if mode == "" {
		return errs.NewGISError("DEMProcessing", fmt.Sprintf("%s -> %s", src, dst),
			errs.ErrConvertFailed,
			fmt.Errorf("mode must be one of hillshade, slope, aspect, color-relief, TRI, TPI, roughness"))
	}

	cfg := newDEMConfig(opts)

	if mode == "color-relief" && cfg.colorFile == "" {
		return errs.NewGISError("DEMProcessing", src, errs.ErrConvertFailed,
			fmt.Errorf("mode=color-relief requires WithDEMColorFile"))
	}

	if err := validateGDALDriver(cfg.format); err != nil {
		cfg.logger.Error("DEMProcessing: invalid format", "format", cfg.format, "err", err)
		return errs.NewGISError("DEMProcessing", src, errs.ErrConvertFailed, err)
	}

	srcDS, err := gdal.OpenEx(src, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		cfg.logger.Error("DEMProcessing: open source", "src", src, "err", err)
		return errs.NewGISError("DEMProcessing", src, errs.ErrConvertFailed, err)
	}
	defer srcDS.Close()

	args := buildDEMArgs(cfg)
	cfg.logger.Debug("DEMProcessing: invoking",
		"src", src, "dst", dst, "mode", mode, "args", args)

	out, dErr := gdal.DEMProcessing(dst, srcDS, mode, cfg.colorFile, args)
	if dErr != nil {
		return errs.NewGISError("DEMProcessing", fmt.Sprintf("%s -> %s", src, dst),
			errs.ErrConvertFailed, dErr)
	}
	defer out.Close()
	return nil
}

// buildDEMArgs renders cfg into the []string gdaldem-style argument
// list. Unit-tested separately so the mapping is locked in without CGo.
func buildDEMArgs(cfg *demConfig) []string {
	var args []string
	if cfg.format != "" {
		args = append(args, "-of", cfg.format)
	}
	if cfg.outputType != "" {
		args = append(args, "-ot", cfg.outputType)
	}
	if cfg.algorithm != "" {
		args = append(args, "-alg", cfg.algorithm)
	}
	if cfg.hasZFactor {
		args = append(args, "-z", formatFloat(cfg.zFactor))
	}
	if cfg.hasScale {
		args = append(args, "-s", formatFloat(cfg.scale))
	}
	if cfg.hasAzimuth {
		args = append(args, "-az", formatFloat(cfg.azimuth))
	}
	if cfg.hasAltitude {
		args = append(args, "-alt", formatFloat(cfg.altitude))
	}
	if cfg.combined {
		args = append(args, "-combined")
	}
	if cfg.multidirect {
		args = append(args, "-multidirectional")
	}
	for _, co := range cfg.creationOptions {
		args = append(args, "-co", co)
	}
	args = append(args, cfg.rawOptions...)
	return args
}
