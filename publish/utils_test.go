package publish

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSupported(t *testing.T) {
	ok := isSupported(".json")
	assert.True(t, ok)
	fail := isSupported(".tiff")
	assert.False(t, fail)
	zippedTest := isSupported(".zip")
	assert.True(t, zippedTest)

}
func TestZippedShapeFile(t *testing.T) {
	newDir, _ := os.MkdirTemp("", "zipped_shapeFile")
	unzipErr := zippedShapeFile("../testdata/faults.zip", newDir)
	assert.Nil(t, unzipErr)
	dummyErr := zippedShapeFile("../testdata/faults_ss.zip", newDir)
	assert.NotNil(t, dummyErr)
	dirErr := zippedShapeFile("../testdata/", newDir)
	assert.NotNil(t, dirErr)
}
func TestPreprocessFile(t *testing.T) {
	finalPath, preProcessErr := preprocessFile("../testdata/faults.zip", "", nil)
	assert.NotNil(t, finalPath)
	assert.NotEqual(t, "", finalPath)
	assert.Nil(t, preProcessErr)
	errFinalPath, err := preprocessFile("../testdata/dummy.zip", "", nil)
	assert.Equal(t, "", errFinalPath)
	assert.NotNil(t, err)
	emptyFinalPath, emptyErr := preprocessFile("../testdata/faults_empty.zip", "", nil)
	assert.Equal(t, "", emptyFinalPath)
	assert.NotNil(t, emptyErr)
}
func TestGetGISFiles(t *testing.T) {
	files, err := GetGISFiles("../testdata")
	// 5 supported files post-PR 5 of v1.1: faults.zip, neighborhood_names_gis.geojson,
	// nested/nyc_wi-fi_hotspot_locations.geojson, sample.gpkg, sample.kml.
	// faults_empty.zip yields no GIS files so doesn't count.
	assert.Equal(t, 5, len(files))
	assert.Nil(t, err)
	noDir, NoDirerr := GetGISFiles("../testdata/sample.gpkg")
	assert.Equal(t, 1, len(noDir))
	assert.Nil(t, NoDirerr)
	dummyFiles, dummyFileserr := GetGISFiles("./testdata_dummy/sample.gpkg")
	assert.Equal(t, 0, len(dummyFiles))
	assert.NotNil(t, dummyFileserr)
}
func TestDBIsAlive(t *testing.T) {
	// TODO(PR 5): move to utils_integration_test.go behind `//go:build
	// integration`. The happy-path branch requires a live PostGIS at
	// localhost:5432.
	t.Skip("integration-flavored: requires PostGIS; rewired in PR 5")
}
