package gismanager

import (
	"io/ioutil"

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
