package gismanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupported(t *testing.T) {
	ok := isSupported(".json")
	assert.True(t, ok)
	fail := isSupported(".tiff")
	assert.False(t, fail)

}
func TestGetGISFiles(t *testing.T) {
	files, err := GetGISFiles("./testdata")
	assert.Equal(t, 3, len(files))
	assert.Nil(t, err)
	noDir, NoDirerr := GetGISFiles("./testdata/sample.gpkg")
	assert.Equal(t, 1, len(noDir))
	assert.Nil(t, NoDirerr)
	dummyFiles, dummyFileserr := GetGISFiles("./testdata_dummy/sample.gpkg")
	assert.Equal(t, 0, len(dummyFiles))
	assert.NotNil(t, dummyFileserr)
}
func TestDBIsAlive(t *testing.T) {
	manager, _ := FromConfig("./testdata/test_config.yml")
	connStr := manager.Datastore.PostgresConnectionString()
	dbErr := DBIsAlive(connStr)
	assert.Nil(t, dbErr)
	connStr = "xxxxx"
	err := DBIsAlive(connStr)
	assert.NotNil(t, err)
}
