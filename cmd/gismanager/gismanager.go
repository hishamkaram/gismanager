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
	return manager.PublishAll(ctx)
}
