package publish

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFeatures_Iterator tests the v2 Features() iterator that replaced
// the v1 GetFeatures() handle-leaking slice form. The new API yields
// each feature once and destroys the C handle as iteration advances,
// so callers don't need to remember to free.
func TestFeatures_Iterator(t *testing.T) {
	manager, _ := FromConfig("../testdata/test_config.yml")
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(context.Background(), files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := Layer{Layer: &layer}

	count, _ := layer.FeatureCount(true)

	seen := 0
	for range gLayer.Features(context.Background()) {
		// Iterate; the iterator destroys each feature as it advances.
		// Counting is the contract we lock in here.
		seen++
	}

	assert.Equal(t, count, seen, "Features iterator should yield every layer feature once")
}

func TestGetLayerSchema(t *testing.T) {
	manager, _ := FromConfig("../testdata/test_config.yml")
	files, _ := GetGISFiles(manager.Source.Path)
	source, _ := manager.OpenSource(context.Background(), files[0], 1)
	layer := source.LayerByIndex(0)
	gLayer := Layer{
		Layer: &layer,
	}
	fields := gLayer.GetLayerSchema()
	assert.NotNil(t, fields)
	assert.True(t, len(fields) > 0)
}
