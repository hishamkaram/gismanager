package gismanager

import (
	"errors"
	"path/filepath"
	"strings"

	gsconfig "github.com/hishamkaram/geoserver"
	"github.com/lukeroth/gdal"
	"github.com/sirupsen/logrus"
)

//GeoserverConfig geoserver configuration
type GeoserverConfig struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
}

//SourceConfig Data Source/Dir configuration
type SourceConfig struct {
	Path string `yaml:"path"`
}

//ManagerConfig is the configuration Object
type ManagerConfig struct {
	Geoserver GeoserverConfig `yaml:"geoserver"`
	Datastore DatastoreConfig `yaml:"datastore"`
	Source    SourceConfig    `yaml:"source"`
	logger    *logrus.Logger
}

//GetGeoserverCatalog return geoserver Catalog instance to deal with geoserver
func (manager *ManagerConfig) GetGeoserverCatalog() *gsconfig.GeoServer {
	gsCatalog := gsconfig.GetCatalog(manager.Geoserver.ServerURL, manager.Geoserver.Username, manager.Geoserver.Password)
	return gsCatalog
}

//OpenSource open data source from a given Path and access permission 0/1
func (manager *ManagerConfig) OpenSource(path string, access int) (source *gdal.DataSource, ok bool) {
	driver, err := manager.GetDriver(path)
	if err != nil {
		manager.logger.Error(err)
		ok = false
		return
	}
	targetSource, success := driver.Open(path, access)
	source = &targetSource
	ok = success
	return
}

//GetDriver return the proper driver based on file path/database connection
func (manager *ManagerConfig) GetDriver(path string) (driver gdal.OGRDriver, err error) {
	if pgRegex.MatchString(path) {
		driver = gdal.OGRDriverByName(postgreSQLDriver)
	} else {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".gpkg":
			driver = gdal.OGRDriverByName(geopackageDriver)
			break
		case ".shp", ".zip":
			driver = gdal.OGRDriverByName(shapeFileDriver)
			break
		case ".json", ".geojson":
			driver = gdal.OGRDriverByName(geoJSONDriver)
			break
		case ".kml":
			driver = gdal.OGRDriverByName(kmlDriver)
			break
		default:
			err = errors.New("Can't Find the Proper Driver")
			manager.logger.Error(err)
		}
	}
	return
}
