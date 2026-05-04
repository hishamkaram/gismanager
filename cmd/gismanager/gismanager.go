// Command gismanager publishes every supported GIS file under the
// configured source directory into a PostGIS datastore and registers the
// resulting tables as GeoServer feature types. See README for the YAML
// config schema and CLAUDE.md for the Docker-only dev workflow.
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
		panic(errors.New("config: --config parameter is required"))
	}
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		panic(errors.New("config: file does not exist"))
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
