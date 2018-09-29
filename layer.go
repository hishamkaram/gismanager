package gismanager

import (
	"fmt"

	"github.com/lukeroth/gdal"
)

//GdalLayer Layer
type GdalLayer struct {
	*gdal.Layer
}

//LayerToPostgis Layer to Postgis
func (layer *GdalLayer) LayerToPostgis(targetSource gdal.DataSource) (newLayer *GdalLayer) {
	if layer.Layer != nil {
		_layer := targetSource.CopyLayer(*layer.Layer, layer.Name(), []string{fmt.Sprintf("GEOMETRY_NAME=%s", layer.GeometryColumn())})
		newLayer = &GdalLayer{
			Layer: &_layer,
		}
	}
	return
}
