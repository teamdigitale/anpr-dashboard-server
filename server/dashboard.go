package main

import (
	"container/heap"
	"math"
	"sort"
	"time"

	"github.com/paulmach/go.geojson"
	"github.com/teamdigitale/anpr-dashboard-server/sqlite"
	null "gopkg.in/guregu/null.v3"
)

//A priority queue to mantain the heap
type ComuniHeap []sqlite.Comune

func (h ComuniHeap) Len() int { return len(h) }

func (h ComuniHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *ComuniHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's length,
	// not just its contents.
	*h = append(*h, x.(sqlite.Comune))
}

func (h *ComuniHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

/**
Order in ascending - descending order the heap size
*/
func (h ComuniHeap) Less(i, j int) bool {
	comuneI := h[i]
	comuneJ := h[j]
	if comuneI.DataSubentro.Valid {
		return comuneI.DataSubentro.Time.After(comuneJ.DataSubentro.Time)

	} else {
		return comuneI.DataPresubentro.Time.After(comuneJ.DataPresubentro.Time)

	}

}

//JSON STRUCTURES
type FornitoreStats struct {
	NumeroComuniSubentro    int `json:"-"`
	NumeroComuniPresubentro int `json:"-"`
	NumeroComuniInattivi    int `json:"-"`
}
type FornitoreSums struct {
	PercentualeComuniSubentro    float64 `json:"percentuale_comuni_subentrati"`
	PercentualeComuniPreSubentro float64 `json:"percentuale_comuni_in_presubentro"`
	PercentualeComuniInattivi    float64 `json:"percentuale_comuni_inattivi"`
	Nome                         string  `json:"nome"`
	//Convenience variable used to calculate the aggregate value avoiding a second loop

}
type DailyStats struct {
	Date            time.Time `json:"date"` // "2016-10-21T00:00:00+00:00",
	Comuni          int       `json:"comuni"`
	Popolazione     int       `json:"popolazione"`
	PopolazioneAire int       `json:"popolazione_aire"`
}

func ConvertFromStat(name string, stats FornitoreStats) FornitoreSums {
	totale := float64(stats.NumeroComuniInattivi + stats.NumeroComuniPresubentro + stats.NumeroComuniSubentro)
	percentualeSubentrato := round((float64(stats.NumeroComuniSubentro) / totale) * 100.00)
	percentualePresubentro := round((float64(stats.NumeroComuniPresubentro) / totale) * 100)
	percentualePercentualeInattivi := round((float64(stats.NumeroComuniInattivi) / totale) * 100)
	return FornitoreSums{
		percentualeSubentrato,
		percentualePresubentro,
		percentualePercentualeInattivi,
		name,
	}
}

type Summaries struct {
	ComuniSubentro             int `json:"com_sub"`
	PopolazioneSubentro        int `json:"pop_sub"`
	ComuniPreSubentro          int `json:"com_pre"`
	PopolazionePresubentro     int `json:"pop_pre"`
	PopolazioneAireSubentro    int `json:"pop_aire"`
	PopolazioneAirePreSubentro int `json:"pop_pre_aire"`
}
type DashBoardData struct {
	LastDateTime time.Time                  `json:"lastDateTime"` // "2016-10-21T00:00:00+00:00",
	Geojson      *geojson.FeatureCollection `json:"geojson"`
	Summaries    *Summaries                 `json:"summaries"`
	Fornitori    []FornitoreSums            `json:"fornitori"`
	Charts       Charts                     `json:"charts"`
}
type Charts struct {
	Subentro    []DailyStats `json:"subentro"`
	Presubentro []DailyStats `json:"presubentro"`
}

/**
Privare Methods
**/
func round(x float64) float64 {

	return math.Round(x)
}
func dateFormattedOrNull(nullTime null.Time) string {
	if !nullTime.Valid {
		return ""
	}
	return nullTime.Time.Format("02/01/2006")
}

func updateFornitoreStat(fornitoriMap *map[string]FornitoreStats, fornitoreName string, subentrati int, presubentrati int, inattivi int) {
	oldEntry, keyExists := (*fornitoriMap)[fornitoreName]
	if keyExists {
		(*fornitoriMap)[fornitoreName] = FornitoreStats{
			oldEntry.NumeroComuniSubentro + subentrati,
			oldEntry.NumeroComuniPresubentro + presubentrati,
			oldEntry.NumeroComuniInattivi + inattivi,
		}
	} else {
		(*fornitoriMap)[fornitoreName] = FornitoreStats{
			subentrati,
			presubentrati,
			inattivi,
		}
	}

}
func addDateToFeatureIfNotEmpty(fc *geojson.Feature, propertyName string, aNullTime null.Time) {
	dateString := dateFormattedOrNull(aNullTime)
	if dateString != "" {
		(*fc).SetProperty(propertyName, dateString)
	}

}

//Convert ad object of time sqlite.Comune to
//<a href="https://tools.ietf.org/html/rfc7946">geoison.Feature</a>
func toFeature(comune sqlite.Comune) *geojson.Feature {
	//get the provincie map singleton
	provincie := *GetProvincieMapInstance()
	feature := geojson.NewPointFeature([]float64{comune.Lon, comune.Lat})
	feature.SetProperty("label", comune.Name)
	feature.SetProperty("PROVINCIA", provincie.Map[comune.Province].ProvinciaExt)
	feature.SetProperty("REGIONE", comune.Region)
	//TODO: Add zone mapping
	feature.SetProperty("ZONA", provincie.Map[comune.Province].Zona)
	feature.SetProperty("popolazione", comune.Population)
	feature.SetProperty("popolazione_aire", comune.PopulationAIRE)
	if comune.DataSubentro.Valid {
		addDateToFeatureIfNotEmpty(feature, "data_subentro", comune.DataSubentro)
	} else {
		//log.Printf("subentro invalid %s", comune.Name)
		addDateToFeatureIfNotEmpty(feature, "data_presubentro", comune.DataPresubentro)
		addDateToFeatureIfNotEmpty(feature, "prima_data_subentro", comune.Subentro.From)
		addDateToFeatureIfNotEmpty(feature, "ultima_data_subentro", comune.Subentro.To)
		addDateToFeatureIfNotEmpty(feature, "data_subentro_preferita", comune.Subentro.PreferredDate)
	}
	return feature
}

//Produce a stack of DailyStats containing the number of Comuni migrated per day
//This stack is garanteed to be ordered by date ascending
func (charts *Charts) pushDailyStats(comune sqlite.Comune, dateTime time.Time, subentro bool) {
	var sl *[]DailyStats
	if subentro {
		sl = &charts.Subentro
	} else {
		sl = &charts.Presubentro
	}

	length := len(*sl)
	//builds a new entry of the histogram of dates - initialized with 1 comune
	var newEntry = DailyStats{dateTime, 1, comune.Population, comune.PopulationAIRE}
	if length == 0 {
		*sl = append(*sl, newEntry)
		return
	}
	d := (*sl)[0]
	newStat := DailyStats{dateTime, d.Comuni + 1, (d.Popolazione + comune.Population), (d.PopolazioneAire + comune.PopulationAIRE)}

	if d.Date == dateTime {
		(*sl)[0] = newStat //Update the stats at the top
	} else {
		*sl = append([]DailyStats{newEntry}, (*sl)...) //Update the stack adding a new node on top and adding the queue
	}

}

//Update the given Chart in order to have the historical series (each following entry has owns the previous data)
func (charts *Charts) updateHistoricalSequence() {
	for c := range charts.Subentro {
		if c > 0 {
			charts.Subentro[c].Comuni = charts.Subentro[c-1].Comuni + charts.Subentro[c].Comuni
			charts.Subentro[c].Popolazione = charts.Subentro[c-1].Popolazione + charts.Subentro[c].Popolazione
			charts.Subentro[c].PopolazioneAire = charts.Subentro[c-1].PopolazioneAire + charts.Subentro[c].PopolazioneAire

		}
	}
	for c := range charts.Presubentro {
		if c > 0 {
			charts.Presubentro[c].Comuni = charts.Presubentro[c-1].Comuni + charts.Presubentro[c].Comuni
			charts.Presubentro[c].Popolazione = charts.Presubentro[c-1].Popolazione + charts.Presubentro[c].Popolazione
			charts.Presubentro[c].PopolazioneAire = charts.Presubentro[c-1].PopolazioneAire + charts.Presubentro[c].PopolazioneAire

		}
	}
}
func GetDashBoardData(comuni []sqlite.Comune) *DashBoardData {
	dashboardData := DashBoardData{}
	lastDateTime := time.Now()
	charts := Charts{}
	summaries := Summaries{}
	fornitoriMap := make(map[string]FornitoreStats)

	fc := geojson.NewFeatureCollection()

	subentrati := &ComuniHeap{}
	preSubentrati := &ComuniHeap{}
	heap.Init(subentrati)
	heap.Init(preSubentrati)
	//Array of inactives
	var inattivi []sqlite.Comune
	//Populate the 2 heaps, one for comuni subentrati and the other for comuni in presubentro
	for _, comune := range comuni {

		if comune.DataSubentro.Valid {
			heap.Push(subentrati, comune)
			continue
		}

		if comune.DataPresubentro.Valid {
			heap.Push(preSubentrati, comune)
			continue
		}
		inattivi = append(inattivi, comune)
	}
	//Iterates over the presubentrati and subentrati heap
	for preSubentrati.Len() > 0 {
		comune := heap.Pop(preSubentrati).(sqlite.Comune)
		feature := toFeature(comune)

		summaries.ComuniPreSubentro = summaries.ComuniPreSubentro + 1
		summaries.PopolazionePresubentro = summaries.PopolazionePresubentro + comune.Population
		summaries.PopolazioneAirePreSubentro = summaries.PopolazioneAirePreSubentro + comune.PopulationAIRE

		charts.pushDailyStats(comune, comune.DataPresubentro.Time, false)
		fc.AddFeature(feature)
		updateFornitoreStat(&fornitoriMap, comune.Fornitore.Name, 0, 1, 0)

	}
	for subentrati.Len() > 0 {
		comune := heap.Pop(subentrati).(sqlite.Comune)

		feature := toFeature(comune)

		summaries.ComuniSubentro = summaries.ComuniSubentro + 1
		summaries.PopolazioneSubentro = summaries.PopolazioneSubentro + comune.Population
		summaries.PopolazioneAireSubentro = summaries.PopolazioneAireSubentro + comune.PopulationAIRE

		charts.pushDailyStats(comune, comune.DataSubentro.Time, true)
		updateFornitoreStat(&fornitoriMap, comune.Fornitore.Name, 1, 0, 0)

		fc.AddFeature(feature)
	}
	//finally append the inattivi
	for _, v := range inattivi {
		fc.AddFeature(toFeature(v))
		updateFornitoreStat(&fornitoriMap, v.Fornitore.Name, 0, 0, 1)
	}
	dashboardData.Geojson = fc
	dashboardData.Summaries = &summaries
	var fornitori []FornitoreSums
	for k, v := range fornitoriMap {
		fornitori = append(fornitori, ConvertFromStat(k, v))
	}
	//Order Fornitori by percentage of Comuni Subentrati
	sort.Slice(fornitori, func(i, j int) bool {
		return fornitori[i].PercentualeComuniSubentro > fornitori[j].PercentualeComuniSubentro
	})
	//Update the historical increasing sequence
	charts.updateHistoricalSequence()

	dashboardData.LastDateTime = lastDateTime
	dashboardData.Fornitori = fornitori
	dashboardData.Charts = charts

	return &dashboardData
}
