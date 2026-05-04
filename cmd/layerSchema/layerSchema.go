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
	for item, walkErr := range manager.Walk(ctx) {
		if walkErr != nil {
			slog.Error("walk", "err", walkErr)
			continue
		}
		fmt.Println(item.Layer.Name())
		for _, f := range item.Layer.GetLayerSchema() {
			fmt.Printf("\n%+v\n", *f)
		}
	}
	return nil
}
