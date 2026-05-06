package publish

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hishamkaram/gismanager/internal/errs"
	"github.com/hishamkaram/gismanager/internal/slogx"
	"github.com/hishamkaram/gismanager/internal/zipx"

	// PostgreSQL driver registered via blank import for database/sql.
	_ "github.com/lib/pq"
	yaml "gopkg.in/yaml.v3"
)

// FromConfig loads a Manager from a YAML file at the given path.
//
// Returns errors wrapping [errs.ErrConfigInvalid]; recover the underlying
// os/yaml error via [errors.As].
//
// Programmatic callers (no YAML file) should prefer [New] with [Option]
// helpers like [WithGeoserver] / [WithDatastore] / [WithSource] /
// [WithLogger]. FromConfig now delegates to New after the YAML decode.
func FromConfig(configFile string) (*Manager, error) {
	logger := slogx.Default()
	absPath, _ := filepath.Abs(configFile)
	yamlFile, readErr := os.ReadFile(absPath) //nolint:gosec // G304: configFile is the operator-supplied --config path; reading it is the documented entry point.
	if readErr != nil {
		logger.Error("read yaml", "path", absPath, "err", readErr)
		return nil, errs.NewGISError("FromConfig", absPath, errs.ErrConfigInvalid, readErr)
	}
	var decoded Manager
	if unmarshalErr := yaml.Unmarshal(yamlFile, &decoded); unmarshalErr != nil {
		logger.Error("unmarshal yaml", "path", absPath, "err", unmarshalErr)
		return nil, errs.NewGISError("FromConfig", absPath, errs.ErrConfigInvalid, unmarshalErr)
	}
	// Apply ${VAR} substitution to operator-supplied string fields
	// before validation so a YAML referencing $PG_PASSWORD doesn't
	// trip the "password is required" check.
	decoded.expandEnv()
	m, err := New(
		WithLogger(logger),
		WithGeoserver(decoded.Geoserver),
		WithDatastore(decoded.Datastore),
		WithSource(decoded.Source),
	)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		logger.Error("validate config", "path", absPath, "err", err)
		return nil, err
	}
	return m, nil
}

func isSupported(ext string) bool {
	for _, a := range supportedEXT {
		if a == ext {
			return true
		}
	}
	return false
}

// GetGISFiles walks root recursively and returns every supported GIS file
// path it finds (shapefile, GeoJSON, GeoPackage, KML, plus zipped shapefile
// bundles which are auto-extracted to a temp directory).
//
// Diagnostic logs use the project default logger from [slogx.Default]. Callers
// that want to thread a custom logger should construct a manager via [New]
// + [WithLogger] and use the manager-driven walk path (added by a
// follow-up PR); the default logger here is preserved for back-compat.
func GetGISFiles(root string) ([]string, error) {
	return getGISFiles(root, slogx.Default())
}

// getGISFiles is the logger-aware implementation. Internal callers
// (PR 3's Walk iterator, etc.) pass the manager's *slog.Logger so
// per-manager logger configuration reaches the zip-extraction inner loop.
func getGISFiles(root string, logger *slog.Logger) ([]string, error) {
	root, _ = filepath.Abs(root)
	var files []string
	fileInfo, statErr := os.Stat(root)
	if statErr != nil {
		return files, statErr
	}
	if !fileInfo.IsDir() {
		extension := strings.ToLower(filepath.Ext(fileInfo.Name()))
		if isSupported(extension) {
			finalPath, preProcessErr := preprocessFile(root, "", logger)
			if preProcessErr != nil {
				return files, preProcessErr
			}
			files = append(files, finalPath)
			return files, nil
		}
		return files, nil
	}
	dirInfo, err := os.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, entry := range dirInfo {
		subFiles, subErr := getGISFiles(path.Join(root, entry.Name()), logger)
		if subErr == nil {
			files = append(files, subFiles...)
		}
	}
	return files, nil
}

// DBIsAlive opens a database/sql connection and pings it. Returns a
// non-nil error if either step fails.
//
// Deprecated: hard-codes context.Background() for the ping; callers
// can't apply deadlines or cancellation. Use [DBIsAliveContext].
func DBIsAlive(dbType string, connectionStr string) error {
	return DBIsAliveContext(context.Background(), dbType, connectionStr)
}

// DBIsAliveContext opens a database/sql connection and pings it,
// honoring ctx for cancellation and deadlines on the ping. The connection
// is always closed before the function returns. Use this in code that
// already has a context in scope (HTTP handlers, library calls that
// thread a ctx, etc.).
func DBIsAliveContext(ctx context.Context, dbType string, connectionStr string) (err error) {
	db, dbErr := sql.Open(dbType, connectionStr)
	if dbErr != nil {
		err = dbErr
		return
	}
	defer func() { _ = db.Close() }()
	if pingErr := db.PingContext(ctx); pingErr != nil {
		err = pingErr
		return
	}
	return
}

func zippedShapeFile(zippedPath string, destPath string) (err error) {
	fileInfo, statErr := os.Stat(zippedPath)
	if statErr != nil || os.IsNotExist(statErr) {
		err = statErr
		return
	}
	if fileInfo.IsDir() {
		err = errors.New("zippedPath must be file not a directory")
		return
	}
	err = zipx.Extract(zippedPath, destPath)
	return
}

func preprocessFile(filePath string, tempPath string, logger *slog.Logger) (finalPath string, err error) {
	if logger == nil {
		logger = slogx.Default()
	}
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip":
		newDir, tempDirErr := os.MkdirTemp(tempPath, "zipped_shapeFile")
		if tempDirErr != nil {
			logger.Error("create temp dir", "parent", tempPath, "err", tempDirErr)
			err = tempDirErr
			return
		}
		logger.Debug("preprocess: created temp dir", "dir", newDir)
		unzipErr := zippedShapeFile(filePath, newDir)
		if unzipErr != nil {
			logger.Error("unzip", "src", filePath, "dest", newDir, "err", unzipErr)
			err = unzipErr
			return
		}
		files, filesErr := GetGISFiles(newDir)
		if filesErr != nil {
			logger.Error("scan extracted dir", "dir", newDir, "err", filesErr)
			err = filesErr
			return
		}
		if len(files) == 0 {
			err = errors.New("cannot find gis files")
			return
		}
		finalPath = files[0]
	default:
		finalPath = filePath
	}
	return
}
