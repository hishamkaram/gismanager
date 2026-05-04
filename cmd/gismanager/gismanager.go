package main

import (
	"context"
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
	ctx := context.Background()
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
				newLayer, postgisErr := gLayer.LayerToPostgis(targetSource, manager, true)
				if postgisErr != nil {
					logger.Error(postgisErr)
					continue
				}
				if newLayer == nil || newLayer.Layer == nil {
					continue
				}
				if err := manager.PublishGeoserverLayer(ctx, newLayer); err != nil {
					logger.Error(err)
					continue
				}
				logger.Info("published")
			}
		}
	}
}
