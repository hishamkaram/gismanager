package gismanager

import "regexp"

var pgRegex = regexp.MustCompile(`^\s?PG:\s?.*$`)

// supportedEXT is the closed set of file extensions GetGISFiles will
// recurse into. Each entry MUST start with a dot — filepath.Ext()
// always returns the dot, so a bare "kml" entry never matches and the
// extension is silently unsupported. (That was the pre-v1.1 bug here:
// "kml" was missing the leading dot and KML files were silently
// dropped from directory walks. Fixed in PR 5 of the v1.1 series.)
var supportedEXT = []string{".zip", ".json", ".gpkg", ".geojson", ".gdb", ".kml", ".shp", ".parquet"}

const (
	geopackageDriver = "GPKG"
	postgreSQLDriver = "PostgreSQL"
	shapeFileDriver  = "ESRI Shapefile"
	geoJSONDriver    = "GeoJSON"
	kmlDriver        = "KML"
	// parquetDriver is the GDAL OGR driver name for Apache Parquet
	// files (with or without GeoParquet metadata). Available in the
	// `ubuntu-full` GDAL image variant from v1.4 onward; see
	// docs/conversions.md and the Dockerfile comment block. Operators
	// running on `ubuntu-small` will see GetDriver succeed but
	// downstream operations will fail with a "driver not registered"
	// GDAL error — switching to ubuntu-full resolves it.
	parquetDriver = "Parquet"
)
