package gismanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildConnectionString(t *testing.T) {
	ds := datastore{
		Host:   "localhost",
		Port:   5432,
		DBName: "gis",
		DBUser: "gis",
		DBPass: "gis",
	}
	conn := ds.BuildConnectionString()
	assert.NotNil(t, conn)
	assert.NotEqual(t, "", conn)
	assert.True(t, pgRegex.Match([]byte(conn)))

}
