package main

import (
	"errors"
	"flag"
	"os"

	"github.com/hishamkaram/gismanager"
)

func main() {
	logger := gismanager.GetLogger()
	configFile := flag.String("config", "", "Config File")
	flag.Parse()
	if *configFile == "" {
		panic(errors.New("config 'Parameter required'"))
	}
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		panic(errors.New("Config File Doesn't exist"))
	}
	manager, confErr := gismanager.FromConfig(*configFile)
	if confErr != nil {
		panic(confErr)
	}
	files, _ := gismanager.GetGISFiles(manager.Source.Path)
	for _, file := range files {
		source, ok := manager.OpenSource(file, 0)
		targetSource, targetOK := manager.OpenSource(manager.Datastore.BuildConnectionString(), 1)
		if ok && targetOK {
			for index := 0; index < source.LayerCount(); index++ {
				layer := source.LayerByIndex(index)
				gLayer := gismanager.GdalLayer{
					Layer: &layer,
				}
				if newLayer, postgisErr := gLayer.LayerToPostgis(targetSource, manager, true); newLayer.Layer != nil || postgisErr != nil {
					ok, pubErr := manager.PublishGeoserverLayer(newLayer)
					if pubErr != nil {
						logger.Error(pubErr)
					}
					if !ok {
						logger.Error("Failed to Publish")
					} else {
						logger.Info("published")
					}
				}

			}
		}
	}
}
