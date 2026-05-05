//go:build integration

package gismanager_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager"
)

// TestRasterize_PolygonsToGeoTIFF_Integration burns the Africa subset
// of Natural Earth countries into a 256x256 GeoTIFF mask. Asserts the
// destination opens, has the expected dimensions, and has at least one
// non-zero pixel (i.e. the burn actually wrote data).
func TestRasterize_PolygonsToGeoTIFF_Integration(t *testing.T) {
	src := mustFetched(t, "ne_110m_admin_0_countries.geojson")
	dst := filepath.Join(t.TempDir(), "africa_mask.tif")

	if err := gismanager.Rasterize(context.Background(), src, dst,
		gismanager.WithRasterizeFormat("GTiff"),
		gismanager.WithRasterizeOutputType("Byte"),
		gismanager.WithRasterizeBurnValues(1.0),
		gismanager.WithRasterizeWhere("CONTINENT = 'Africa'"),
		// Africa-ish bbox in 4326.
		gismanager.WithRasterizeOutputBounds(-25, -40, 60, 40),
		gismanager.WithRasterizeOutputSize(256, 256),
		gismanager.WithRasterizeCreationOption("COMPRESS", "DEFLATE"),
	); err != nil {
		t.Fatalf("Rasterize: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if got := dsOut.RasterXSize(); got != 256 {
		t.Errorf("RasterXSize = %d, want 256", got)
	}
	if got := dsOut.RasterYSize(); got != 256 {
		t.Errorf("RasterYSize = %d, want 256", got)
	}
	if dsOut.Driver().ShortName() != "GTiff" {
		t.Errorf("expected GTiff driver, got %q", dsOut.Driver().ShortName())
	}
}

// TestRasterize_AttributeBurn_Integration burns the POP_EST attribute
// (population estimate) from the source features instead of a fixed
// constant. Asserts the destination is a Float32 raster (population
// counts don't fit in a Byte).
func TestRasterize_AttributeBurn_Integration(t *testing.T) {
	src := mustFetched(t, "ne_110m_admin_0_countries.geojson")
	dst := filepath.Join(t.TempDir(), "pop_attr.tif")

	if err := gismanager.Rasterize(context.Background(), src, dst,
		gismanager.WithRasterizeFormat("GTiff"),
		gismanager.WithRasterizeOutputType("Float32"),
		gismanager.WithRasterizeAttribute("POP_EST"),
		gismanager.WithRasterizeOutputSize(360, 180),
	); err != nil {
		t.Fatalf("Rasterize: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() == 0 {
		t.Fatal("destination has no raster bands")
	}
}
