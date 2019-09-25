package main

import (
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/teamdigitale/anpr-dashboard-server/sqlite"

	null "gopkg.in/guregu/null.v3"
)

//Data
var fornitori = []sqlite.Fornitore{
	sqlite.Fornitore{1, "Fornitore 1", "www.fornitore1.it", "fornitore1@mondo.it"},
	sqlite.Fornitore{2, "Fornitore 2", "www.fornitore2.it", "fornitore2@mondo.it"},
}

/*
Build a basic comune for the given input
the population are generated from random number
*/
func buildComuneFrom(name string, cod_istat string, prov string, datasubentro int64, datapresubentro int64, lat float64, lon float64, fornitore sqlite.Fornitore) sqlite.Comune {
	var comune sqlite.Comune
	comune.Name = name
	comune.CodiceIstat = cod_istat
	comune.Province = prov
	if datapresubentro > 0 {
		comune.DataPresubentro = null.NewTime(time.Unix(datapresubentro, 0), true)

	}
	if datasubentro > 0 {
		comune.DataSubentro = null.NewTime(time.Unix(datasubentro, 0), true)
	}
	comune.Lat = lat
	comune.Lon = lon
	comune.Population = rand.Int()
	comune.PopulationAIRE = rand.Int()

	comune.Fornitore = fornitore
	return comune
}

var comuni = []sqlite.Comune{
	buildComuneFrom("Comune1", "00145", "AP", -1, 1547164800, 45.3595112, 11.7890789, fornitori[0]),
	buildComuneFrom("Comune2", "00146", "RM", 1547264800, -1, 46.3595112, 12.7890789, fornitori[1]),
	buildComuneFrom("Comune3", "00147", "PC", 1567264800, -1, 46.3595112, 12.7890789, fornitori[0]),
}

func TestDateFormatting(t *testing.T) {
	InitServerConfigFromFile("./tools/config.sample.yaml")
	var date = dateFormattedOrNull(comuni[1].DataSubentro)
	assert.Equal(t, "12/01/2019", date)
}
func TestGetGetDashBoardData(t *testing.T) {
	InitServerConfigFromFile("./tools/config.sample.yaml")
	var dashboardData = GetDashBoardData(comuni)
	assert.Len(t, dashboardData.Geojson.Features, 3)

	feature1 := dashboardData.Geojson.Features[0]
	feature2 := dashboardData.Geojson.Features[1]

	log.Printf("Feature[0]: %v", feature1)
	log.Printf("Feature[1]: %v", feature2)
	//Basic geoJson check
	//Presubentro comes first
	assert.Nil(t, feature1.Properties["data_subentro"])
	assert.Equal(t, "11/01/2019", feature1.Properties["data_presubentro"])
	assert.Nil(t, feature1.Properties["prima_data_subentro"])
	assert.Nil(t, feature1.Properties["ultima_data_subentro"])
	assert.Nil(t, feature1.Properties["data_subentro_preferita"])

	assert.Equal(t, "31/08/2019", feature2.Properties["data_subentro"])
	assert.Nil(t, feature2.Properties["data_presubentro"])

	assert.NotEmpty(t, dashboardData.Charts)
	assert.NotEmpty(t, dashboardData.Fornitori)
	assert.NotEmpty(t, dashboardData.Geojson)

	//Summaries check
	assert.Equal(t, 2, dashboardData.Summaries.ComuniSubentro)
	assert.Equal(t, 1, dashboardData.Summaries.ComuniPreSubentro)
	assert.Equal(t, (comuni[1].Population + comuni[2].Population), dashboardData.Summaries.PopolazioneSubentro)
	assert.Equal(t, (comuni[1].PopulationAIRE + comuni[2].PopulationAIRE), dashboardData.Summaries.PopolazioneAireSubentro)
	assert.Equal(t, (comuni[0].Population), dashboardData.Summaries.PopolazionePresubentro)
	assert.Equal(t, (comuni[0].PopulationAIRE), dashboardData.Summaries.PopolazioneAirePreSubentro)

	//Fornitori Check
	//the second supplier has just 1 comune and is migrated
	assert.Equal(t, float64(100), dashboardData.Fornitori[0].PercentualeComuniSubentro)
	assert.Equal(t, fornitori[1].Name, dashboardData.Fornitori[0].Nome)
	assert.Equal(t, float64(50), dashboardData.Fornitori[1].PercentualeComuniSubentro)
	assert.Equal(t, fornitori[0].Name, dashboardData.Fornitori[1].Nome)
	//Carts Checks
	assert.Len(t, dashboardData.Charts.Presubentro, 1)
	assert.Len(t, dashboardData.Charts.Subentro, 2)

	assert.Equal(t, 1, dashboardData.Charts.Subentro[0].Comuni)
	assert.Equal(t, 2, dashboardData.Charts.Subentro[1].Comuni)
	assert.Equal(t, comuni[1].Population, dashboardData.Charts.Subentro[0].Popolazione)
	assert.Equal(t, comuni[1].DataSubentro.Time, dashboardData.Charts.Subentro[0].Date)
	assert.Equal(t, comuni[2].DataSubentro.Time, dashboardData.Charts.Subentro[1].Date)

	assert.Equal(t, comuni[1].Population+comuni[2].Population, dashboardData.Charts.Subentro[1].Popolazione)

}
