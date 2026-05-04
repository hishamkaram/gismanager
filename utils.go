package gismanager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/hishamkaram/gismanager/internal/zipx"

	// PostgreSQL driver registered via blank import for database/sql.
	_ "github.com/lib/pq"
	yaml "gopkg.in/yaml.v3"
)

// FromConfig loads a ManagerConfig from a YAML file at the given path.
func FromConfig(configFile string) (config *ManagerConfig, err error) {
	gpkgConfig := ManagerConfig{}
	gpkgConfig.logger = GetLogger()
	absPath, _ := filepath.Abs(configFile)
	yamlFile, err := os.ReadFile(absPath)
	if err != nil {
		gpkgConfig.logger.Errorf("yamlFile.Get err   %v ", err)
		return
	}
	err = yaml.Unmarshal(yamlFile, &gpkgConfig)
	if err != nil {
		gpkgConfig.logger.Errorf("Unmarshal: %v", err)
		return
	}
	config = &gpkgConfig
	return
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
func GetGISFiles(root string) ([]string, error) {
	root, _ = filepath.Abs(root)
	var files []string
	fileInfo, statErr := os.Stat(root)
	if statErr != nil {
		return files, statErr
	}
	if !fileInfo.IsDir() {
		extension := strings.ToLower(filepath.Ext(fileInfo.Name()))
		if isSupported(extension) {
			finalPath, preProcessErr := preprocessFile(root, "")
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
		subFiles, subErr := GetGISFiles(path.Join(root, entry.Name()))
		if subErr == nil {
			files = append(files, subFiles...)
		}
	}
	return files, nil
}

// DBIsAlive opens a database/sql connection and pings it. Returns a
// non-nil error if either step fails.
//
// TODO(PR 4): take a context.Context argument so callers can apply
// deadlines / cancellation.
func DBIsAlive(dbType string, connectionStr string) (err error) {
	db, dbErr := sql.Open(dbType, connectionStr)
	if dbErr != nil {
		err = dbErr
		return
	}
	if pingErr := db.PingContext(context.Background()); pingErr != nil {
		_ = db.Close()
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

func preprocessFile(filePath string, tempPath string) (finalPath string, err error) {
	logger := GetLogger()
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip":
		newDir, tempDirErr := os.MkdirTemp(tempPath, "zipped_shapeFile")
		fmt.Println(newDir)
		if tempDirErr != nil {
			logger.Error(tempDirErr)
			err = tempDirErr
			return
		}
		unzipErr := zippedShapeFile(filePath, newDir)
		if unzipErr != nil {
			logger.Error(unzipErr)
			err = unzipErr
			return
		}
		files, filesErr := GetGISFiles(newDir)
		if filesErr != nil {
			logger.Error(filesErr)
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
