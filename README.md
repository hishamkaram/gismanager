[![Go Report Card](https://goreportcard.com/badge/github.com/hishamkaram/gismanager)](https://goreportcard.com/report/github.com/hishamkaram/gismanager)
[![GitHub license](https://img.shields.io/github/license/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/blob/master/LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/issues)
[![Coverage Status](https://coveralls.io/repos/github/hishamkaram/gismanager/badge.svg?branch=master&service=github)](https://coveralls.io/github/hishamkaram/gismanager?branch=master&service=github)
[![Build Status](https://travis-ci.org/hishamkaram/gismanager.svg?branch=master)](https://travis-ci.org/hishamkaram/gismanager)
[![Documentation](https://godoc.org/github.com/hishamkaram/gismanager?status.svg)](https://godoc.org/github.com/hishamkaram/gismanager?)
[![GitHub forks](https://img.shields.io/github/forks/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/network)
[![GitHub stars](https://img.shields.io/github/stars/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/stargazers)
[![Twitter](https://img.shields.io/twitter/url/https/github.com/hishamkaram/gismanager/edit/master/README.md.svg?style=social)](https://twitter.com/intent/tweet?text=Wow:&url=https%3A%2F%2Fgithub.com%2Fhishamkaram%2Fgeoserver%2Fedit%2Fmaster%2FREADME.md)


<p align="center">
  <img src="http://geoserver.org/img/OSGeo_project.png" width="200"/>
</p>
<p align="center">
  <img src="https://i.imgur.com/31CL1xg.png" width="200"/>
</p>

# GISManager
Publish Your GIS Data(Vector Data) to PostGIS and Geoserver

- How to install:
    - `go get -v github.com/hishamkaram/gismanager`
- Usage:
  - create `ManagerConfig` instance:
    ```
    manager:= gismanager.ManagerConfig{
      Geoserver: gismanager.GeoserverConfig{WorkspaceName: "golang", Username: "admin", Password: "geoserver", ServerURL: "http://localhost:8080/geoserver"},
      Datastore: gismanager.DatastoreConfig{Host: "localhost", Port: 5432, DBName: "gis", DBUser: "golang", DBPass: "golang", Name: "gismanager_data"},
      Source:    gismanager.SourceConfig{Path: "./testdata"},
      logger:    gismanager.GetLogger(),
    }
    ```
  - `testdata` folder content:
    ```
    ./testdata/
    ├── neighborhood_names_gis.geojson
    ├── nested
    │   └── nyc_wi-fi_hotspot_locations.geojson
    ├── sample.gpkg
    ```
  - get Supported GIS Files:
    ```
    files, _ := gismanager.GetGISFiles(manager.Source.Path)
    for _, file := range files {
      fmt.Println(file)
    }
    ```
    - output:
      ```
      <full_path>/testdata/neighborhood_names_gis.geojson
      <full_path>/testdata/nested/nyc_wi-fi_hotspot_locations.geojson
      <full_path>/testdata/sample.gpkg
      ```
  - read files and get layers Schema:
    ```
      for _, file := range files {
        source, ok := manager.OpenSource(file, 0)
        if ok {
          for index := 0; index < source.LayerCount(); index++ {
            layer := source.LayerByIndex(index)
            gLayer := gismanager.GdalLayer{
              Layer: &layer,
            }
            fmt.Println(layer.Name())
            for _, f := range gLayer.GetLayerSchema() {
              fmt.Printf("\n%+v\n", *f)
            }
          }
        }
      }
    ```
    - output sample:
      ```
      neighborhood_names_gis

      {Name:geom Type:POINT}

      {Name:stacked Type:String}

      {Name:name Type:String}

      {Name:annoline1 Type:String}

      {Name:annoline3 Type:String}

      {Name:objectid Type:String}

      {Name:annoangle Type:String}

      {Name:annoline2 Type:String}

      {Name:borough Type:String}
      ...
      ```
  - add your gis data to your database:
    ```
      for _, file := range files {
          source, ok := manager.OpenSource(file, 0)
          targetSource, targetOK := manager.OpenSource(manager.Datastore.BuildConnectionString(), 1)
          if ok && targetOK {
            for index := 0; index < source.LayerCount(); index++ {
              layer := source.LayerByIndex(index)
              gLayer := gismanager.GdalLayer{
                Layer: &layer,
              }
              newLayer, postgisErr := gLayer.LayerToPostgis(targetSource, manager, true)
              if postgisErr != nil {
                panic(postgisErr)
              }
              logger.Infof("Layer: %s added to you database", newLayer.Name())
            }
          }
      }
    ```
    - output:
      ```
      INFO[14-10-2018 17:28:37] Layer: neighborhood_names_gis added to you database 
      INFO[14-10-2018 17:28:38] Layer: nyc_wi_fi_hotspot_locations added to you database 
      INFO[14-10-2018 17:28:38] Layer: hwy_patrol added to you database
      ```
   - update the previous code to publish your postgis layers to geoserver
     ```
      for _, file := range files {
        source, ok := manager.OpenSource(file, 0)
        targetSource, targetOK := manager.OpenSource(manager.Datastore.BuildConnectionString(), 1)
        if ok && targetOK {
          for index := 0; index < source.LayerCount(); index++ {
            layer := source.LayerByIndex(index)
            gLayer := gismanager.GdalLayer{
              Layer: &layer,
            }
            if newLayer, postgisErr := gLayer.LayerToPostgis(targetSource, manager, true); newLayer.Layer != nil || postgisErr != nil {
              ok, pubErr := manager.PublishGeoserverLayer(newLayer)
              if pubErr != nil {
                logger.Error(pubErr)
              }
              if !ok {
                logger.Error("Failed to Publish")
              } else {
                logger.Info("published")
              }
            }

          }
        }
      }
     ```
     - output:
     ```
      INFO[14-10-2018 17:37:07] url:http://localhost:8080/geoserver/rest/workspaces/golang  Status=404  
      ERRO[14-10-2018 17:37:07] No such workspace: 'golang' found            
      INFO[14-10-2018 17:37:07] url:http://localhost:8080/geoserver/rest/workspaces  Status=201  
      INFO[14-10-2018 17:37:07] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis?quietOnNotFound=true  Status=404  
      INFO[14-10-2018 17:37:07] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores  Status=201  
      ERRO[14-10-2018 17:37:07] {"featureType":{"name":"neighborhood_names_gis","nativeName":"neighborhood_names_gis"}} 
      INFO[14-10-2018 17:37:07] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis/featuretypes  Status=201  
      INFO[14-10-2018 17:37:07] published                                    
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang  Status=200  
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis?quietOnNotFound=true  Status=200  
      ERRO[14-10-2018 17:37:08] {"featureType":{"name":"nyc_wi_fi_hotspot_locations","nativeName":"nyc_wi_fi_hotspot_locations"}} 
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis/featuretypes  Status=201  
      INFO[14-10-2018 17:37:08] published                                    
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang  Status=200  
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis?quietOnNotFound=true  Status=200  
      ERRO[14-10-2018 17:37:08] {"featureType":{"name":"hwy_patrol","nativeName":"hwy_patrol"}} 
      INFO[14-10-2018 17:37:08] url:http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gis/featuretypes  Status=201  
      INFO[14-10-2018 17:37:08] published 
     ```
     - done check you geoserver or via geoserver rest api url http://localhost:8080/geoserver/rest/layers.json : 
       ```
        {
          "layers": {
            "layer": [..., {
              "name": "golang:hwy_patrol",
              "href": "http:\/\/localhost:8080\/geoserver\/rest\/layers\/golang%3Ahwy_patrol.json"
            }, {
              "name": "golang:neighborhood_names_gis",
              "href": "http:\/\/localhost:8080\/geoserver\/rest\/layers\/golang%3Aneighborhood_names_gis.json"
            }, {
              "name": "golang:nyc_wi_fi_hotspot_locations",
              "href": "http:\/\/localhost:8080\/geoserver\/rest\/layers\/golang%3Anyc_wi_fi_hotspot_locations.json"
            }]
          }
        }
       ```
 ---
 
# Todo:
  - [ ] Handle zipped shapefiles
  - [ ] backup postgis as geopackage
