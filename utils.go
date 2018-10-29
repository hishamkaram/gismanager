package gismanager

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"

	//postgres Driver
	_ "github.com/lib/pq"
	yaml "gopkg.in/yaml.v2"
)

//FromConfig load GIS Manager config from yaml file
func FromConfig(configFile string) (config *ManagerConfig, err error) {
	gpkgConfig := ManagerConfig{}
	gpkgConfig.logger = GetLogger()
	path, _ := filepath.Abs(configFile)
	yamlFile, err := ioutil.ReadFile(path)
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

//GetGISFiles retrun List of All GIS Files in this path
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
	dirInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, file := range dirInfo {
		subFiles, subErr := GetGISFiles(path.Join(root, file.Name()))
		if subErr == nil {
			files = append(files, subFiles...)
		}
	}
	return files, nil
}

//DBIsAlive check if database alive
func DBIsAlive(dbType string, connectionStr string) (err error) {
	db, dbErr := sql.Open(dbType, connectionStr)
	if dbErr != nil {
		err = dbErr
		return
	}
	if pingErr := db.Ping(); pingErr != nil {
		db.Close()
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
	err = archiver.Zip.Open(zippedPath, destPath)
	return
}
func preprocessFile(filePath string, tempPath string) (finalPath string, err error) {
	logger := GetLogger()
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".zip":
		newDir, tempDirErr := ioutil.TempDir(tempPath, "zipped_shapeFile")
		fmt.Println(newDir)
		if tempDirErr != nil {
			logger.Error(tempDirErr)
			err = tempDirErr
			break
		}
		unzipErr := zippedShapeFile(filePath, newDir)
		if unzipErr != nil {
			logger.Error(unzipErr)
			err = unzipErr
			break
		}
		files, filesErr := GetGISFiles(newDir)
		if filesErr != nil {
			logger.Error(filesErr)
			err = filesErr
			break
		}
		if len(files) == 0 {
			err = errors.New("cannot find gis files")
			break
		}
		finalPath = files[0]
		break
	default:
		finalPath = filePath
	}
	return
}
