package main

import (
	"encoding/csv"
	"os"
	"sync"
)

type ProvincieMap struct {
	Map map[string]Provincia
}
type Provincia struct {
	ProvinciaExt string
	Zona         string
}

var instance *ProvincieMap
var mu sync.Mutex

func GetProvincieMapInstance() *ProvincieMap {
	if instance == nil {
		mu.Lock()
		defer mu.Unlock()
		instance = &ProvincieMap{}
		instance.Map = make(map[string]Provincia)
		f, err := os.Open("./vc/prov.tsv")
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

	}
	return instance
}
