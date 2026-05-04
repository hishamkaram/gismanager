package gismanager

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.NotNil(t, logger)
	assert.IsType(t, &slog.Logger{}, logger)
}
