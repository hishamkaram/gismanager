package gismanager

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogger(t *testing.T) {
	logger := GetLogger()
	assert.NotNil(t, logger)
	assert.Equal(t, reflect.TypeOf(logger).String(), "*logrus.Logger")

}
