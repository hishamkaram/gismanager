//go:build integration

package convert_test

// Integration test for the v1.4 GeoParquet support. Exercises the
// full read/write contract: writes a GeoParquet from the tracked
// legacy GeoJSON fixture via ConvertVector, then opens it via the
// new .parquet extension dispatch and asserts feature-count parity
// with the source.
//
// Requires the dev/test-runner image to be built from the
// `ubuntu-full` GDAL base (the default since v1.4) — `ubuntu-small`
// excludes the Parquet driver and the test would fail at the first
// ConvertVector with a "driver Parquet not registered" GDAL error.

import (
	"context"
	"testing"

	"github.com/hishamkaram/gismanager"
	"github.com/hishamkaram/gismanager/convert"
)

// TestConvertVector_GeoParquetRoundTrip_Integration round-trips a
// vector dataset through GeoParquet: GeoJSON -> Parquet -> read.
// Verifies (a) the Parquet driver is registered (the ubuntu-full
// image swap landed correctly), (b) ConvertVector accepts the
// "Parquet" format, and (c) OpenSource dispatches the .parquet
// extension to the right driver.
func TestConvertVector_GeoParquetRoundTrip_Integration(t *testing.T) {
	src := "../testdata/neighborhood_names_gis.geojson"
	parquetPath := "/vsimem/test_geoparquet_roundtrip.parquet"

	// 1. Source-side baseline: count features in the GeoJSON.
	mgr, err := gismanager.New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srcDS, err := mgr.OpenSource(context.Background(), src, 0)
	if err != nil {
		t.Fatalf("OpenSource(%s): %v", src, err)
	}
	srcLayer := srcDS.LayerByIndex(0)
	srcCount, ok := srcLayer.FeatureCount(true)
	if !ok {
		t.Fatal("FeatureCount on source GeoJSON failed")
	}
	srcDS.Destroy()
	if srcCount == 0 {
		t.Fatal("source GeoJSON has zero features; test fixture broken")
	}

	// 2. Convert GeoJSON -> GeoParquet via the new format support.
	if err := convert.ConvertVector(context.Background(), src, parquetPath,
		convert.WithVectorFormat(parquetDriverName),
		convert.WithVectorOverwrite(),
	); err != nil {
		t.Fatalf("ConvertVector to GeoParquet: %v", err)
	}

	// 3. Read the GeoParquet back via OpenSource — the .parquet
	//    extension must dispatch to the Parquet driver.
	dstDS, err := mgr.OpenSource(context.Background(), parquetPath, 0)
	if err != nil {
		t.Fatalf("OpenSource(GeoParquet): %v", err)
	}
	defer dstDS.Destroy()

	dstLayer := dstDS.LayerByIndex(0)
	dstCount, ok := dstLayer.FeatureCount(true)
	if !ok {
		t.Fatal("FeatureCount on round-tripped GeoParquet failed")
	}

	// 4. Assert feature-count parity. Geometry-level fidelity is GDAL's
	//    contract; here we just lock in that the driver dispatch and
	//    the conversion round-trip aren't silently dropping features.
	if dstCount != srcCount {
		t.Errorf("feature count mismatch: GeoJSON=%d, GeoParquet=%d", srcCount, dstCount)
	}
	t.Logf("GeoParquet round-trip ok: %d features", dstCount)
}

// parquetDriverName mirrors the unexported [gismanager.parquetDriver]
// constant without exposing it. We can't reach the unexported name from
// this external `convert_test` package, so we duplicate the string
// here. If a future PR exports the driver-name constants (a v2 change),
// this duplication goes away.
const parquetDriverName = "Parquet"
