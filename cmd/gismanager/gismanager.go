// Command gismanager publishes every supported GIS file under the
// configured source directory into a PostGIS datastore and registers the
// resulting tables as GeoServer feature types. See README for the YAML
// config schema and CLAUDE.md for the Docker-only dev workflow.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"os"

	"github.com/hishamkaram/gismanager"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("gismanager", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	logger := gismanager.GetLogger()
	configFile := flag.String("config", "", "Config File")
	flag.Parse()
	if *configFile == "" {
		return errors.New("config: --config parameter is required")
	}
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		return errors.New("config: file does not exist")
	}
	manager, err := gismanager.FromConfig(*configFile)
	if err != nil {
		return err
	}
	files, _ := gismanager.GetGISFiles(manager.Source.Path)
	for _, file := range files {
		source, srcErr := manager.OpenSource(ctx, file, 0)
		if srcErr != nil {
			logger.Error("open source", "file", file, "err", srcErr)
			continue
		}
		targetSource, dsErr := manager.OpenSource(ctx, manager.Datastore.BuildConnectionString(), 1)
		if dsErr != nil {
			logger.Error("open postgis target", "err", dsErr)
			continue
		}
		for index := 0; index < source.LayerCount(); index++ {
			layer := source.LayerByIndex(index)
			gLayer := gismanager.GdalLayer{Layer: &layer}
			newLayer, postgisErr := gLayer.LayerToPostgis(targetSource, manager, true)
			if postgisErr != nil {
				logger.Error("load to postgis", "file", file, "err", postgisErr)
				continue
			}
			if newLayer == nil || newLayer.Layer == nil {
				continue
			}
			if err := manager.PublishGeoserverLayer(ctx, newLayer); err != nil {
				logger.Error("publish", "file", file, "err", err)
				continue
			}
			logger.Info("published", "file", file, "layer", newLayer.Name())
		}
	}
	return nil
}
