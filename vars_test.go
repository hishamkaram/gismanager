package gismanager

import "testing"

// TestPgRegex_Matches and TestPgRegex_DoesNotMatch lock in the
// PostgreSQL connection-string detection used by [(*ManagerConfig).GetDriver]
// to choose between the OGR PG driver and a file-extension dispatch.
// driver_table_test.go's TestGetDriver_Table exercises this regex
// indirectly via the public GetDriver path; the focused tests here
// document the regex contract on its own so a future maintainer can
// reason about the matching rules without reading the dispatch code.

func TestPgRegex_Matches(t *testing.T) {
	cases := []string{
		"PG: host=localhost port=5432 dbname=gis user=u password=p",
		"PG:host=localhost port=5432",
		" PG: host=postgis port=5432 dbname=gis",
		"PG: ",                             // minimal valid form (just the prefix)
		"PG:host=h",                        // single key
		"PG: host=h port=5432\tdbname=gis", // mixed whitespace inside body
		// Leading newline / tab match too — the regex's `\s?`
		// permits any single whitespace char before PG:. Documented
		// here so a future "tighten to literal space only" PR can
		// flip the expectation explicitly rather than discover the
		// looseness via integration regression.
		"\nPG: host=localhost",
		"\tPG: host=localhost",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if !pgRegex.MatchString(in) {
				t.Errorf("pgRegex.MatchString(%q) = false; want true", in)
			}
		})
	}
}

func TestPgRegex_DoesNotMatch(t *testing.T) {
	cases := []string{
		"",                         // empty
		"data/sample.gpkg",         // file path
		"data/sample.shp",          // file path
		"pg: host=localhost",       // lowercase prefix
		"PG host=localhost",        // missing colon
		"prefixPG: host=localhost", // PG: not at the start
		"  PG: host=localhost",     // two leading spaces (regex permits at most one)
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			if pgRegex.MatchString(in) {
				t.Errorf("pgRegex.MatchString(%q) = true; want false", in)
			}
		})
	}
}

// TestSupportedEXT_ContainsExpected guards the closed set of supported
// extensions. Adding a new format MUST update this test alongside
// vars.go's supportedEXT slice.
func TestSupportedEXT_ContainsExpected(t *testing.T) {
	want := map[string]bool{
		".zip":     true,
		".json":    true,
		".gpkg":    true,
		".geojson": true,
		".gdb":     true,
		".kml":     true,
		".shp":     true,
		".parquet": true, // added v1.4 alongside the ubuntu-full base swap
	}
	got := map[string]bool{}
	for _, ext := range supportedEXT {
		got[ext] = true
	}
	for ext := range want {
		if !got[ext] {
			t.Errorf("supportedEXT missing %q", ext)
		}
	}
	for ext := range got {
		if !want[ext] {
			t.Errorf("supportedEXT has unexpected entry %q (update the test if intentional)", ext)
		}
	}
}

// TestDriverNameConstants_NotEmpty is a one-liner per constant that
// would catch a typo'd driver-name like `parquetDriver = ""` slipping
// through code review. Each constant must resolve to a non-empty
// string; GetDriver feeds these directly to gdal.OGRDriverByName,
// which returns a zero-value driver on empty input.
func TestDriverNameConstants_NotEmpty(t *testing.T) {
	cases := map[string]string{
		"geopackageDriver": geopackageDriver,
		"postgreSQLDriver": postgreSQLDriver,
		"shapeFileDriver":  shapeFileDriver,
		"geoJSONDriver":    geoJSONDriver,
		"kmlDriver":        kmlDriver,
		"parquetDriver":    parquetDriver,
	}
	for name, value := range cases {
		t.Run(name, func(t *testing.T) {
			if value == "" {
				t.Errorf("%s = empty string; expected a real OGR driver name", name)
			}
		})
	}
}
