package gismanager

import "regexp"

var pgRegex = regexp.MustCompile(`^\s?PG:\s?.*$`)

// supportedEXT is the closed set of file extensions GetGISFiles will
// recurse into. Each entry MUST start with a dot — filepath.Ext()
// always returns the dot, so a bare "kml" entry never matches and the
// extension is silently unsupported. (That was the pre-v1.1 bug here:
// "kml" was missing the leading dot and KML files were silently
// dropped from directory walks. Fixed in PR 5 of the v1.1 series.)
var supportedEXT = []string{".zip", ".json", ".gpkg", ".geojson", ".gdb", ".kml", ".shp"}

const (
	geopackageDriver = "GPKG"
	postgreSQLDriver = "PostgreSQL"
	shapeFileDriver  = "ESRI Shapefile"
	geoJSONDriver    = "GeoJSON"
	kmlDriver        = "KML"
)
