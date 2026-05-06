// Command layerSchema prints the geometry + attribute schema of every
// supported GIS file under the configured source directory. Read-only
// inspector — does not load to PostGIS or publish to GeoServer.
//
// Output modes:
//
//   - default (text)  — human-readable per-layer block
//   - -json           — newline-delimited single JSON document with
//     a top-level array of {path,name,fields[]} entries,
//     suitable for `jq` / shell scripting / Terraform.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/hishamkaram/gismanager"
	"github.com/hishamkaram/gismanager/cmd/internal/cli"
)

// layerEntry is the JSON shape for one yielded layer when -json is set.
// Field tags are explicit so the JSON keys stay stable even if the
// underlying struct fields are renamed in a future refactor.
type layerEntry struct {
	Path   string                   `json:"path"`
	Name   string                   `json:"name"`
	Fields []*gismanager.LayerField `json:"fields"`
}

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
	jsonFlag := fs.Bool("json", false, "emit a single JSON document on stdout instead of human-readable text")
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

	if *jsonFlag {
		return runJSON(ctx, manager, os.Stdout)
	}
	return runText(ctx, manager, os.Stdout)
}

// runText emits the human-readable per-layer block format. Walk errors
// are logged to stderr via slog (the configured logger) but don't abort
// the loop — same behavior as pre-1.4.
func runText(ctx context.Context, manager *gismanager.ManagerConfig, out *os.File) error {
	for item, walkErr := range manager.Walk(ctx) {
		if walkErr != nil {
			slog.Error("walk", "err", walkErr)
			continue
		}
		// Discarding write errors is intentional — the inspector is a
		// best-effort fire-and-forget on stdout. If the writer is
		// closed mid-output, the next write returns and the program
		// exits naturally; surfacing every per-line error would just
		// spam noise.
		_, _ = fmt.Fprintln(out, item.Layer.Name())
		for _, f := range item.Layer.GetLayerSchema() {
			_, _ = fmt.Fprintf(out, "\n%+v\n", *f)
		}
	}
	return nil
}

// runJSON collects every successfully-walked layer into a slice of
// [layerEntry] values and writes a single JSON array to out at the
// end. Walk errors go to stderr via slog (matching the text path);
// only readable layers appear in the JSON document.
//
// Compact (single-line) JSON is the default — pipe through `jq` or
// `python -m json.tool` if a human is reading it. Compact output is
// the right shell-pipeline default: smaller, faster to parse, and a
// drop-in for tooling that reads stdout.
func runJSON(ctx context.Context, manager *gismanager.ManagerConfig, out *os.File) error {
	entries := []layerEntry{}
	for item, walkErr := range manager.Walk(ctx) {
		if walkErr != nil {
			slog.Error("walk", "err", walkErr)
			continue
		}
		entries = append(entries, layerEntry{
			Path:   item.Path,
			Name:   item.Layer.Name(),
			Fields: item.Layer.GetLayerSchema(),
		})
	}
	enc := json.NewEncoder(out)
	if err := enc.Encode(entries); err != nil {
		return fmt.Errorf("encode JSON: %w", err)
	}
	return nil
}
