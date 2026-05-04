// Package gismanager publishes GIS data files (shapefile, GeoJSON,
// GeoPackage, KML) to PostGIS, then exposes the resulting tables as
// GeoServer feature types via the geoserver/v2 REST client.
package gismanager

import "fmt"

// DatastoreConfig holds the PostGIS connection parameters that GIS Manager
// uses both as the GDAL OGR target (for loading) and as the GeoServer
// datastore to publish.
type DatastoreConfig struct {
	Host   string `yaml:"host"`
	Port   uint   `yaml:"port"`
	DBName string `yaml:"database"`
	DBUser string `yaml:"username"`
	DBPass string `yaml:"password"`
	Name   string `yaml:"name"`
}

// BuildConnectionString returns the GDAL-style OGR connection string for
// PostgreSQL/PostGIS (the leading "PG:" prefix tells OGR which driver to
// use).
func (ds *DatastoreConfig) BuildConnectionString() string {
	return fmt.Sprintf("PG: host=%s port=%d dbname=%s user=%s password=%s", ds.Host, ds.Port, ds.DBName, ds.DBUser, ds.DBPass)
}

// PostgresConnectionString returns a database/sql-compatible PostgreSQL
// connection URL (driver "postgres" via lib/pq).
func (ds *DatastoreConfig) PostgresConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", ds.DBUser, ds.DBPass, ds.Host, ds.Port, ds.DBName)
}
