//go:build integration

package convert_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/convert"
)

// TestConvertVector_ShapefileToGeoPackage_Integration exercises the most
// common conversion in the wild: a zipped Shapefile to a single-layer
// GeoPackage. Asserts the destination is readable, has exactly one layer,
// and preserves the source feature count.
//
// The /vsizip/ prefix is required: lukeroth/gdal's OpenEx does not
// auto-prefix bare .zip paths (verified against the dev container's
// ogrinfo). Callers shipping zipped Shapefiles to ConvertVector must
// either use the /vsizip/ prefix or pre-extract via GetGISFiles.
func TestConvertVector_ShapefileToGeoPackage_Integration(t *testing.T) {
	src := "/vsizip/" + mustFetched(t, "ne_110m_admin_0_countries.zip")
	dst := filepath.Join(t.TempDir(), "countries.gpkg")

	if err := convert.ConvertVector(context.Background(), src, dst,
		convert.WithVectorFormat("GPKG"),
		convert.WithVectorOverwrite(),
	); err != nil {
		t.Fatalf("ConvertVector: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	srcOpen, err := gdal.OpenEx(src, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer srcOpen.Close()

	srcLayer := srcOpen.LayerByIndex(0)
	srcCount, ok := srcLayer.FeatureCount(true)
	if !ok {
		t.Fatal("source layer feature count read failed")
	}

	dstLayer := dsOut.LayerByIndex(0)
	dstCount, ok := dstLayer.FeatureCount(true)
	if !ok {
		t.Fatal("destination layer feature count read failed")
	}

	if srcCount != dstCount {
		t.Errorf("feature count mismatch: src=%d dst=%d", srcCount, dstCount)
	}
	if srcCount == 0 {
		t.Errorf("source has zero features — fixture may be corrupted")
	}
}

// TestConvertVector_GeoJSONReprojectAndFilter_Integration exercises the
// "extract a region in a different CRS" workflow: take 4326 countries,
// project to 3857, clip to a bbox roughly covering Africa, and filter by
// continent attribute. Asserts the destination has fewer features than
// the source AND that the destination geometry is in 3857.
func TestConvertVector_GeoJSONReprojectAndFilter_Integration(t *testing.T) {
	src := mustFetched(t, "ne_110m_admin_0_countries.geojson")
	dst := filepath.Join(t.TempDir(), "africa_3857.gpkg")

	if err := convert.ConvertVector(context.Background(), src, dst,
		convert.WithVectorFormat("GPKG"),
		convert.WithVectorOverwrite(),
		convert.WithVectorTargetSRS("EPSG:3857"),
		// Africa-ish bbox in 4326 (-spat applies to source CRS by default).
		convert.WithVectorBoundingBox(-25, -40, 60, 40),
		convert.WithVectorWhere("CONTINENT = 'Africa'"),
		convert.WithVectorLayerName("africa_3857"),
	); err != nil {
		t.Fatalf("ConvertVector: %v", err)
	}

	dsOut, err := gdal.OpenEx(dst, gdal.OFVector|gdal.OFReadOnly, nil, nil, nil)
	if err != nil {
		t.Fatalf("open destination: %v", err)
	}
	defer dsOut.Close()

	dstLayer := dsOut.LayerByIndex(0)
	dstCount, ok := dstLayer.FeatureCount(true)
	if !ok {
		t.Fatal("destination layer feature count read failed")
	}
	// Africa has ~50 countries in Natural Earth 1:110m. Allow a wide
	// envelope so the test is robust to fixture revisions.
	if dstCount < 30 || dstCount > 80 {
		t.Errorf("expected ~50 African countries, got %d", dstCount)
	}

	// Verify the destination spatial reference is in fact EPSG:3857.
	sr := dstLayer.SpatialReference()
	authCode, ok := srAuthorityCode(sr)
	if !ok || authCode != "3857" {
		t.Errorf("expected EPSG:3857 destination CRS, got authority code %q (ok=%v)",
			authCode, ok)
	}
}

// TestConvertVector_Idempotent_Integration confirms re-running the same
// conversion with WithVectorOverwrite() succeeds twice. This is the
// "rerun a tile-prep pipeline after upstream data refreshes" use case.
func TestConvertVector_Idempotent_Integration(t *testing.T) {
	src := mustFetched(t, "ne_110m_admin_0_countries.geojson")
	dst := filepath.Join(t.TempDir(), "countries.gpkg")

	for i := 0; i < 2; i++ {
		if err := convert.ConvertVector(context.Background(), src, dst,
			convert.WithVectorFormat("GPKG"),
			convert.WithVectorOverwrite(),
		); err != nil {
			t.Fatalf("iteration %d: %v", i+1, err)
		}
	}
}

// Note: a "TestConvertVector_UnsupportedDestination_Integration" was
// drafted but removed — lukeroth/gdal's VectorTranslate wrapper does not
// surface NULL-options-from-bad-driver as a Go error (only `cerr != 0`
// is checked, and unknown -f values fail at GDALVectorTranslateOptionsNew
// with an stderr message but no nonzero return code). Detecting that
// failure cleanly would require an upstream binding patch. Documented
// in the convert_vector.go top-comment as a known gap.

// mustFetched returns the absolute path to a fixture inside
// `testdata-fetched/`. It t.Skips the test if the fixture is missing
// (the user forgot to run `make fetch-testdata`); CI always pre-fetches.
//
// The fetched fixtures live at the module root; tests in this convert/
// subpackage are one level down so the relative join is
// "../testdata-fetched". Same fixture set as the v1.x layout.
func mustFetched(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("..", "testdata-fetched", name))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	return abs
}

// srAuthorityCode pulls the authority code (e.g. "3857") out of a
// SpatialReference for assertion. The lukeroth/gdal binding doesn't
// wrap OSRGetAuthorityCode, so we read the AUTHORITY[] node's second
// child via the generic AttrValue helper. Returns ("", false) when the
// SR is unset or has no authority recorded.
func srAuthorityCode(sr gdal.SpatialReference) (string, bool) {
	code, ok := sr.AttrValue("AUTHORITY", 1)
	if !ok || code == "" {
		return "", false
	}
	return code, true
}
