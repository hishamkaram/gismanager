package gismanager

import (
	"errors"
	"testing"

	"github.com/lukeroth/gdal"
)

// TestGetDriver_Table is the table-driven canonical test for the
// path-or-connection-string → OGR driver dispatch. Replaces the ad-hoc
// per-extension assertions in manager_test.go::TestGetDriver with a
// single source of truth. Every supported extension + edge case lives
// here so adding a new format means adding one row.
func TestGetDriver_Table(t *testing.T) {
	mgr, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name       string
		path       string
		wantDriver string // empty → expect ErrUnsupportedFormat
	}{
		// --- supported extensions, lowercase ---
		{name: "geopackage", path: "data/sample.gpkg", wantDriver: geopackageDriver},
		{name: "shapefile", path: "data/cities.shp", wantDriver: shapeFileDriver},
		{name: "zipped-shapefile", path: "data/cities.zip", wantDriver: shapeFileDriver},
		{name: "geojson", path: "data/places.geojson", wantDriver: geoJSONDriver},
		{name: "json-as-geojson", path: "data/places.json", wantDriver: geoJSONDriver},
		{name: "kml", path: "data/places.kml", wantDriver: kmlDriver},

		// --- supported extensions, uppercase / mixed (filepath.Ext is
		// case-sensitive but GetDriver lowercases internally, so these
		// must dispatch the same as their lowercase form) ---
		{name: "shapefile-uppercase", path: "data/CITIES.SHP", wantDriver: shapeFileDriver},
		{name: "geojson-mixed", path: "data/Places.GeoJSON", wantDriver: geoJSONDriver},
		{name: "kml-uppercase", path: "data/PLACES.KML", wantDriver: kmlDriver},
		{name: "geopackage-mixed", path: "data/Sample.Gpkg", wantDriver: geopackageDriver},

		// --- PostgreSQL connection strings (pgRegex match) ---
		{name: "pg-typical", path: "PG: host=localhost port=5432 dbname=gis user=u password=p", wantDriver: postgreSQLDriver},
		{name: "pg-no-leading-space", path: "PG:host=localhost port=5432", wantDriver: postgreSQLDriver},
		{name: "pg-leading-space", path: " PG: host=postgis port=5432 dbname=gis", wantDriver: postgreSQLDriver},

		// --- unsupported / unknown ---
		{name: "tiff-raster", path: "data/elevation.tiff", wantDriver: ""},
		{name: "tif-raster", path: "data/elevation.tif", wantDriver: ""},
		{name: "csv", path: "data/points.csv", wantDriver: ""},
		{name: "xml", path: "data/metadata.xml", wantDriver: ""},
		{name: "no-extension", path: "data/somefile", wantDriver: ""},
		{name: "empty-path", path: "", wantDriver: ""},
		{name: "double-extension-tar-gz", path: "data/archive.tar.gz", wantDriver: ""},

		// --- edge: known-supported extension that ISN'T tied to a
		// recognized OGR driver in GetDriver's switch (.gdb is in
		// supportedEXT but GetDriver doesn't dispatch it). Documents
		// the discrepancy; fix is a v2 cleanup. ---
		{name: "gdb-listed-but-unrouted", path: "data/store.gdb", wantDriver: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drv, err := mgr.GetDriver(tc.path)
			if tc.wantDriver == "" {
				if !errors.Is(err, ErrUnsupportedFormat) {
					t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
				}
				if drv != (gdal.OGRDriver{}) {
					t.Errorf("expected zero-value OGRDriver on error, got %#v", drv)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			want := gdal.OGRDriverByName(tc.wantDriver)
			if drv != want {
				t.Errorf("driver mismatch: got %#v, want %#v (driver name %q)", drv, want, tc.wantDriver)
			}
		})
	}
}

// TestSupportedEXT_AllHaveLeadingDot guards the v1.0 bug where the
// "kml" entry in supportedEXT lacked a leading dot — filepath.Ext()
// returns ".kml" so the entry never matched and KML files were
// silently dropped from directory walks. Fixed in PR 5; this test
// keeps it from regressing.
func TestSupportedEXT_AllHaveLeadingDot(t *testing.T) {
	for _, ext := range supportedEXT {
		if ext == "" || ext[0] != '.' {
			t.Errorf("supportedEXT entry %q lacks a leading '.', will never match filepath.Ext output", ext)
		}
	}
}

// TestSupportedEXT_IncludesKML guards the specific KML branch so a
// future trim of supportedEXT can't silently drop KML support again.
func TestSupportedEXT_IncludesKML(t *testing.T) {
	if !isSupported(".kml") {
		t.Errorf("isSupported(.kml) = false; expected true after PR 5 of v1.1 series")
	}
}
