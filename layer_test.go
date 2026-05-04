package gismanager

import (
	"context"
	"testing"

	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
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
	assert.True(t, len(fields) > 0)

}

type ManagerLayerSuite struct {
	suite.Suite
	manager *ManagerConfig
}

func (suite *ManagerLayerSuite) SetupSuite() {
	// TODO(PR 5): move this whole suite to layer_integration_test.go behind
	// `//go:build integration`. The suite end-to-end-tests PostGIS + GeoServer
	// publish; PR 1 keeps the structure and skips at the entry point so the
	// unit job stays green without a live compose stack.
	suite.T().Skip("integration-flavored: requires PostGIS + GeoServer; rewired in PR 5")
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
	publishErr := manager.PublishGeoserverLayer(context.Background(), newLayer)
	assert.Nil(suite.T(), publishErr)
}

func (suite *ManagerLayerSuite) TearDownSuite() {
	if suite.manager == nil {
		return // SetupSuite skipped; nothing to tear down.
	}
	catalog, err := suite.manager.GetGeoserverCatalog()
	assert.Nil(suite.T(), err)
	deleteErr := catalog.Workspaces.Delete(
		context.Background(),
		suite.manager.Geoserver.WorkspaceName,
		workspaces.DeleteOptions{Recurse: true},
	)
	assert.Nil(suite.T(), deleteErr)
}

func TestManagerLayerSuite(t *testing.T) {
	suite.Run(t, new(ManagerLayerSuite))
}
