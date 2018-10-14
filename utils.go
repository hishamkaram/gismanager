package gismanager

import (
	"database/sql"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
	yaml "gopkg.in/yaml.v2"
)

//FromConfig load geoserver config from yaml file
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
		files = append(files, path.Join(root, fileInfo.Name()))
		return files, nil
	}
	dirInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}
	for _, file := range dirInfo {
		if file.IsDir() {
			subFiles, subErr := GetGISFiles(path.Join(root, file.Name()))
			if subErr == nil {
				files = append(files, subFiles...)
			}
		} else {
			extension := strings.ToLower(filepath.Ext(file.Name()))
			if isSupported(extension) {
				files = append(files, path.Join(root, file.Name()))
			}
		}
	}

	return files, nil
}

//DBIsAlive check if database alive
func DBIsAlive(connectionStr string) (err error) {
	db, dbErr := sql.Open("postgres", connectionStr)
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
