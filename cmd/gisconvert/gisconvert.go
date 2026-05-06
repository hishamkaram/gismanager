// Command gisconvert exposes gismanager's conversion subsystem as a CLI:
// vector format conversion (the ogr2ogr equivalent), raster format
// conversion (the gdal_translate equivalent), Cloud-Optimized GeoTIFF
// generation, and raster reprojection (the gdalwarp equivalent).
//
// Usage:
//
//	gisconvert -mode vector -src in.shp -dst out.gpkg -format GPKG
//	gisconvert -mode vector -src in.geojson -dst out.gpkg \
//	    -t-srs EPSG:3857 -bbox -25,-40,60,40 -where "CONTINENT='Africa'"
//	gisconvert -mode raster -src in.tif -dst out.png -format PNG -bands 1,2,3
//	gisconvert -mode raster -src in.tif -dst out.cog.tif -cog
//	gisconvert -mode raster -src in.tif -dst out.warp.tif \
//	    -s-srs EPSG:32618 -t-srs EPSG:3857 -resample bilinear
//
// gisconvert is a thin shell over [gismanager.ConvertVector],
// [gismanager.ConvertRaster], [gismanager.ToCOG], and
// [gismanager.ReprojectRaster]. It uses the stdlib `flag` package — no
// runtime dependency beyond the gismanager library itself.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/hishamkaram/gismanager"
	"github.com/hishamkaram/gismanager/cmd/internal/cli"
)

func main() { os.Exit(realMain()) }

// realMain is the testable entry point; see the matching helper in
// cmd/gismanager/gismanager.go for rationale.
func realMain() int {
	ctx, cancel := cli.SignalContext(context.Background())
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		if errors.Is(err, cli.ErrVersionRequested) {
			return 0
		}
		slog.Error("gisconvert", "err", err)
		return 1
	}
	return 0
}

// run is the entry point separated from main() so it can be tested
// directly with arbitrary argument slices and a controllable context.
func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("gisconvert", flag.ContinueOnError)
	mode := fs.String("mode", "", "conversion mode: 'vector' or 'raster'")
	src := fs.String("src", "", "source path (file or /vsi*/-prefixed URL)")
	dst := fs.String("dst", "", "destination path (file or /vsi*/-prefixed URL)")
	format := fs.String("format", "", "output driver name (GPKG, GeoJSON, GTiff, COG, PNG, ...). When empty, GDAL infers from -dst extension.")
	versionFlag := fs.Bool("version", false, "print build version and exit")

	// Vector flags.
	tSRS := fs.String("t-srs", "", "target SRS (e.g. EPSG:3857). Required for 'raster' mode reproject; optional for 'vector' mode reproject.")
	sSRS := fs.String("s-srs", "", "source SRS override (rarely needed)")
	bbox := fs.String("bbox", "", "bounding box 'minX,minY,maxX,maxY' (vector: source CRS, raster reproject: target CRS)")
	where := fs.String("where", "", "OGR SQL WHERE filter, vector mode only")
	simplify := fs.Float64("simplify", 0, "Douglas-Peucker simplification tolerance, vector mode only")
	selectFields := fs.String("select", "", "comma-separated attribute fields to keep (vector mode only)")
	layer := fs.String("layer", "", "destination layer name (vector mode only)")
	overwrite := fs.Bool("overwrite", false, "overwrite the destination if it exists (vector mode only)")

	// Raster flags.
	bands := fs.String("bands", "", "comma-separated 1-based band indices to keep (raster mode only)")
	resample := fs.String("resample", "", "resampling algorithm (near, bilinear, cubic, ...) (raster mode only)")
	tr := fs.String("tr", "", "target resolution 'xRes,yRes' (raster mode only)")
	cog := fs.Bool("cog", false, "raster shortcut: emit a Cloud-Optimized GeoTIFF using ToCOG defaults")
	cutlineDS := fs.String("cutline", "", "vector dataset path used as a cookie-cutter for raster reprojection")
	cutlineLayer := fs.String("cutline-layer", "", "layer name within -cutline (optional)")

	// Repeated -co flags (creation options) for raster mode.
	var cos rasterCreationOptions
	fs.Var(&cos, "co", "raster creation option, KEY=VAL (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *versionFlag {
		cli.PrintVersion(os.Stdout, "gisconvert")
		return cli.ErrVersionRequested
	}
	if *mode == "" || *src == "" || *dst == "" {
		return errors.New("gisconvert: -mode, -src, and -dst are all required")
	}

	switch *mode {
	case "vector":
		return runVector(ctx, *src, *dst, *format, *sSRS, *tSRS, *bbox,
			*where, *simplify, *selectFields, *layer, *overwrite)
	case "raster":
		return runRaster(ctx, *src, *dst, *format, *sSRS, *tSRS, *bbox,
			*bands, *resample, *tr, *cog, *cutlineDS, *cutlineLayer, []string(cos))
	default:
		return fmt.Errorf("gisconvert: -mode must be 'vector' or 'raster', got %q", *mode)
	}
}

func runVector(ctx context.Context, src, dst, format, sSRS, tSRS, bbox,
	where string, simplify float64, selectFields, layer string, overwrite bool,
) error {
	opts := []gismanager.VectorConvertOption{}
	if format != "" {
		opts = append(opts, gismanager.WithVectorFormat(format))
	}
	if sSRS != "" {
		opts = append(opts, gismanager.WithVectorSourceSRS(sSRS))
	}
	if tSRS != "" {
		opts = append(opts, gismanager.WithVectorTargetSRS(tSRS))
	}
	if bbox != "" {
		minX, minY, maxX, maxY, err := parseBBox(bbox)
		if err != nil {
			return err
		}
		opts = append(opts, gismanager.WithVectorBoundingBox(minX, minY, maxX, maxY))
	}
	if where != "" {
		opts = append(opts, gismanager.WithVectorWhere(where))
	}
	if simplify > 0 {
		opts = append(opts, gismanager.WithVectorSimplify(simplify))
	}
	if selectFields != "" {
		fields := strings.Split(selectFields, ",")
		opts = append(opts, gismanager.WithVectorSelectFields(fields...))
	}
	if layer != "" {
		opts = append(opts, gismanager.WithVectorLayerName(layer))
	}
	if overwrite {
		opts = append(opts, gismanager.WithVectorOverwrite())
	}
	return gismanager.ConvertVector(ctx, src, dst, opts...)
}

func runRaster(ctx context.Context, src, dst, format, sSRS, tSRS, bbox,
	bands, resample, tr string, cog bool,
	cutlineDS, cutlineLayer string, creationOpts []string,
) error {
	opts := []gismanager.RasterConvertOption{}
	if format != "" {
		opts = append(opts, gismanager.WithRasterFormat(format))
	}
	for _, co := range creationOpts {
		k, v, ok := strings.Cut(co, "=")
		if !ok {
			return fmt.Errorf("gisconvert: -co value %q must be KEY=VAL", co)
		}
		opts = append(opts, gismanager.WithRasterCreationOption(k, v))
	}
	if bbox != "" {
		minX, minY, maxX, maxY, err := parseBBox(bbox)
		if err != nil {
			return err
		}
		opts = append(opts, gismanager.WithRasterOutputBounds(minX, minY, maxX, maxY))
	}
	if bands != "" {
		bs, err := parseBands(bands)
		if err != nil {
			return err
		}
		opts = append(opts, gismanager.WithRasterBands(bs...))
	}
	if resample != "" {
		opts = append(opts, gismanager.WithRasterResamplingAlg(resample))
	}
	if tr != "" {
		xRes, yRes, err := parseRes(tr)
		if err != nil {
			return err
		}
		opts = append(opts, gismanager.WithRasterTargetResolution(xRes, yRes))
	}
	if cutlineDS != "" {
		opts = append(opts, gismanager.WithRasterCutline(cutlineDS, cutlineLayer))
	}

	// Reproject mode: both -s-srs and -t-srs supplied (or just -t-srs;
	// gdalwarp can pick up the source SRS from the input).
	if tSRS != "" {
		if sSRS == "" {
			return errors.New("gisconvert: raster reprojection requires both -s-srs and -t-srs")
		}
		return gismanager.ReprojectRaster(ctx, src, dst, sSRS, tSRS, opts...)
	}

	// COG shortcut.
	if cog {
		return gismanager.ToCOG(ctx, src, dst, opts...)
	}

	return gismanager.ConvertRaster(ctx, src, dst, opts...)
}

// parseBBox parses "minX,minY,maxX,maxY" into four float64s.
func parseBBox(s string) (minX, minY, maxX, maxY float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return 0, 0, 0, 0, fmt.Errorf("gisconvert: -bbox must be minX,minY,maxX,maxY (4 floats), got %q", s)
	}
	floats := make([]float64, 4)
	for i, p := range parts {
		v, parseErr := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if parseErr != nil {
			return 0, 0, 0, 0, fmt.Errorf("gisconvert: -bbox: bad float %q: %w", p, parseErr)
		}
		floats[i] = v
	}
	return floats[0], floats[1], floats[2], floats[3], nil
}

// parseBands parses "1,2,3" into []int.
func parseBands(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("gisconvert: -bands: bad int %q: %w", p, err)
		}
		out = append(out, v)
	}
	return out, nil
}

// parseRes parses "xRes,yRes" into (x, y) float64s.
func parseRes(s string) (xRes, yRes float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("gisconvert: -tr must be xRes,yRes, got %q", s)
	}
	xRes, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("gisconvert: -tr xRes: %w", err)
	}
	yRes, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("gisconvert: -tr yRes: %w", err)
	}
	return xRes, yRes, nil
}

// rasterCreationOptions implements flag.Value so users can pass `-co`
// repeatedly to accumulate creation options.
type rasterCreationOptions []string

func (r *rasterCreationOptions) String() string { return strings.Join(*r, " ") }

func (r *rasterCreationOptions) Set(v string) error {
	*r = append(*r, v)
	return nil
}
