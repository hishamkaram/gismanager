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
	"github.com/hishamkaram/gismanager/cmd/internal/cli"
)

func main() { os.Exit(realMain()) }

// realMain is the testable entry point; see the matching helper in
// cmd/gismanager/gismanager.go for rationale.
func realMain() int {
	ctx, cancel := cli.SignalContext(context.Background())
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		if errors.Is(err, cli.ErrVersionRequested) {
			return 0
		}
		slog.Error("layerSchema", "err", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("layerSchema", flag.ContinueOnError)
	configFile := fs.String("config", "", "Config File")
	versionFlag := fs.Bool("version", false, "print build version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *versionFlag {
		cli.PrintVersion(os.Stdout, "layerSchema")
		return cli.ErrVersionRequested
	}
	if err := cli.RequireFlag("layerSchema", "config", *configFile); err != nil {
		return err
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
