package gismanager

import (
	"io/ioutil"
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
	newDir, _ := ioutil.TempDir("", "zipped_shapeFile")
	unzipErr := zippedShapeFile("./testdata/faults.zip", newDir)
	assert.Nil(t, unzipErr)
	dummyErr := zippedShapeFile("./testdata/faults_ss.zip", newDir)
	assert.NotNil(t, dummyErr)
	dirErr := zippedShapeFile("./testdata/", newDir)
	assert.NotNil(t, dirErr)
}
func TestPreprocessFile(t *testing.T) {
	finalPath, preProcessErr := preprocessFile("./testdata/faults.zip", "")
	assert.NotNil(t, finalPath)
	assert.NotEqual(t, "", finalPath)
	assert.Nil(t, preProcessErr)
	errFinalPath, err := preprocessFile("./testdata/dummy.zip", "")
	assert.Equal(t, "", errFinalPath)
	assert.NotNil(t, err)
	emptyFinalPath, emptyErr := preprocessFile("./testdata/faults_empty.zip", "")
	assert.Equal(t, "", emptyFinalPath)
	assert.NotNil(t, emptyErr)
}
func TestGetGISFiles(t *testing.T) {
	files, err := GetGISFiles("./testdata")
	assert.Equal(t, 4, len(files))
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
	dbErr := DBIsAlive("postgres", connStr)
	assert.Nil(t, dbErr)
	connStr = "mysql://"
	err := DBIsAlive("mysql", connStr)
	assert.NotNil(t, err)
	connStr = "postgresql://dummy:dummy@localhost:5438/dummy_db?sslmode=disable"
	pingErr := DBIsAlive("postgres", connStr)
	assert.NotNil(t, pingErr)
}
