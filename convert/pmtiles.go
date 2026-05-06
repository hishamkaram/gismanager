package convert

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/protomaps/go-pmtiles/pmtiles"

	"github.com/hishamkaram/gismanager/internal/errs"
	"github.com/hishamkaram/gismanager/internal/slogx"
)

// PMTilesOption configures [ToPMTiles] at call time.
type PMTilesOption func(*pmtilesConfig)

type pmtilesConfig struct {
	logger      *slog.Logger
	deduplicate bool
}

func newPMTilesConfig(opts []PMTilesOption) *pmtilesConfig {
	cfg := &pmtilesConfig{
		logger:      slogx.Default(),
		deduplicate: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	return cfg
}

// WithPMTilesLogger threads a custom *slog.Logger through ToPMTiles.
// The underlying go-pmtiles library uses stdlib *log.Logger; the
// gismanager wrapper bridges via [slog.NewLogLogger] so structured
// log records still flow through the manager's configured handler
// (file, JSON, OTel, etc.).
func WithPMTilesLogger(l *slog.Logger) PMTilesOption {
	return func(c *pmtilesConfig) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithPMTilesDeduplicate controls whether the PMTiles writer deduplicates
// identical tile bodies (default: true). Deduplication shrinks output
// at the cost of one extra hash-and-compare per tile during the
// conversion. Pass `false` only if you have a very specific reason to
// preserve byte-identical duplicates (e.g. testing the writer).
func WithPMTilesDeduplicate(dedupe bool) PMTilesOption {
	return func(c *pmtilesConfig) { c.deduplicate = dedupe }
}

// ToPMTiles converts the MBTiles archive at src into a PMTiles v3
// archive at dst. The source must be a valid MBTiles file — typically
// produced by [ConvertRaster] with `WithRasterFormat("MBTILES")` for
// raster inputs, or by an upstream tippecanoe / QGIS / GDAL pipeline
// for vector inputs.
//
// Two-stage pipeline for callers who start from a raster:
//
//	if err := convert.ConvertRaster(ctx, "scene.tif", "scene.mbtiles",
//	    convert.WithRasterFormat("MBTILES")); err != nil { ... }
//	if err := convert.ToPMTiles(ctx, "scene.mbtiles", "scene.pmtiles"); err != nil { ... }
//
// PMTiles is the dominant 2026 single-file tile archive format,
// suitable for serverless distribution from S3, HTTP, or any
// range-request-capable storage. See https://docs.protomaps.com/pmtiles/.
//
// Direct raster -> PMTiles (skipping the intermediate MBTiles) is
// tracked as a v1.5 follow-up; for v1.4 the two-step path is
// the supported route.
//
// Errors are wrapped with [errs.ErrConvertFailed]; recover the underlying
// go-pmtiles or filesystem error via [errors.As] into [*GISError]. The
// Op field is "ToPMTiles".
func ToPMTiles(ctx context.Context, src, dst string, opts ...PMTilesOption) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cfg := newPMTilesConfig(opts)

	// Stat the source so a missing-file error surfaces with a useful
	// path; otherwise the underlying pmtiles.Convert error mentions
	// only the missing inode without naming the user-supplied path.
	if _, err := os.Stat(src); err != nil {
		cfg.logger.Error("ToPMTiles: stat source", "src", src, "err", err)
		return errs.NewGISError("ToPMTiles", src, errs.ErrConvertFailed, err)
	}

	// pmtiles.Convert wants a temp scratch *os.File for staging the
	// final write; we own its lifecycle so the caller doesn't need to.
	tmpDir := filepath.Dir(dst)
	if tmpDir == "" {
		tmpDir = "."
	}
	tmp, err := os.CreateTemp(tmpDir, ".gismanager-pmtiles-*.tmp")
	if err != nil {
		cfg.logger.Error("ToPMTiles: create temp file", "dir", tmpDir, "err", err)
		return errs.NewGISError("ToPMTiles", dst, errs.ErrConvertFailed, err)
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	cfg.logger.Debug("ToPMTiles: invoking pmtiles.Convert",
		"src", src, "dst", dst, "deduplicate", cfg.deduplicate)

	// Bridge the *slog.Logger to a stdlib *log.Logger so the underlying
	// pmtiles.Convert progress lines flow through the manager's
	// configured handler (file, JSON, OTel bridge, etc.) just like
	// every other library log.
	stdLogger := slog.NewLogLogger(cfg.logger.Handler(), slog.LevelInfo)

	if err := pmtiles.Convert(stdLogger, src, dst, cfg.deduplicate, tmp); err != nil {
		return errs.NewGISError("ToPMTiles",
			fmt.Sprintf("%s -> %s", src, dst),
			errs.ErrConvertFailed, err)
	}
	return nil
}
