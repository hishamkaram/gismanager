package convert_test

// Note: this file lives under the _test.go non-integration build tag so it
// runs in the unit suite. It exercises GDAL's /vsimem/ in-process VFS
// (no network, no filesystem write) using a tracked legacy fixture as
// the source and /vsimem/ as the destination — proving the conversion
// surface is cloud-aware (any /vsi*/ prefix flows through).

import (
	"context"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/convert"
)

// TestConvertVector_VsiMemDestination_Unit confirms ConvertVector accepts
// a /vsimem/ destination path and the resulting in-process dataset is
// readable. The source is the tracked legacy fixture so this test stays
// in the unit suite (no `make fetch-testdata` prerequisite).
func TestConvertVector_VsiMemDestination_Unit(t *testing.T) {
	src := "../testdata/neighborhood_names_gis.geojson"
	dst := "/vsimem/test_convert_vector.gpkg"
	// /vsimem/ entries are freed when the test process exits; the
	// lukeroth/gdal binding doesn't wrap VSIUnlink so we let GC + EOL
	// cleanup handle it. WithVectorOverwrite() above makes the test
	// re-runnable in -count=N mode.

	if err := convert.ConvertVector(context.Background(), src, dst,
		convert.WithVectorFormat("GPKG"),
		convert.WithVectorOverwrite(),
	); err != nil {
		t.Fatalf("ConvertVector to /vsimem/: %v", err)
	}

	ds, err := gdal.OpenEx(dst, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open /vsimem/ destination: %v", err)
	}
	defer ds.Close()

	layer := ds.LayerByIndex(0)
	count, ok := layer.FeatureCount(true)
	if !ok {
		t.Fatal("feature count read failed")
	}
	if count == 0 {
		t.Fatal("destination has zero features")
	}
}
