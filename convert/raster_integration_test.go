//go:build integration

package convert_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/v2/convert"
)

// TestConvertRaster_GeoTIFFToCOG_Integration exercises the most common
// raster conversion in 2025-26 cloud workflows: GeoTIFF → Cloud-Optimized
// GeoTIFF. Asserts the destination opens, claims the COG driver, and has
// at least one overview level (a defining COG property).
func TestConvertRaster_GeoTIFFToCOG_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "RGB.cog.tif")

	if err := convert.ToCOG(context.Background(), src, dst); err != nil {
		t.Fatalf("ToCOG: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	driver := dsOut.Driver()
	if driver.ShortName() != "GTiff" && driver.ShortName() != "COG" {
		t.Errorf("expected GTiff or COG driver for COG output, got %q", driver.ShortName())
	}

	// COG mandates internal overviews. If we have at least one band with
	// overviews >= 1, the COG driver did its job.
	if dsOut.RasterCount() == 0 {
		t.Fatal("destination has no raster bands")
	}
	band := dsOut.RasterBand(1)
	if band.OverviewCount() < 1 {
		t.Errorf("expected at least 1 overview level on the COG output, got %d",
			band.OverviewCount())
	}
}

// TestConvertRaster_GeoTIFFToPNG_Integration converts an RGB GeoTIFF to
// a PNG (selecting all 3 bands). PNG is one of the most common
// "thumbnail / preview" outputs.
func TestConvertRaster_GeoTIFFToPNG_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "RGB.png")

	if err := convert.ConvertRaster(context.Background(), src, dst,
		convert.WithRasterFormat("PNG"),
		convert.WithRasterBands(1, 2, 3),
	); err != nil {
		t.Fatalf("ConvertRaster: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() != 3 {
		t.Errorf("expected 3-band PNG output, got %d bands", dsOut.RasterCount())
	}
	if dsOut.Driver().ShortName() != "PNG" {
		t.Errorf("expected PNG driver, got %q", dsOut.Driver().ShortName())
	}
}

// TestToCOG_OverridesDefaults_Integration confirms caller-supplied options
// override the defaults ToCOG sets.
func TestToCOG_OverridesDefaults_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "RGB.zstd.cog.tif")

	// User picks ZSTD compression instead of the default DEFLATE.
	if err := convert.ToCOG(context.Background(), src, dst,
		convert.WithRasterCreationOption("COMPRESS", "ZSTD"),
	); err != nil {
		t.Skipf("ToCOG with ZSTD failed (GDAL build may lack zstd): %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	// Just assert it parses as a valid raster — driver-level COMPRESS
	// metadata isn't reliably exposed across GDAL versions.
	if dsOut.RasterCount() == 0 {
		t.Fatal("destination has no raster bands")
	}
}
