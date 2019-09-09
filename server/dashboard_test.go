package main

import (
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

var comuni = []sqlite.Comune{
	{1, "00145", "Comune 1", "Marche", "AP", 1111122, 11111, null.IntFrom(10), 45.3595112, 11.7890789, fornitori[0], sqlite.Responsible{"Donald", "Duck", "+39111111", "+39348111", "donald.duck@test.it"}, sqlite.Indirizzo{"val gardena", "6453", "11", "comune1@pec.it"}, sqlite.Subentro{},
		//Data Subentro:11/01/2019
		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.BoolFrom(true),
		null.IntFrom(1),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.IntFrom(1),
		null.BoolFrom(true),
		null.BoolFrom(true),
		null.IntFrom(10),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.TimeFromPtr(nil),
		null.StringFromPtr(nil),
		null.StringFromPtr(nil),
	},
	{2, "00146", "Comune 2", "Marche", "AP", 1111122, 1111, null.IntFrom(5), 46.3595112, 12.7890789, fornitori[1], sqlite.Responsible{"Donald", "Duck", "+39111111", "+39348111", "donald.duck@test.it"}, sqlite.Indirizzo{"val di fassa", "6455", "12", "comune2@pec.it"}, sqlite.Subentro{},

		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.BoolFrom(true),
		null.IntFrom(1),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.IntFrom(1),
		null.BoolFrom(true),
		null.BoolFrom(true),
		null.IntFrom(15),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.NewTime(time.Unix(1547164800, 0), true),
		null.TimeFromPtr(nil),
		null.StringFromPtr(nil),
		null.StringFromPtr(nil),
	},
}

func TestDateFormatting(t *testing.T) {
	InitServerConfigFromFile("./tools/config.sample.yaml")
	var date = dateFormattedOrNull(comuni[0].DataSubentro)
	assert.Equal(t, "11/01/2019", date)
}
func TestGetGetDashBoardData(t *testing.T) {
	InitServerConfigFromFile("./tools/config.sample.yaml")
	var dashboardData = GetDashBoardData(comuni)
	assert.NotEmpty(t, dashboardData.Charts)
	assert.NotEmpty(t, dashboardData.Fornitori)
	assert.NotEmpty(t, dashboardData.Geojson)
	//TODO create more tests

}
