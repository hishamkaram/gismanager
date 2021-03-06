package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/hishamkaram/gismanager"
)

func main() {
	configFile := flag.String("config", "", "Config File")
	flag.Parse()
	if *configFile == "" {
		panic(errors.New("config 'Parameter required'"))
	}
	if _, err := os.Stat(*configFile); os.IsNotExist(err) {
		panic(errors.New("Config File Doesn't exist"))
	}
	manager, confErr := gismanager.FromConfig(*configFile)
	if confErr != nil {
		panic(confErr)
	}
	files, _ := gismanager.GetGISFiles(manager.Source.Path)
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
}
