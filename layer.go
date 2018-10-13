package gismanager

import (
	"errors"
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

//PublishGeoserverLayer publish Layer to postgis
func (manager *ManagerConfig) PublishGeoserverLayer(layer *GdalLayer) (ok bool, err error) {
	catalog := manager.GetGeoserverCatalog()
	workspaceExists, _ := catalog.WorkspaceExists(manager.Geoserver.WorkspaceName)
	if !workspaceExists {
		workspaceCreated, workspaceCreateErr := catalog.CreateWorkspace(manager.Geoserver.WorkspaceName)
		if workspaceCreateErr != nil || !workspaceCreated {
			manager.logger.Error(workspaceCreateErr)
			err = workspaceCreateErr
			return
		}
	}
	storeExits, _ := catalog.DatastoreExists(manager.Geoserver.WorkspaceName, manager.Datastore.Name, true)
	if !storeExits {
		datastoreConnection := gsconfig.DatastoreConnection{
			Name:   manager.Datastore.Name,
			Host:   manager.Datastore.Host,
			Port:   int(manager.Datastore.Port),
			DBName: manager.Datastore.DBName,
			Type:   "postgis",
			DBUser: manager.Datastore.DBUser,
			DBPass: manager.Datastore.DBPass,
		}
		created, createErr := catalog.CreateDatastore(datastoreConnection, manager.Geoserver.WorkspaceName)
		if createErr != nil || !created {
			manager.logger.Error(createErr)
			err = createErr
			return
		}
	}
	ok, err = catalog.PublishPostgisLayer(manager.Geoserver.WorkspaceName, manager.Datastore.Name, layer.Name(), layer.Name())
	return
}

//LayerToPostgis add Layer to Postgis
func (layer *GdalLayer) LayerToPostgis(targetSource *gdal.DataSource, manager *ManagerConfig, overwrite bool) (newLayer *GdalLayer, err error) {
	if targetSource == nil {
		err = errors.New("Invalid Datasource")
		return
	}
	if layer.Layer == nil {
		err = errors.New("Invalid Layer")
		return
	}
	datasource := *targetSource
	var options []string
	geomName := layer.GeometryColumn()
	if geomName != "" {
		options = append(options, fmt.Sprintf("GEOMETRY_NAME=%s", layer.GeometryColumn()))
	}
	if overwrite {
		options = append(options, "OVERWRITE=YES")
	}
	_layer := datasource.CopyLayer(*layer.Layer, layer.Name(), options)
	newLayer = &GdalLayer{
		Layer: &_layer,
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
	logger := GetLogger()
	if layer.Layer != nil {
		count, ok := layer.Layer.FeatureCount(true)
		if !ok {
			logger.Error("Could not read features")
		} else {
			logger.Infof("We Found %d Feature", count)
			for index := 0; index < count; index++ {
				f := layer.Layer.Feature(index)
				features = append(features, &f)
			}
		}
	}
	return
}
