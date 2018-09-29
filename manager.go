package gismanager

import (
	"errors"
	"path/filepath"
	"strings"

	gsconfig "github.com/hishamkaram/geoserver"
	"github.com/lukeroth/gdal"
	"github.com/sirupsen/logrus"
)

type geoserver struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
}

type source struct {
	Path string `yaml:"path"`
}

//ManagerConfig is the configuration Object
type ManagerConfig struct {
	Geoserver geoserver `yaml:"geoserver"`
	Datastore datastore `yaml:"datastore"`
	Source    source    `yaml:"source"`
	logger    *logrus.Logger
}

//GetGeoserverCatalog publish return geoserver Catalog
func (manager *ManagerConfig) GetGeoserverCatalog() *gsconfig.GeoServer {
	gsCatalog := gsconfig.GetCatalog(manager.Geoserver.ServerURL, manager.Geoserver.Username, manager.Geoserver.Password)
	return gsCatalog
}

//PublishGeoserverLayer publish Layer to postgis
func (manager *ManagerConfig) PublishGeoserverLayer(layer *GdalLayer) (ok bool, err error) {
	if layer != nil {
		gsCatalog := manager.GetGeoserverCatalog()
		exists, storeErr := gsCatalog.DatastoreExists(manager.Geoserver.WorkspaceName, manager.Datastore.Name, true)
		if storeErr != nil {
			err = storeErr
		}
		if exists {
			ok, err = gsCatalog.PublishPostgisLayer(manager.Geoserver.WorkspaceName, manager.Datastore.Name, layer.Name(), layer.Name())
		}
	}
	return
}

//OpenSource open Datasource
func (manager *ManagerConfig) OpenSource(path string, access int) (source gdal.DataSource, ok bool) {
	driver, err := manager.GetDriver(path)
	if err != nil {
		panic(err)
	}
	source, ok = driver.Open(path, access)
	return
}

//GetDriver open Datasource
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
		case ".json":
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
