package gismanager

import (
	"fmt"

	"github.com/lukeroth/gdal"
)

//GdalLayer Layer
type GdalLayer struct {
	*gdal.Layer
}

//LayerField Field
type LayerField struct {
	Name string
	Type string
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

//GetGeomtryName Get Geometry Name
func (layer *GdalLayer) GetGeomtryName() (geometryName string) {
	geom := gdal.Create(layer.Layer.Type())
	geometryName = geom.Name()
	return
}

//GetLayerSchema Get Layer Schema
func (layer *GdalLayer) GetLayerSchema() (fields []*LayerField) {
	if layer.Layer != nil {
		layerDef := layer.Layer.Definition()
		geomField := LayerField{
			Name: layer.Layer.GeometryColumn(),
			Type: layer.GetGeomtryName(),
		}
		fields = append(fields, &geomField)
		for index := 0; index < layerDef.FieldCount(); index++ {
			fieldDef := layerDef.FieldDefinition(index)
			layerField := LayerField{
				Name: fieldDef.Name(),
				Type: fieldDef.Type().Name(),
			}
			fields = append(fields, &layerField)

		}
	}
	return
}

//GetFeature Get Layer Features
func (layer *GdalLayer) GetFeature() (features []*gdal.Feature) {
	if layer.Layer != nil {
		count, ok := layer.Layer.FeatureCount(true)
		if !ok {

		} else {
			for index := 0; index < count; index++ {
				f := layer.Layer.Feature(index)
				features = append(features, &f)
			}
		}
	}
	return
}
