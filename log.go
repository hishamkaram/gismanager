package gismanager

import (
	"log/slog"
	"os"
)

// GetLogger returns the project's default structured logger: a stdlib
// *slog.Logger writing text-formatted records to stderr.
//
// Callers building their own pipeline should construct an *slog.Logger
// directly (any handler — JSON, lumberjack-rotated, otel, etc.) and pass
// it on the [ManagerConfig].logger field via the constructor (PR 4 still
// loads it from FromConfig + GetLogger; functional-options constructor
// lands in a follow-up).
func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}
