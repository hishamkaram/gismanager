package gismanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestGetFeatures(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := GdalLayer{
		Layer: &layer,
	}
	count, _ := layer.FeatureCount(true)
	features := gLayer.GetFeatures()
	assert.NotNil(t, features)
	assert.Equal(t, count, len(features))

}
func TestGetLayerSchema(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := GdalLayer{
		Layer: &layer,
	}
	fields := gLayer.GetLayerSchema()
	assert.NotNil(t, fields)
	assert.Equal(t, 9, len(fields))

}

type ManagerLayerSuite struct {
	suite.Suite
	manager *ManagerConfig
}

func (suite *ManagerLayerSuite) SetupSuite() {
	manager, _ := FromConfig("./testdata/test_config.yml")
	suite.manager = manager
}

func (suite *ManagerLayerSuite) TestLayerOperations() {
	manager := suite.manager
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := GdalLayer{
		Layer: &layer,
	}
	dummyGLayer := GdalLayer{}
	targetSource, _ := manager.OpenSource(manager.Datastore.BuildConnectionString(), 1)
	nilStore, nilStoreErr := gLayer.LayerToPostgis(nil, manager, true)
	assert.Nil(suite.T(), nilStore)
	assert.NotNil(suite.T(), nilStoreErr)
	nilLayer, nilLayerErr := dummyGLayer.LayerToPostgis(targetSource, manager, true)
	assert.Nil(suite.T(), nilLayer)
	assert.NotNil(suite.T(), nilLayerErr)
	newLayer, err := gLayer.LayerToPostgis(targetSource, manager, true)
	assert.NotNil(suite.T(), newLayer)
	assert.Nil(suite.T(), err)
	ok, publishErr := manager.PublishGeoserverLayer(newLayer)
	assert.True(suite.T(), ok)
	assert.Nil(suite.T(), publishErr)
}

func (suite *ManagerLayerSuite) TearDownSuite() {
	catalog := suite.manager.GetGeoserverCatalog()
	deleted, err := catalog.DeleteWorkspace(suite.manager.Geoserver.WorkspaceName, true)
	assert.True(suite.T(), deleted)
	assert.Nil(suite.T(), err)
}
func TestManagerLayerSuite(t *testing.T) {
	suite.Run(t, new(ManagerLayerSuite))
}
