// Package slogx provides a single helper that returns the project's
// default *slog.Logger. Moved here from the root gismanager package
// as part of the v2 restructure groundwork (Phase 1) so the future
// convert/ and publish/ subpackages can reference it without a
// circular import through the root.
//
// External callers continue to call gismanager.GetLogger() via the
// thin wrapper the root package keeps. In v2, the convert/ and
// publish/ subpackages call slogx.Default() directly.
package slogx

import (
	"log/slog"
	"os"
)

// Default returns a *slog.Logger writing text-formatted records to
// stderr — the project's fallback logger when no caller-supplied
// *slog.Logger is configured.
//
// Production users typically construct their own *slog.Logger (any
// handler — JSON, file-rotated, OpenTelemetry-bridged) and pass it
// to library entry points via the available functional options.
func Default() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}
