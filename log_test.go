package gismanager

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetLogger confirms the v1.x public API (gismanager.GetLogger)
// resolves to a non-nil *slog.Logger. The implementation lives in
// [internal/slogx.Default]; the deeper test coverage of that path
// is in internal/slogx/default_test.go.
func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.NotNil(t, logger)
	assert.IsType(t, &slog.Logger{}, logger)
}
