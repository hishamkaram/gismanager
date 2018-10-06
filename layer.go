package gismanager

import (
	"fmt"

	gsconfig "github.com/hishamkaram/geoserver"
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
func (layer *GdalLayer) LayerToPostgis(targetSource gdal.DataSource, manager *ManagerConfig) (newLayer *GdalLayer) {
	catalog := gsconfig.GetCatalog(manager.Geoserver.ServerURL, manager.Geoserver.Username, manager.Geoserver.Password)
	storeExits, datastoreErr := catalog.DatastoreExists(manager.Geoserver.WorkspaceName, manager.Datastore.Name, true)
	if datastoreErr != nil {
		manager.logger.Error(datastoreErr)
		return
	}
	if !storeExits {
		datastoreConnection := gsconfig.DatastoreConnection{
			Name:   manager.Datastore.Name,
			Host:   manager.Datastore.Host,
			Port:   int(manager.Datastore.Port),
			DBName: manager.Datastore.DBName,
			DBUser: manager.Datastore.DBUser,
			DBPass: manager.Datastore.DBPass,
		}
		created, createErr := catalog.CreateDatastore(datastoreConnection, manager.Geoserver.WorkspaceName)
		if createErr != nil || !created {
			manager.logger.Error(createErr)
			return
		}
	}
	if layer.Layer != nil {
		var options []string
		geomName := layer.GeometryColumn()
		if geomName != "" {
			options = append(options, fmt.Sprintf("GEOMETRY_NAME=%s", layer.GeometryColumn()))
		}
		_layer := targetSource.CopyLayer(*layer.Layer, layer.Name(), options)
		newLayer = &GdalLayer{
			Layer: &_layer,
		}
	}
	return
}

//GetGeomtryName Get Geometry Name point/line/....etc
func (layer *GdalLayer) GetGeomtryName() (geometryName string) {
	geom := gdal.Create(layer.Layer.Type())
	geometryName = geom.Name()
	return
}

//GetLayerSchema return slice of layer fields
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

//GetFeatures return layer features
func (layer *GdalLayer) GetFeatures() (features []*gdal.Feature) {
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
