package gismanager

import "regexp"

var pgRegex = regexp.MustCompile(`^\s?PG:\s?.*$`)
var supportedEXT = []string{".zip", ".json", ".geojson", ".gdb", "kml", ".shp"}

const (
	geopackageDriver  = "GPKG"
	postgreSQLDriver  = "PostgreSQL"
	shapeFileDriver   = "ESRI Shapefile"
	geoJSONDriver     = "GeoJSON"
	kmlDriver         = "KML"
	openFileGDBDriver = "OpenFileGDB"
	esriJSONDriver    = "ESRIJSON"
)
