package gismanager

import (
	"errors"
	"path/filepath"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/lukeroth/gdal"
	"github.com/sirupsen/logrus"
)

// GeoserverConfig holds the GeoServer endpoint and credentials. WorkspaceName
// is the workspace gismanager publishes layers into; it is created on first
// use if missing.
type GeoserverConfig struct {
	WorkspaceName string `yaml:"workspace"`
	ServerURL     string `yaml:"url"`
	Username      string `yaml:"username"`
	Password      string `yaml:"password"`
}

// SourceConfig points at the directory (or single file) gismanager scans for
// supported GIS data files.
type SourceConfig struct {
	Path string `yaml:"path"`
}

// ManagerConfig is the top-level configuration loaded from YAML.
type ManagerConfig struct {
	Geoserver GeoserverConfig `yaml:"geoserver"`
	Datastore DatastoreConfig `yaml:"datastore"`
	Source    SourceConfig    `yaml:"source"`
	logger    *logrus.Logger
}

// GetGeoserverCatalog returns a GeoServer v2 client configured against the
// manager's GeoServer endpoint. Each call constructs a fresh client; the
// underlying HTTP transport is the stdlib default. Returns an error if the
// server URL is malformed.
func (manager *ManagerConfig) GetGeoserverCatalog() (*geoserver.Client, error) {
	return geoserver.New(
		manager.Geoserver.ServerURL,
		geoserver.WithBasicAuth(manager.Geoserver.Username, manager.Geoserver.Password),
	)
}

// OpenSource opens a GDAL data source at the given path. access is the GDAL
// permission flag (0 read-only, 1 read-write).
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

// GetDriver returns the OGR driver appropriate for the given path or
// connection string. PostgreSQL connection strings (matching pgRegex) get
// the PostgreSQL driver; everything else dispatches on file extension.
func (manager *ManagerConfig) GetDriver(path string) (driver gdal.OGRDriver, err error) {
	if pgRegex.MatchString(path) {
		driver = gdal.OGRDriverByName(postgreSQLDriver)
	} else {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".gpkg":
			driver = gdal.OGRDriverByName(geopackageDriver)
		case ".shp", ".zip":
			driver = gdal.OGRDriverByName(shapeFileDriver)
		case ".json", ".geojson":
			driver = gdal.OGRDriverByName(geoJSONDriver)
		case ".kml":
			driver = gdal.OGRDriverByName(kmlDriver)
		default:
			err = errors.New("can't find the proper driver")
			manager.logger.Error(err)
		}
	}
	return
}
