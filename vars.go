package gismanager

import "regexp"

var pgRegex = regexp.MustCompile(`^\s?PG:\s?.*$`)

const (
	geopackageDriver  = "GPKG"
	postgreSQLDriver  = "PostgreSQL"
	shapeFileDriver   = "ESRI Shapefile"
	geoJSONDriver     = "GeoJSON"
	kmlDriver         = "KML"
	openFileGDBDriver = "OpenFileGDB"
	esriJSONDriver    = "ESRIJSON"
)
