package gismanager

import (
	"testing"

	"github.com/lukeroth/gdal"
	"github.com/stretchr/testify/assert"
)

func TestFromConfig(t *testing.T) {
	manager, confErr := FromConfig("./testdata/test_config.yml")
	assert.Nil(t, confErr)
	assert.NotNil(t, manager)
	expected := ManagerConfig{
		Geoserver: geoserver{WorkspaceName: "golang", Username: "admin", Password: "geoserver", ServerURL: "http://localhost:8080/geoserver"},
		Datastore: datastore{Host: "localhost", Port: 5432, DBName: "gis", DBUser: "golang", DBPass: "golang", Name: "gismanager_data"},
		Source:    source{Path: "./testdata"},
		logger:    manager.logger,
	}
	assert.Equal(t, expected.Geoserver, manager.Geoserver)
	// assert.Equal(t, expected.Datastore, manager.Datastore)
	assert.Equal(t, expected.Source, manager.Source)
	assert.Equal(t, expected.logger, manager.logger)
	nilManager, nilConfErr := FromConfig("./testdata/test_configs.yml")
	assert.NotNil(t, nilConfErr)
	assert.Nil(t, nilManager)
	errManager, errConfErr := FromConfig("./testdata/test_config_err.yml")
	assert.NotNil(t, errConfErr)
	assert.Nil(t, errManager)
}
func TestOpenSource(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	datasource, ok := manager.OpenSource("./testdata/sample.gpkg", 0)
	assert.True(t, ok)
	assert.NotNil(t, datasource)
	nilDatasource, fail := manager.OpenSource("./testdata/sample_dummy.xml", 0)
	assert.False(t, fail)
	assert.Nil(t, nilDatasource)

}
func TestGetGeoserverCatalog(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	catalog := manager.GetGeoserverCatalog()
	assert.NotNil(t, catalog)

}
func TestGetDriver(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	gpkgDriver, gpkgErr := manager.GetDriver("./testdata/sample.gpkg")
	assert.Nil(t, gpkgErr)
	assert.NotNil(t, gpkgDriver)
	assert.Equal(t, gdal.OGRDriverByName(geopackageDriver), gpkgDriver)
	jsonDriver, jsonErr := manager.GetDriver("./testdata/neighborhood_names_gis.geojson")
	assert.Nil(t, jsonErr)
	assert.NotNil(t, jsonDriver)
	assert.Equal(t, gdal.OGRDriverByName(geoJSONDriver), jsonDriver)
	zipDriver, zipErr := manager.GetDriver("./testdata/shapeFile.zip")
	assert.Nil(t, zipErr)
	assert.NotNil(t, zipDriver)
	assert.Equal(t, gdal.OGRDriverByName(shapeFileDriver), zipDriver)
	KMLDriver, KMLErr := manager.GetDriver("./testdata/layer.kml")
	assert.Nil(t, KMLErr)
	assert.NotNil(t, KMLDriver)
	assert.Equal(t, gdal.OGRDriverByName(kmlDriver), KMLDriver)
	pgDriver, pgErr := manager.GetDriver("PG: host=localhost port=5432 dbname=gis user=golang password=golang")
	assert.Nil(t, pgErr)
	assert.NotNil(t, pgDriver)
	assert.Equal(t, gdal.OGRDriverByName(postgreSQLDriver), pgDriver)
	tiffDriver, tiffErr := manager.GetDriver("./testdata/layer.tiff")
	assert.NotNil(t, tiffErr)
	assert.Equal(t, gdal.OGRDriver{}, tiffDriver)

}
