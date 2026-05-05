//go:build integration

package gismanager_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager"
)

// TestReprojectRaster_GeoTIFFToWebMercator_Integration exercises the
// dominant raster reprojection in 2025-26 web tile pipelines: a UTM-zone
// GeoTIFF (EPSG:32618) reprojected to Web Mercator (EPSG:3857). Asserts
// the destination opens, has the expected CRS, and contains pixel data.
func TestReprojectRaster_GeoTIFFToWebMercator_Integration(t *testing.T) {
	src := mustFetched(t, "RGB.byte.tif")
	dst := filepath.Join(t.TempDir(), "RGB.3857.tif")

	if err := gismanager.ReprojectRaster(context.Background(),
		src, dst, "EPSG:32618", "EPSG:3857",
		gismanager.WithRasterFormat("GTiff"),
		gismanager.WithRasterResamplingAlg("bilinear"),
	); err != nil {
		t.Fatalf("ReprojectRaster: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() == 0 {
		t.Fatal("destination has no raster bands")
	}

	// Verify the projection includes the EPSG:3857 marker. We don't use
	// a parsed SR comparison because Dataset.Projection() returns WKT
	// directly and GDAL has multiple equivalent ways to express 3857.
	wkt := dsOut.Projection()
	if wkt == "" {
		t.Errorf("expected non-empty projection WKT on warped output")
	}
	// The Web Mercator WKT contains "Mercator" and either AUTHORITY
	// "3857" or one of the EPSG:3857 aliases.
	if !containsAny(wkt, []string{"3857", "Mercator_1SP", "WGS_1984_Web_Mercator", "Pseudo-Mercator"}) {
		t.Errorf("expected Web Mercator marker in projection WKT, got:\n%s", wkt)
	}
}

func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(haystack); i++ {
			if haystack[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}
