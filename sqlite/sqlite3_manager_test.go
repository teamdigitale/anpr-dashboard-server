package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"gopkg.in/guregu/null.v3"
)

const testDB = "testSubentroDB.db"

var fornitori = []Fornitore{
	Fornitore{1, "Fornitore 1", "www.fornitore1.it", "fornitore1@mondo.it"},
	Fornitore{2, "Fornitore 2", "www.fornitore2.it", "fornitore2@mondo.it"},
}

var comuni = []Comune{
	{1, "00145", "Comune 1", "Marche", "AP", 1111122, 11111, null.IntFrom(10), 45.3595112, 11.7890789, fornitori[0], Responsible{"Donald", "Duck", "+39111111", "+39348111", "donald.duck@test.it"}, Indirizzo{"val gardena", "6453", "11", "comune1@pec.it"}, Subentro{},
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
	{2, "00146", "Comune 2", "Marche", "AP", 1111122, 1111, null.IntFrom(5), 46.3595112, 12.7890789, fornitori[1], Responsible{"Donald", "Duck", "+39111111", "+39348111", "donald.duck@test.it"}, Indirizzo{"val di fassa", "6455", "12", "comune2@pec.it"}, Subentro{},

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

var comuniInattivi = []Comune{
	{1, "00145", "Comune 3", "Marche", "AP", 1111122, 11111, null.IntFrom(10), 45.3595112, 11.7890789, fornitori[0], Responsible{"Donald", "Duck", "+39111111", "+39348111", "donald.duck@test.it"}, Indirizzo{"val gardena", "6453", "11", "comune1@pec.it"}, Subentro{},
		//Data Subentro:11/01/2019
		null.TimeFromPtr(nil),
		null.TimeFromPtr(nil),
		null.TimeFromPtr(nil),
		null.BoolFrom(true),
		null.IntFrom(1),
		null.TimeFromPtr(nil),
		null.IntFrom(1),
		null.BoolFrom(true),
		null.BoolFrom(true),
		null.IntFrom(10),
		null.TimeFromPtr(nil),
		null.TimeFromPtr(nil),
		null.TimeFromPtr(nil),
		null.StringFromPtr(nil),
		null.StringFromPtr(nil),
	},
}

func checkError(e error) {
	if e != nil {
		log.Fatal().Err(e)

	}

}
func createTestDB() *sql.DB {
	var db = OpenDB(testDB)
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	tx.Exec(kDatabaseSchema)

	tx.Commit()
	return db
}
func removeDB() {
	checkError(os.Remove(testDB))
}
func TestInsertComuni(t *testing.T) {
	var db = createTestDB()

	InsertComuni(db, comuni)
	assert.Equal(t, len(comuni), getTableCount(db, "COMUNE"))
	db.Close()
	removeDB()
}
func TestInsertFornitori(t *testing.T) {
	var db = createTestDB()
	InsertFornitori(db, fornitori)
	assert.Equal(t, len(fornitori), getTableCount(db, "FORNITORE"))
	db.Close()
	removeDB()
}

func TestSearchComuni(t *testing.T) {
	var db = createTestDB()

	InsertFornitori(db, fornitori)
	InsertComuni(db, comuni)

	searchFilter := SearchFilter{
		Fornitore: Fornitore{
			Id: 1,
		},
	}

	var comuni = SearchComuni(db, searchFilter)
	assert.Equal(t, 1, len(comuni))
	assert.Equal(t, "Comune 1", comuni[0].Name)
	assert.Equal(t, "Fornitore 1", comuni[0].Fornitore.Name)
	assert.True(t, comuni[0].DataSubentro.Valid, "Data Subentro is not valid")
	assert.True(t, comuni[0].DataPresubentro.Valid, "Data PreSubentro is not valid")

	assert.Equal(t, null.IntFrom(10), comuni[0].SCConsegnate)
	//Search for id Comune

	var comuniById = SearchComuni(db, SearchFilter{
		Comune: Comune{
			Id: 1,
		},
	})
	assert.Equal(t, 1, len(comuniById))
	assert.Equal(t, "Comune 1", comuni[0].Name)
	assert.Equal(t, "Fornitore 1", comuni[0].Fornitore.Name)
	assert.True(t, comuni[0].IPProvenienza.Bool, "Ip Provenienza should be binded")
	assert.True(t, comuni[0].EmailPec.Bool, "EmailPec should be binded")
	assert.True(t, comuni[0].DataRitiroSm.Valid, "Data Ritiro SM is not valid")

	db.Close()
	removeDB()

}
func TestSearchComuniWithFilter(t *testing.T) {
	var db = createTestDB()

	InsertFornitori(db, fornitori)
	InsertComuni(db, comuniInattivi)

	var inactives = SearchComuni(db, SearchFilter{
		Exclusion: &Exclusion{
			4,
		},
	})
	assert.Empty(t, inactives)
}
func TestSearchSubentro(t *testing.T) {

	var db = createTestDB()

	InsertFornitori(db, fornitori)
	InsertComuni(db, comuni)

	searchFilter := SearchFilter{
		Comune: Comune{
			Subentro: Subentro{
				From:          null.NewTime(time.Unix(1547164800, 0), true),
				To:            null.NewTime(time.Unix(1547251200, 0), true),
				PreferredDate: null.NewTime(time.Unix(1515715200, 0), true),
			},
		},
	}

	var comuni []Comune = SearchComuni(db, searchFilter)
	fmt.Printf("found %d comni", len(comuni))
	db.Close()

	removeDB()
}

func TestSaveSubentro(t *testing.T) {
	var db = createTestDB()
	var comune = comuni[0]
	comune.Subentro = Subentro{

		From:          null.NewTime(time.Unix(1547164800, 0), true),
		To:            null.NewTime(time.Unix(1547251200, 0), true),
		PreferredDate: null.NewTime(time.Unix(1515715200, 0), true),
		IP:            null.StringFrom("1.1.1.1"),
	}

	var ti = time.Unix(1547164800, 0)
	fmt.Println(ti)
	InsertComuni(db, comuni)
	InsertFornitori(db, fornitori)
	SaveOrUpdateSubentro(db, comune)
	searchFilter := SearchFilter{
		Fornitore: Fornitore{
			Id: 1,
		},
	}
	var comuniUpdated = SearchComuni(db, searchFilter)
	//log.Printf("URL %s", comuniUpdated[0].Subentro.IP.String)
	assert.NotEmpty(t, comuniUpdated[0].Subentro.IP.String)
	db.Close()

	removeDB()
}

func TestMarshaller(t *testing.T) {

	var aTime = null.NewTime(time.Unix(1547164800, 0), true)

	jsonBytes, error := json.Marshal(aTime)
	if error != nil {
		log.Fatal().Err(error)
	}

	var nullTime null.Time
	error = json.Unmarshal([]byte("{\"Time\":\"2017-09-14T06:06:51.018Z\",\"Valid\":true}"), &nullTime)
	if error != nil {
		log.Fatal().Err(error)
	}
	log.Print(nullTime)
	assert.True(t, nullTime.Valid, "Time is not valid")

	jsonBytes, error = json.Marshal(comuni[0])
	if error != nil {
		log.Fatal().Err(error)
	}
	log.Printf("Marshal %s", string(jsonBytes))

	//MARSHAL A comune
	var aJsonString = "{\"Id\":4,\"Name\":\"ABBADIA SAN SALVATORE\",\"Region\":\"\",\"Province\":\"\",\"Population\":0,\"Pec\":\"\",\"Lat\":42.8809992,\"Lon\":11.6773601,\"Fornitore\":{\"id\":4,\"Name\":\"MAGGIOLI\",\"Url\":\"\"},\"Responsible\":{\"Name\":\"\",\"Surname\":\"\",\"Phone\":\"\",\"Mobile\":\"\",\"Email\":\"\"},\"Subentro\":{\"From\":{\"Time\":\"2017-09-14T07:27:41.175Z\",\"Valid\":true},\"To\":{\"Time\":\"2017-09-14T07:27:41.175Z\",\"Valid\":true},\"PreferredDate\":{\"Time\":\"2017-09-14T07:27:41.175Z\",\"Valid\":true}}}"
	var comune Comune
	error = json.Unmarshal([]byte(aJsonString), &comune)
	if error != nil {
		log.Fatal().Err(error)
	}
	log.Print(comune.Subentro.From)
}

func getTableCount(db *sql.DB, table string) int {
	rows, err := db.Query("select count(*) from " + table)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer rows.Close()
	var nResult int = 0
	rows.Next()
	rows.Scan(&nResult)

	return nResult
}
func TestSaveComment(t *testing.T) {
	var db = createTestDB()
	var comune = comuni[0]
	var comment = Comment{

		Comune: comune,
		Author: "mirko@teamdigitale.governo.it",

		Date:    null.NewTime(time.Unix(1515715200, 0), true),
		Content: "A comment created on the Dashboard",
	}

	InsertComuni(db, comuni)
	SaveOrUpdateComment(db, comment)
	var comments = SearchComment(db, comune)
	fmt.Print(comments[0])
	db.Close()

	removeDB()
}
func TestUpdateComuneFornitore(t *testing.T) {

	var db = createTestDB()

	InsertFornitori(db, fornitori)
	InsertComuni(db, comuni)

	codiceIstat := comuni[0].CodiceIstat
	assert.Equal(t, comuni[0].Fornitore.Id, fornitori[0].Id)
	log.Print(codiceIstat)

	UpdateComuneFornitore(db, codiceIstat, fornitori[1].Id)

	searchFilter := SearchFilter{
		Comune: Comune{
			CodiceIstat: codiceIstat,
		},
	}
	var comuni_updated []Comune = SearchComuni(db, searchFilter)
	assert.Equal(t, comuni_updated[0].Fornitore.Id, fornitori[1].Id)

	db.Close()

	removeDB()
}
func TestUpdateSubentroDateForComune(t *testing.T) {
	var db = createTestDB()

	InsertComuni(db, comuni)

	//var comune = Comune{}
	comuni[0].DataSubentro = null.NewTime(time.Unix(1510417710, 0), true)
	UpdateComuneCheckListDate(db, []Comune{comuni[0]})
	var date int64
	db.QueryRow("SELECT DATA_SUBENTRO FROM COMUNE WHERE CODICE_ISTAT=?", comuni[0].CodiceIstat).Scan(&date)
	log.Print("Date", date)
	log.Print(time.Unix(date, 0).Date())
	assert.NotNil(t, date)
	var dateFromDb = time.Unix(date, 0)
	assert.Equal(t, time.Month(11), dateFromDb.Month())
	assert.Equal(t, 11, dateFromDb.Day())
	assert.Equal(t, 2017, dateFromDb.Year())
	removeDB()

}
