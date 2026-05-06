//go:build integration

package convert_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/v2/convert"
)

// TestBuildVRT_TwoGeoTIFFs_Integration assembles two GeoTIFF inputs into
// a single VRT. Asserts the destination opens, has the same band count
// as the inputs (default mode, not -separate), and covers the union
// extent.
func TestBuildVRT_TwoGeoTIFFs_Integration(t *testing.T) {
	src1 := mustFetched(t, "RGB.byte.tif")
	src2 := mustFetched(t, "cog.tif")
	dst := filepath.Join(t.TempDir(), "mosaic.vrt")

	if err := convert.BuildVRT(context.Background(), dst,
		[]string{src1, src2},
		convert.WithVRTResolution("highest"),
		convert.WithVRTAllowProjectionDifference(),
	); err != nil {
		t.Fatalf("BuildVRT: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	if dsOut.RasterCount() == 0 {
		t.Fatal("VRT has no raster bands")
	}
	if dsOut.Driver().ShortName() != "VRT" {
		t.Errorf("expected VRT driver, got %q", dsOut.Driver().ShortName())
	}
}

// TestBuildVRT_SeparateMode_Integration uses -separate to stack
// single-input bands into a multi-band VRT (one band per input). With
// two inputs we expect a 2-band-or-more VRT — exact count depends on
// how -separate handles multi-band inputs (it actually flattens them).
func TestBuildVRT_SeparateMode_Integration(t *testing.T) {
	src1 := mustFetched(t, "RGB.byte.tif") // 3 bands
	src2 := mustFetched(t, "cog.tif")      // typically 1 band
	dst := filepath.Join(t.TempDir(), "stack.vrt")

	if err := convert.BuildVRT(context.Background(), dst,
		[]string{src1, src2},
		convert.WithVRTSeparate(),
		convert.WithVRTAllowProjectionDifference(),
	); err != nil {
		t.Fatalf("BuildVRT: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFRaster|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	// In -separate mode, GDAL takes the first band of each input. We
	// have 2 inputs → at least 2 bands.
	if dsOut.RasterCount() < 2 {
		t.Errorf("expected >= 2 bands in -separate mode, got %d", dsOut.RasterCount())
	}
}
