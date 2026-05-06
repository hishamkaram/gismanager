// Package cli provides shared scaffolding for gismanager's command-line
// binaries (gismanager, layerSchema, gisconvert). It is internal —
// external consumers should use the top-level [github.com/hishamkaram/gismanager]
// library API instead.
//
// The helpers here cover three concerns common to every binary:
//
//   - SignalContext — Ctrl-C / SIGTERM propagate cleanly through PostGIS
//     pings, GeoServer REST calls, and any in-flight context-aware GDAL
//     conversion (the conversion subsystem honors ctx at the boundary).
//   - PrintVersion — operators can verify which build is running in
//     prod via a stable single-line format.
//   - RequireFlag — uniform error envelope for missing required flags
//     across all three CLIs.
package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

// Version, Commit, and Date are populated at build time via -ldflags:
//
//	go build -ldflags='\
//	    -X github.com/hishamkaram/gismanager/cmd/internal/cli.Version=v1.4.0 \
//	    -X github.com/hishamkaram/gismanager/cmd/internal/cli.Commit=abc1234 \
//	    -X github.com/hishamkaram/gismanager/cmd/internal/cli.Date=2026-05-06T12:00:00Z'
//
// When unset (e.g. plain `go run` or `go install` from source),
// [PrintVersion] falls back to runtime/debug.ReadBuildInfo so output
// stays useful without any build-time integration.
var (
	Version = ""
	Commit  = ""
	Date    = ""
)

// SignalContext returns a context that cancels on os.Interrupt (SIGINT,
// e.g. Ctrl-C) or syscall.SIGTERM (e.g. `kill <pid>` or a Kubernetes
// pod terminating). Caller must defer the returned cancel func to
// release the signal handler.
//
// Use this in place of context.Background() at every CLI main() so a
// long-running publish or conversion can be interrupted cleanly. The
// gismanager library is context-first by design — every public method
// honors ctx at the boundary, even if (per the v1.3 CHANGELOG) some
// underlying CGo calls remain synchronous and uninterruptible.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// PrintVersion writes a single line of build metadata to w in a
// machine-friendly format suitable for shell pipelines:
//
//	<name> version=<v> commit=<c> built=<d>
//
// Each field falls back to runtime/debug.ReadBuildInfo (vcs.revision,
// vcs.time, Main.Version) when the corresponding ldflag value is empty,
// and to the literal "(unknown)" only when both ldflag and build info
// are missing.
func PrintVersion(w io.Writer, name string) {
	v, c, d := Version, Commit, Date
	if v == "" || c == "" || d == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if v == "" && info.Main.Version != "" {
				v = info.Main.Version
			}
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "" {
						c = s.Value
					}
				case "vcs.time":
					if d == "" {
						d = s.Value
					}
				}
			}
		}
	}
	if v == "" {
		v = "(unknown)"
	}
	if c == "" {
		c = "(unknown)"
	}
	if d == "" {
		d = "(unknown)"
	}
	// Discarding the write error is intentional — PrintVersion is a
	// best-effort fire-and-forget on stdout for an operator-facing
	// CLI; if stdout is closed the binary has bigger problems and
	// the next slog.Error will surface them.
	_, _ = fmt.Fprintf(w, "%s version=%s commit=%s built=%s\n", name, v, c, d)
}

// RequireFlag returns a stable error envelope for a missing required
// CLI flag. The error message is "<binary>: --<flag> is required",
// matching the style users see for `flag` package parse errors.
//
// Returns nil if value is non-empty.
func RequireFlag(binary, flag, value string) error {
	if value == "" {
		return fmt.Errorf("%s: --%s is required", binary, flag)
	}
	return nil
}

// ErrVersionRequested is returned by [HandleVersionFlag] when the user
// passed -version. Callers should treat this as a clean exit (status 0,
// no error log).
var ErrVersionRequested = errors.New("version requested")
