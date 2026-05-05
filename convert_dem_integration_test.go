//go:build integration

package gismanager_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager"
)

// TestDEMProcessing_Hillshade_Integration runs hillshade against the
// rasterio fixture (RGB.byte.tif is 3-band but GDAL accepts the first
// band as the elevation grid). Asserts the destination is a single-band
// raster.
func TestDEMProcessing_Hillshade_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "hs.tif")

	if err := gismanager.DEMProcessing(context.Background(), src, dst, "hillshade",
		gismanager.WithDEMFormat("GTiff"),
		gismanager.WithDEMAzimuth(315),
		gismanager.WithDEMAltitude(45),
	); err != nil {
		t.Fatalf("DEMProcessing hillshade: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	// Hillshade produces a single Byte band.
	if dsOut.RasterCount() != 1 {
		t.Errorf("expected 1-band hillshade output, got %d", dsOut.RasterCount())
	}
}

// TestDEMProcessing_Slope_Integration runs slope against the same input.
// Slope is a single-band Float32 raster.
func TestDEMProcessing_Slope_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "slope.tif")

	if err := gismanager.DEMProcessing(context.Background(), src, dst, "slope",
		gismanager.WithDEMFormat("GTiff"),
	); err != nil {
		t.Fatalf("DEMProcessing slope: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() != 1 {
		t.Errorf("expected 1-band slope output, got %d", dsOut.RasterCount())
	}
}

// TestDEMProcessing_TRI_Integration runs Terrain Ruggedness Index.
// Single-band Float32 output describing local elevation variability.
func TestDEMProcessing_TRI_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "tri.tif")

	if err := gismanager.DEMProcessing(context.Background(), src, dst, "TRI",
		gismanager.WithDEMFormat("GTiff"),
	); err != nil {
		t.Fatalf("DEMProcessing TRI: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() != 1 {
		t.Errorf("expected 1-band TRI output, got %d", dsOut.RasterCount())
	}
}
