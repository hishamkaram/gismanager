// Command layerSchema prints the geometry + attribute schema of every
// supported GIS file under the configured source directory. Read-only
// inspector — does not load to PostGIS or publish to GeoServer.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/hishamkaram/gismanager"
)

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("layerSchema", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
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
			slog.Error("open source", "file", file, "err", srcErr)
			continue
		}
		for index := 0; index < source.LayerCount(); index++ {
			layer := source.LayerByIndex(index)
			gLayer := gismanager.GdalLayer{Layer: &layer}
			fmt.Println(layer.Name())
			for _, f := range gLayer.GetLayerSchema() {
				fmt.Printf("\n%+v\n", *f)
			}
		}
	}
	return nil
}
