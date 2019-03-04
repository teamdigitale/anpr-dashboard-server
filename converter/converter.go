// This file was downloaded from:
//   https://github.com/googlemaps/google-maps-services-go/tree/master/examples/geocoding/cmdline
//
// And modified by ccontavalli@gmail.com to generate json files with city
// and companies data used for the ANPR system.
//
// The changes are ugly, main purpose of the work was to get clean and
// normalized data as quickly as possible.

// This is the original Copyright notice for the code:
//
// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main contains a simple command line tool for Geocoding API
// Documentation: https://developers.google.com/maps/documentation/geocoding/

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	//"github.com/kr/pretty"
	"golang.org/x/net/context"
	"googlemaps.github.io/maps"

	"github.com/teamdigitale/anpr-dashboard-server/sqlite"
	"gopkg.in/guregu/null.v3"
)

var (
	flag_schede  = flag.String("schede", "", "Name of the file containing the schede.")
	flag_comuni  = flag.String("output_comuni", "comuni.json", "Output file with the list of cities.")
	flag_aziende = flag.String("output_aziende", "aziende.json", "Output file with the list of software houses.")
	flag_sqlite  = flag.String("output_sqlite", "", "Sqlite output file.")
	flag_cache   = flag.String("cache", "cache.json", "Cache of address to location mappings.")

	flag_apikey   = flag.String("key", "", "API Key for using Google Maps API.")
	flag_language = flag.String("language", "it", "The language in which to return results.")
	flag_region   = flag.String("region", "it", "The region code, specified as a ccTLD two-character value.")
)

type Comune struct {
	Nome        string // 1
	Provincia   string // 2
	CodiceIstat string // 8

	Via    string // 20
	Cap    string // 21
	Civico string // 22

	Popolazione int // 19
	Postazioni  int // 14

	NomeReferente      string // 13
	CognomeReferente   string // 12
	TelefonoReferente  string // 10
	CellulareReferente string // 16
	EmailReferente     string // 11

	PecComune string // 9

	IdAzienda int

	Location maps.LatLng
	Address  string
}

type CacheEntry struct {
	Query string
	Error string

	Location []maps.GeocodingResult
}

type CachedClient struct {
	client *maps.Client
	apikey string

	index map[string]CacheEntry
	file  *os.File

	changes int
	saves   int
}

func NewCache(apikey string, filename string) (*CachedClient, error) {
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(f)

	var entries []CacheEntry
	err = decoder.Decode(&entries)
	if err != nil && err != io.EOF {
		return nil, err
	}

	client := &CachedClient{nil, apikey, make(map[string]CacheEntry), f, 0, 0}
	for _, entry := range entries {
		client.index[entry.Query] = entry
	}

	return client, nil
}

func (cc *CachedClient) Save() error {
	// Check first if anything worth changing, changed.
	if cc.saves == cc.changes {
		return nil
	}

	entries := make([]CacheEntry, 0, 10000)
	for _, value := range cc.index {
		entries = append(entries, value)
	}

	cc.file.Seek(0, 0)
	cc.file.Truncate(0)
	encoder := json.NewEncoder(cc.file)
	err := encoder.Encode(entries)
	cc.file.Sync()

	cc.saves = cc.changes

	return err
}

func (cc *CachedClient) Lookup(address string) ([]maps.GeocodingResult, error) {
	// Check the cache first.
	entry, ok := cc.index[address]
	if ok {
		if entry.Error == "" {
			return entry.Location, nil
		}

		if !strings.Contains(entry.Error, "OVER_QUERY_LIMIT") {
			return entry.Location, fmt.Errorf("%s", entry.Error)
		}

		// Retry, in case the error was OVER_QUERY_LIMIT before.
	}

	if cc.client == nil {
		if cc.apikey == "" {
			log.Fatal("Need to locate address %s - Must provide --apikey option", address)
		}
		mclient, err := maps.NewClient(maps.WithAPIKey(cc.apikey))
		if err != nil {
			log.Fatal("Cannot initialize maps apis: %v", err)
		}
		cc.client = mclient
	}

	results, err := GeoCode(cc.client, address)
	message := ""
	if err != nil {
		message = err.Error()
	}

	// Fill the cache next, avoid saving OVER_QUERY_LIMIT errors.
	if !strings.Contains(message, "OVER_QUERY_LIMIT") {
		cc.index[address] = CacheEntry{address, message, results}
		cc.changes += 1
	}
	return results, err
}

type Azienda struct {
	Nome string
	Id   int

	Comuni int
}

const (
	kFieldNome      = 1
	kFieldProvincia = 2

	kFieldSoftwareHouse = 3

	kFieldCodiceIstat = 8

	kFieldPecComune         = 9
	kFieldTelefonoReferente = 10
	kFieldEmailReferente    = 11
	kFieldCognomeReferente  = 12
	kFieldNomeReferente     = 13

	kFieldPostazioni         = 14
	kFieldCellulareReferente = 16
	kFieldPopolazione        = 19

	kFieldVia    = 20
	kFieldCap    = 21
	kFieldCivico = 22
)

func GeoCode(client *maps.Client, Address string) ([]maps.GeocodingResult, error) {
	r := &maps.GeocodingRequest{
		Address:  Address,
		Language: *flag_language,
		Region:   *flag_region,
	}

	resp, err := client.Geocode(context.Background(), r)
	return resp, err
}

func FindLocation(client *CachedClient, record []string) *maps.GeocodingResult {
	addresses := []string{
		fmt.Sprintf("%s, %s, %s %s %s", record[kFieldVia], record[kFieldCivico], record[kFieldCap], record[kFieldNome], record[kFieldProvincia]),
		fmt.Sprintf("%s, %s, %s %s", record[kFieldVia], record[kFieldCivico], record[kFieldNome], record[kFieldProvincia]),
		fmt.Sprintf("%s, %s %s", record[kFieldVia], record[kFieldNome], record[kFieldProvincia]),
		fmt.Sprintf("%s %s", record[kFieldNome], record[kFieldProvincia]),
		fmt.Sprintf("%s, Italy", record[kFieldCap]),
		fmt.Sprintf("%s", record[kFieldNome]),
	}

	for index, address := range addresses {
		location, err := client.Lookup(address)
		if err != nil {
			log.Printf("[%d] ERROR while locationg: %s - %v\n", index, address, err)
			continue
		}
		if len(location) < 1 {
			log.Printf("[%d] ERROR could not locate: %s - %v\n", index, address, err)
			continue
		}
		return &location[0]
	}
	log.Printf("ERROR could not locate: %v", addresses)
	return nil
}

func ParseSchede(schede *os.File, aziende map[string]*Azienda, comuni *[]*Comune) {
	client, err := NewCache(*flag_apikey, *flag_cache)
	if err != nil {
		log.Fatal("Cannot initialize cache: %v", err)
	}
	defer client.Save()

	cr := csv.NewReader(schede)
	cr.FieldsPerRecord = -1
	for i := 0; ; i++ {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println(err)
			continue
		}

		// Skip headers.
		if i == 0 {
			continue
		}

		swhouse := record[kFieldSoftwareHouse]
		azienda, found := aziende[swhouse]
		if !found {
			azienda = &Azienda{swhouse, len(aziende), 0}
			aziende[swhouse] = azienda
		}
		azienda.Comuni += 1

		popolazione, err := strconv.Atoi(record[kFieldPopolazione])
		if err != nil {
			log.Printf("ERROR Could not convert popolazione in %v", record)
			popolazione = 0
		}
		postazioni, err := strconv.Atoi(record[kFieldPostazioni])
		if err != nil {
			log.Printf("ERROR Could not convert postazioni in %v", record)
			postazioni = 0
		}

		location := FindLocation(client, record)
		comune := &Comune{
			Nome:               record[kFieldNome],
			Provincia:          record[kFieldProvincia],
			CodiceIstat:        record[kFieldCodiceIstat],
			Via:                record[kFieldVia],
			Cap:                record[kFieldCap],
			Civico:             record[kFieldCivico],
			Popolazione:        popolazione,
			Postazioni:         postazioni,
			NomeReferente:      record[kFieldNomeReferente],
			CognomeReferente:   record[kFieldCognomeReferente],
			TelefonoReferente:  record[kFieldTelefonoReferente],
			CellulareReferente: record[kFieldCellulareReferente],
			EmailReferente:     record[kFieldEmailReferente],
			PecComune:          record[kFieldPecComune],

			IdAzienda: azienda.Id,

			Location: location.Geometry.Location,
			Address:  location.FormattedAddress,
		}

		*comuni = append(*comuni, comune)
		client.Save()
	}

}

func SaveSQLite(aformatted []*Azienda, comuni []*Comune) {
	db := sqlite.OpenDB(*flag_sqlite)
	sqlite.InitDB(db)

	sqlite_fornitori := []sqlite.Fornitore{}
	for id, azienda := range aformatted {
		sqlite_fornitore := sqlite.Fornitore{
			Id:   id + 1,
			Name: azienda.Nome,
		}
		log.Printf("Fornitore: %v", sqlite_fornitore)
		sqlite_fornitori = append(sqlite_fornitori, sqlite_fornitore)
	}
	sqlite.InsertFornitori(db, sqlite_fornitori)

	sqlite_comuni := []sqlite.Comune{}
	for id, comune := range comuni {
		responsabile := sqlite.Responsible{
			Name:    comune.NomeReferente,
			Surname: comune.CognomeReferente,
			Phone:   comune.TelefonoReferente,
			Mobile:  comune.CellulareReferente,
			Email:   comune.EmailReferente,
		}
		indirizzo := sqlite.Indirizzo{
			Via:    comune.Via,
			Cap:    comune.Cap,
			Civico: comune.Civico,
			Pec:    comune.PecComune,
		}
		sqlite_comune := sqlite.Comune{
			Id:          id + 1,
			CodiceIstat: comune.CodiceIstat,
			Name:        comune.Nome,
			Province:    comune.Provincia,
			Population:  comune.Popolazione,
			Postazioni:  null.IntFrom(int64(comune.Postazioni)),
			Lat:         comune.Location.Lat,
			Lon:         comune.Location.Lng,
			Fornitore:   sqlite_fornitori[comune.IdAzienda],
			Responsible: responsabile,
			Indirizzo:   indirizzo,
		}
		log.Printf("Comune: %v", sqlite_comune)
		sqlite_comuni = append(sqlite_comuni, sqlite_comune)
	}
	sqlite.InsertComuni(db, sqlite_comuni)

	sqlite.Close(db)
}

func main() {
	flag.Parse()

	if *flag_schede == "" {
		log.Fatal("Must provide flag -schede")
	}

	fd, err := os.Open(*flag_schede)
	if err != nil {
		log.Fatal("Cannot open file: %v", err)
	}

	aziende := make(map[string]*Azienda)
	comuni := make([]*Comune, 0, 10000)

	ParseSchede(fd, aziende, &comuni)

	aformatted := make([]*Azienda, len(aziende))
	for _, v := range aziende {
		aformatted[v.Id] = v
	}

	if *flag_sqlite == "" {
		jsona, _ := json.MarshalIndent(aformatted, "", "  ")
		ioutil.WriteFile(*flag_aziende, jsona, 0644)

		jsonc, _ := json.MarshalIndent(comuni, "", "  ")
		ioutil.WriteFile(*flag_comuni, jsonc, 0644)
	} else {
		SaveSQLite(aformatted, comuni)
	}
}
