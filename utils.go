package gismanager

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

//FromConfig load geoserver config from yaml file
func FromConfig(configFile string) (config *ManagerConfig, err error) {
	gpkgConfig := ManagerConfig{}
	gpkgConfig.logger = GetLogger()
	yamlFile, err := ioutil.ReadFile(configFile)
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
	var files []string
	fileInfo, statErr := os.Stat(root)
	if statErr != nil {
		return files, statErr
	}
	if !fileInfo.IsDir() {
		files = append(files, fileInfo.Name())
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
				files = append(files, file.Name())
			}
		}
	}

	return files, nil
}
