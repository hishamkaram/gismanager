package publish

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/v2/errs"
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

// Manager is the top-level configuration loaded from YAML.
type Manager struct {
	Geoserver GeoserverConfig `yaml:"geoserver"`
	Datastore DatastoreConfig `yaml:"datastore"`
	Source    SourceConfig    `yaml:"source"`
	logger    *slog.Logger

	// publishConcurrency caps how many GeoServer feature-type
	// creations [PublishAll] dispatches in parallel. Zero (the
	// zero-value) means "use [defaultPublishConcurrency]"; an explicit
	// positive value (set via [WithPublishConcurrency]) overrides.
	// One means strictly serial publish — useful for diagnostic
	// runs against finicky GeoServer instances.
	publishConcurrency int
}

// Validate checks that every required field on a Manager is set
// to a non-zero value and reports the missing fields in a single error.
// The error wraps [errs.ErrConfigInvalid]; recover the field list via
// [errors.As] into [*errs.GISError] and inspect Cause.Error().
//
// Validate covers the publish-pipeline contract — every field needed
// for the walk → PostGIS-load → GeoServer-publish flow. Programmatic
// callers using only a subset of the manager's surface (e.g.
// OpenSource for ad-hoc reads) can skip Validate entirely; only
// [FromConfig] calls it automatically, since YAML callers signal
// "I intend a full publish setup" by writing the YAML in the first
// place.
func (c *Manager) Validate() error {
	var problems []string
	if c.Geoserver.ServerURL == "" {
		problems = append(problems, "geoserver.url is required")
	}
	if c.Geoserver.WorkspaceName == "" {
		problems = append(problems, "geoserver.workspace is required")
	}
	if c.Geoserver.Username == "" {
		problems = append(problems, "geoserver.username is required")
	}
	if c.Geoserver.Password == "" {
		problems = append(problems, "geoserver.password is required")
	}
	if c.Datastore.Host == "" {
		problems = append(problems, "datastore.host is required")
	}
	if c.Datastore.Port == 0 {
		problems = append(problems, "datastore.port is required (and non-zero)")
	}
	if c.Datastore.DBName == "" {
		problems = append(problems, "datastore.database is required")
	}
	if c.Datastore.DBUser == "" {
		problems = append(problems, "datastore.username is required")
	}
	if c.Datastore.DBPass == "" {
		problems = append(problems, "datastore.password is required")
	}
	if c.Datastore.Name == "" {
		problems = append(problems, "datastore.name is required")
	}
	if c.Source.Path == "" {
		problems = append(problems, "source.path is required")
	}
	if len(problems) > 0 {
		return errs.NewGISError("Validate", "", errs.ErrConfigInvalid,
			errors.New(strings.Join(problems, "; ")))
	}
	return nil
}

// expandEnv applies os.ExpandEnv (`$VAR` and `${VAR}` substitution) to
// every operator-supplied string field on the manager config. Used
// internally by [FromConfig] after YAML decode and before validation,
// so YAML files can reference secrets without inlining them:
//
//	geoserver:
//	  password: ${GEOSERVER_PASSWORD}
//	datastore:
//	  password: ${PG_PASSWORD}
//
// Variables that aren't set in the environment resolve to empty strings
// (matching os.ExpandEnv); Validate then rejects the empty result with a
// useful field-name error message.
//
// Datastore.Port (uint) is not expanded — port numbers should come from
// config, not env vars; if a user truly needs a parameterized port they
// should set it via the [WithDatastore] option in code.
func (c *Manager) expandEnv() {
	c.Geoserver.ServerURL = os.ExpandEnv(c.Geoserver.ServerURL)
	c.Geoserver.WorkspaceName = os.ExpandEnv(c.Geoserver.WorkspaceName)
	c.Geoserver.Username = os.ExpandEnv(c.Geoserver.Username)
	c.Geoserver.Password = os.ExpandEnv(c.Geoserver.Password)
	c.Datastore.Host = os.ExpandEnv(c.Datastore.Host)
	c.Datastore.DBName = os.ExpandEnv(c.Datastore.DBName)
	c.Datastore.DBUser = os.ExpandEnv(c.Datastore.DBUser)
	c.Datastore.DBPass = os.ExpandEnv(c.Datastore.DBPass)
	c.Datastore.Name = os.ExpandEnv(c.Datastore.Name)
	c.Source.Path = os.ExpandEnv(c.Source.Path)
}

// GetGeoserverCatalog returns a GeoServer v2 client configured against the
// manager's GeoServer endpoint. Each call constructs a fresh client; the
// underlying HTTP transport is the stdlib default. Returns an error if the
// server URL is malformed.
func (manager *Manager) GetGeoserverCatalog() (*geoserver.Client, error) {
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
// Errors wrap [errs.ErrUnsupportedFormat] (driver lookup failed) or
// [errs.ErrInvalidDatasource] (driver matched but Open returned !ok). Match via
// [errors.Is]; recover details via [errors.As] into *errs.GISError.
func (manager *Manager) OpenSource(_ context.Context, path string, access int) (*gdal.DataSource, error) {
	driver, err := manager.GetDriver(path)
	if err != nil {
		// GetDriver already logged + wrapped; just return.
		return nil, err
	}
	targetSource, ok := driver.Open(path, access)
	if !ok {
		manager.logger.Error("open source", "path", path, "access", access)
		return nil, errs.NewGISError("OpenSource", path, errs.ErrInvalidDatasource, nil)
	}
	return &targetSource, nil
}

// GetDriver returns the OGR driver appropriate for the given path or
// connection string. PostgreSQL connection strings (matching pgRegex) get
// the PostgreSQL driver; everything else dispatches on file extension.
//
// On unsupported extensions, returns an error wrapping
// [errs.ErrUnsupportedFormat] (match via [errors.Is]).
func (manager *Manager) GetDriver(path string) (driver gdal.OGRDriver, err error) {
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
		case ".parquet":
			driver = gdal.OGRDriverByName(parquetDriver)
		default:
			err = errs.NewGISError("GetDriver", path, errs.ErrUnsupportedFormat, nil)
			manager.logger.Error("get driver", "path", path, "err", err)
		}
	}
	return
}
