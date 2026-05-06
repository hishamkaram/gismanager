package gismanager

import (
	"log/slog"

	"github.com/hishamkaram/gismanager/internal/slogx"
)

// GetLogger returns the project's default structured logger: a stdlib
// *slog.Logger writing text-formatted records to stderr.
//
// As of the v2 restructure groundwork (Phase 1), the implementation
// lives in [internal/slogx.Default]; this is a thin wrapper for
// v1.x compatibility. v2 callers should construct their own
// *slog.Logger directly (any handler — JSON, lumberjack-rotated,
// OpenTelemetry-bridged) and pass it on via the available
// functional options. The internal package is not part of the
// v2 public API.
func GetLogger() *slog.Logger {
	return slogx.Default()
}
