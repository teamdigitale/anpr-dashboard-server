package main

import (
	"encoding/csv"
	"os"
	"sync"
)

//ProvinciaMap {"AP":{ProvinciaExt:"Ascoli Piceno", Zona:"Marche Centro" }}
type ProvincieMap struct {
	Map map[string]Provincia
}
type Provincia struct {
	ProvinciaExt string
	Zona         string
}

var instance *ProvincieMap
var onceVC sync.Once

//GetProvincieMapInstance is the singleton thread safe implementation - loads in memory the prov.tsv Dicionary and returns a Map with prov code as key
func GetProvincieMapInstance() *ProvincieMap {
	onceVC.Do(func() {
		//for thread safety a mutex acquires the lock on the instance creation
		instance = &ProvincieMap{}
		instance.Map = make(map[string]Provincia)
		f, err := os.Open(GetServerConfig().StorageOptions.Vocabularies + "prov.tsv")
		if err != nil {
			panic(err)
		}
		lines, err := csv.NewReader(f).ReadAll()
		if err != nil {
			panic(err)
		}

		for _, l := range lines {
			(*instance).Map[l[0]] = Provincia{l[1], l[3]}
		}

		defer f.Close()

	})
	return instance
}
