package gismanager

import (
	"context"
	"log/slog"
	"path/filepath"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/lukeroth/gdal"
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
	logger    *slog.Logger
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
// permission flag (0 read-only, 1 read-write). ctx is reserved for future
// cancellation support — GDAL bindings don't propagate it today, but
// callers should still pass a real context so a downstream-aware version
// is a non-breaking swap.
//
// Errors wrap [ErrUnsupportedFormat] (driver lookup failed) or
// [ErrInvalidDatasource] (driver matched but Open returned !ok). Match via
// [errors.Is]; recover details via [errors.As] into *GISError.
func (manager *ManagerConfig) OpenSource(_ context.Context, path string, access int) (*gdal.DataSource, error) {
	driver, err := manager.GetDriver(path)
	if err != nil {
		// GetDriver already logged + wrapped; just return.
		return nil, err
	}
	targetSource, ok := driver.Open(path, access)
	if !ok {
		manager.logger.Error("open source", "path", path, "access", access)
		return nil, newGISError("OpenSource", path, ErrInvalidDatasource, nil)
	}
	return &targetSource, nil
}

// GetDriver returns the OGR driver appropriate for the given path or
// connection string. PostgreSQL connection strings (matching pgRegex) get
// the PostgreSQL driver; everything else dispatches on file extension.
//
// On unsupported extensions, returns an error wrapping
// [ErrUnsupportedFormat] (match via [errors.Is]).
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
			err = newGISError("GetDriver", path, ErrUnsupportedFormat, nil)
			manager.logger.Error("get driver", "path", path, "err", err)
		}
	}
	return
}
