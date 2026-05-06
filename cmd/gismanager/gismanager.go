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

	"github.com/hishamkaram/gismanager/v2/cmd/internal/cli"
	"github.com/hishamkaram/gismanager/v2/publish"
)

func main() { os.Exit(realMain()) }

// realMain is the testable entry point: it owns the signal-aware ctx,
// honors deferred cleanup, and translates an error return into a
// process exit code (0 for success or -version, 1 for any other
// failure). Splitting it out from main lets the deferred `cancel()`
// run before the os.Exit call.
func realMain() int {
	ctx, cancel := cli.SignalContext(context.Background())
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		if errors.Is(err, cli.ErrVersionRequested) {
			return 0
		}
		slog.Error("gismanager", "err", err)
		return 1
	}
	return 0
}

func run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("gismanager", flag.ContinueOnError)
	configFile := fs.String("config", "", "Config File")
	versionFlag := fs.Bool("version", false, "print build version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *versionFlag {
		cli.PrintVersion(os.Stdout, "gismanager")
		return cli.ErrVersionRequested
	}
	if err := cli.RequireFlag("gismanager", "config", *configFile); err != nil {
		return err
	}
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		return errors.New("config: file does not exist")
	}
	manager, err := publish.FromConfig(*configFile)
	if err != nil {
		return err
	}
	return manager.PublishAll(ctx)
}
