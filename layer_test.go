package gismanager

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFeatures(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(context.Background(), files[0], 1)
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
	source, _ := manager.OpenSource(context.Background(), files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := GdalLayer{
		Layer: &layer,
	}
	fields := gLayer.GetLayerSchema()
	assert.NotNil(t, fields)
	assert.True(t, len(fields) > 0)
}
