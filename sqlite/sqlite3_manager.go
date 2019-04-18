package sqlite

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gopkg.in/guregu/null.v3"
)

// internal conts
const (
	PROACTIVE       = "ProactiveEmailAlert"
	REACTIVE        = "ReactiveEmailAlert"
	kDatabaseSchema = `
BEGIN TRANSACTION;

CREATE TABLE IF NOT EXISTS SUBENTRO (
 ID_COMUNE INTEGER NOT NULL,
 RANGE_FROM INTEGER NOT NULL,
 RANGE_TO INTEGER NOT NULL,
 FINAL_DATE INTEGER,
 IP TEXT,
 FOREIGN KEY(ID_COMUNE) REFERENCES COMUNE(ID)
);

CREATE TABLE IF NOT EXISTS FORNITORE (
 ID INTEGER PRIMARY KEY AUTOINCREMENT,
 NAME TEXT,
 URL TEXT
);

CREATE TABLE IF NOT EXISTS COMUNE (
 ID INTEGER PRIMARY KEY AUTOINCREMENT,
 ID_FORNITORE INTEGER,
 NAME TEXT NOT NULL,
 PROVINCIA TEXT NOT NULL,
 REGION TEXT NOT NULL,
 CODICE_ISTAT TEXT,
 POPOLAZIONE INT NOT NULL,
 POPOLAZIONE_AIRE INT NOT NULL,
 DATA_SUBENTRO INT,
 POSTAZIONI INT,
 INDIRIZZO_VIA TEXT,
 INDIRIZZO_CAP TEXT,
 INDIRIZZO_CIVICO TEXT,
 PEC_COMUNE TEXT NULL,
 LAT REAL NOT NULL,
 LON REAL NOT NULL,
 NOME_REFERENTE TEXT,
 COGNOME_REFERENTE TEXT,
 TELEFONO_REFERENTE TEXT,
 CELLULARE_REFERENTE TEXT,
 EMAIL_REFERENTE TEXT,
 DATA_ABILITAZIONE_TEST INT,
 DATA_PRESUBENTRO INT,
 ABILITAZIONE_PREFETTURA BOOLEAN,
 UTENTI_ABILITATI  INT,
 DATA_CONSEGNA_SC INT,
 NUMERO_LETTORI INT,
 IPPROVENIENZA BOOLEAN,
 EMAILPEC BOOLEAN,
 SC_CONSEGNATE INT,
 DATA_RITIRO_SC INT,
 DATA_PRIMO_PRESUBENTRO INT,

 FOREIGN KEY(ID_FORNITORE) REFERENCES FORNITORE(ID)
);

CREATE TABLE IF NOT EXISTS COMMENTO (
 ID INTEGER PRIMARY KEY AUTOINCREMENT,
 ID_COMUNE INTEGER,
 AUTHOR TEXT NOT NULL,
 DATE INTEGER NOT NULL,
 CONTENT TEXT NOT NULL,
 FOREIGN KEY(ID_COMUNE) REFERENCES ID_COMUNE(ID)
);
COMMIT;
`
	k_dateFormat = "02/01/2006"
)

type OrderType int
type ExclusionType int

const (
	FornitoreName OrderType = iota
	DateFinal
	Population
	SubentroFrom
)

const (
	ExcludeWithoutFinalDate ExclusionType = iota
	ExcludeWithoutSubentro
	ExcludeAlreadyWithSubentro
)

type ComuneFornitore struct {
	CodiceIstat string `json:"CodiceIstat"`
	FornitoreID int    `json:"FornitoreId"`
}
type AnomalieSearchFilter struct {
	CodiceIstat string     `json:"CodiceIstat"`
	Order       *Order     `json:"order"`
	Exclusion   *Exclusion `json:"exclusion"`
}
type SearchFilter struct {
	Fornitore Fornitore  `json:"fornitore"`
	Comune    Comune     `json:"comune"`
	Order     *Order     `json:"order"`
	Exclusion *Exclusion `json:"exclusion"`
}
type Indirizzo struct {
	Via    string
	Cap    string
	Civico string
	Pec    string
}
type Responsible struct {
	Name    string
	Surname string
	Phone   string
	Mobile  string
	Email   string
}
type Fornitore struct {
	Id    int `json:"id"`
	Name  string
	Url   string
	eMail string
}
type Order struct {
	OrderType OrderType
}
type Exclusion struct {
	ExclusionType ExclusionType
}

type Anomalie struct {
	Description   string
	CodiceIstat   string
	Name          string
	Population    int
	Code          string
	Number        int
	ClassAnomalia string
	TipoAnomalia  string
}

//A base structure for Comune including basic contact information
type Comune struct {
	Id          int
	CodiceIstat string
	Name        string
	Region      string // Non esistente in schede monitoraggio.
	Province    string

	Population             int
	PopulationAIRE         int
	Postazioni             null.Int
	Lat                    float64
	Lon                    float64
	Fornitore              Fornitore
	Responsible            Responsible `json:"-"`
	Indirizzo              Indirizzo   `json:"-"`
	Subentro               Subentro
	DataSubentro           null.Time
	DataAbilitazione       null.Time
	DataPresubentro        null.Time
	AbilitazionePrefettura null.Bool
	UtentiAbilitati        null.Int
	DataConsegnaSm         null.Time
	NumeroLettori          null.Int
	IPProvenienza          null.Bool
	EmailPec               null.Bool
	SCConsegnate           null.Int
	DataRitiroSm           null.Time
	DataPrimoPresubentro   null.Time
}

//Holds just the basic information about a Comue
type ComuneInfo struct {
	CodiceIstat                      string
	Name                             string
	DataSubentro                     null.Time
	DataAbilitazione                 null.Time
	DataPresubentro                  null.Time
	PianificazioneIntervalloSubentro Subentro
}
type Comment struct {
	Id      int
	Comune  Comune
	Date    null.Time
	Author  string
	Content string
}
type Alerting struct {
	Name           string
	FornitoreName  string
	FornitoreEmail string
	DateFrom       string
	DateTo         string
}

type SubentroInfo struct {
	DateFinal  null.Time
	Comuni     null.Int
	Population null.Int
}

//Convert a Comune into ComuneInfo holding just the information needed in order to show at the frontend
func (c *Comune) ToComuneInfo() ComuneInfo {
	info := ComuneInfo{
		CodiceIstat:                      c.CodiceIstat,
		Name:                             c.Name,
		DataSubentro:                     c.DataSubentro,
		DataPresubentro:                  c.DataPresubentro,
		DataAbilitazione:                 c.DataAbilitazione,
		PianificazioneIntervalloSubentro: c.Subentro,
	}
	return info

}
func (c *Comment) AsString() string {
	return fmt.Sprintf("Comune: %s (%s)\nAuthor: %s\nComment: %s",
		c.Comune.Name, c.Comune.CodiceIstat, c.Author, c.Content)
}

func NewComuneFromCSVRecord(csv_record []string) (*Comune, error) {
	msgs := []string{}

	if len(csv_record) != 6 {
		msgs = append(msgs, "ERRORE: numero colonne deve essere 6")
		return nil, errors.New(strings.Join(msgs, "\n"))
	}

	id, err := strconv.ParseInt(csv_record[0], 10, 64)
	if err != nil {
		msgs = append(msgs, err.Error())
	}

	codice_istat := csv_record[1]
	name := csv_record[2]
	var subentro Subentro

	if csv_record[3] != "" {
		from, err := time.Parse(k_dateFormat, csv_record[3])
		if err != nil {
			msgs = append(msgs, err.Error())
		} else {
			subentro.From = null.TimeFrom(from)
		}
	}
	if csv_record[4] != "" {
		to, err := time.Parse(k_dateFormat, csv_record[4])
		if err != nil {
			msgs = append(msgs, err.Error())
		} else {
			subentro.To = null.TimeFrom(to)
		}
	}
	if csv_record[5] != "" {
		preferred, err := time.Parse(k_dateFormat, csv_record[5])
		if err != nil {
			msgs = append(msgs, err.Error())
		} else {
			subentro.PreferredDate = null.TimeFrom(preferred)
		}
	}
	comune := Comune{
		Id:          int(id),
		CodiceIstat: codice_istat,
		Name:        name,
		Subentro:    subentro,
	}

	if len(msgs) == 0 {
		return &comune, nil
	}

	return nil, errors.New(strings.Join(msgs, "\n"))
}

func FormatIfNotEmpty(anullTime null.Time) string {
	if anullTime.Valid {
		return anullTime.Time.Format("02/01/2006")
	} else {
		return ""
	}

}

func (c *Comune) AsCSVRecord() []string {
	return []string{
		fmt.Sprintf("%d", c.Id),
		c.CodiceIstat,
		c.Name,
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.From)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.To)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.PreferredDate)),
	}
}
func (c *Comune) AsCSVRecordForNotify() []string {
	return []string{
		fmt.Sprintf("%d", c.Id),
		c.CodiceIstat,
		c.Name,
		c.Fornitore.Name,
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.From)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.To)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.PreferredDate)),
		//fmt.Sprintf("%s", c.Subentro.IP.String),
	}
}

func (c *Comune) AsDetailedCSVRecord() []string {
	return []string{
		fmt.Sprintf("%d", c.Id),
		c.CodiceIstat,
		c.Name,
		c.Province,
		c.Region,
		fmt.Sprintf("%d", c.Population),
		fmt.Sprintf("%d", c.PopulationAIRE),

		fmt.Sprintf("%d", c.Postazioni.Int64),
		fmt.Sprintf("%f", c.Lat),
		fmt.Sprintf("%f", c.Lon),
		c.Fornitore.Name,
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.From)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.To)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.Subentro.PreferredDate)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.DataConsegnaSm)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.DataRitiroSm)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.DataSubentro)),
		//fmt.Sprintf("%s", c.Subentro.IP),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.DataPrimoPresubentro)),
		fmt.Sprintf("%s", FormatIfNotEmpty(c.DataPresubentro)),
	}
}

type Subentro struct {
	From          null.Time
	To            null.Time
	PreferredDate null.Time
	IP            null.String
}

//HasValidDates one date is valid
func (s *Subentro) HasValidDates() bool {
	if s.From.Valid || s.To.Valid || s.PreferredDate.Valid {
		return true
	}
	return false
}

func sameDate(a time.Time, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

//IsWellFormed if the date interval is consitent
func (s *Subentro) IsWellFormed(comune *Comune) (bool, error) {

	msgs := []string{}
	old := comune.Subentro
	if s.To.Time.Before(s.From.Time) {
		msgs = append(msgs, "data iniziale deve essere minore di data finale")
	}
	if s.PreferredDate.Valid && s.PreferredDate.Time.Before(s.From.Time) {
		msgs = append(msgs, "data iniziale deve essere minore di data preferita")
	}
	if s.PreferredDate.Valid && s.To.Time.Before(s.PreferredDate.Time) {
		msgs = append(msgs, "data finale deve essere maggiore di data preferita")
	}

	if !sameDate(s.From.Time, old.From.Time) && s.From.Time.Before(time.Now()) {
		msgs = append(msgs, "data iniziale non può essere nel passato")
	}
	//These checks are valid for new items
	isPresentAndAnticipated := old.From.Valid && !sameDate(s.From.Time, old.From.Time) && s.PreferredDate.Valid && s.PreferredDate.Time.Before(old.PreferredDate.Time)
	isNew := !old.From.Valid
	fiveDays := time.Duration(5 * 24 * time.Hour)
	isNewOrAnticipated := isNew || isPresentAndAnticipated
	if isNewOrAnticipated && comune.DataPresubentro.Valid && comune.DataConsegnaSm.Valid && s.From.Time.Before(time.Now().Add(fiveDays)) {
		//!sameDate(s.From.Time, old.From.Time)
		msgs = append(msgs, "La data iniziale deve essere ad almeno 5 giorni da data odierna in caso di presubentro avvenuto e smartcard controllate")
	}
	twentyDays := time.Duration(20 * 24 * time.Hour)
	if isNewOrAnticipated && !comune.DataPresubentro.Valid && !sameDate(s.From.Time, old.From.Time) && s.From.Time.Before(time.Now().Add(twentyDays)) {
		msgs = append(msgs, "La data iniziale deve sempre ad almeno 20 giorni da data odierna in caso di mancato presubentro")
	}
	//
	thirtyDays := time.Duration(30 * 24 * time.Hour)
	if s.From.Time.Add(thirtyDays).Before(s.To.Time) {
		msgs = append(msgs, "il range dell'intervallo non può essere superiore a 30 giorni")
	}

	if len(msgs) == 0 {
		return true, nil
	}

	return false, errors.New(strings.Join(msgs, ", "))
}

func OpenDB(dbName string) *sql.DB {

	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		panic(err)
	}
	return db
}

func InitDB(db *sql.DB) {
	db.Exec(kDatabaseSchema)
}

func SaveOrUpdateSubentro(db *sql.DB, comune Comune) {
	//Check if a row EXISTS
	var subentro = comune.Subentro
	var sqlSelect = fmt.Sprintf("SELECT COUNT(*) FROM SUBENTRO WHERE  ID_COMUNE=%d", comune.Id)

	var saveOrUpdateSQL string

	if !subentro.From.Valid || !subentro.To.Valid {
		log.Printf("Passed an invalid Subentro date from:%v to:%v, ignore", subentro.From, subentro.To)
		return

	}

	if IsPresent(db, sqlSelect) {
		if subentro.PreferredDate.Valid {
			saveOrUpdateSQL = fmt.Sprintf("UPDATE SUBENTRO SET RANGE_FROM=%d, RANGE_TO=%d, FINAL_DATE=%d, IP='%s' WHERE ID_COMUNE=%d;", subentro.From.Time.Unix(), subentro.To.Time.Unix(), subentro.PreferredDate.Time.Unix(), subentro.IP.String, comune.Id)
		} else {
			saveOrUpdateSQL = fmt.Sprintf("UPDATE SUBENTRO SET RANGE_FROM=%d, FINAL_DATE=NULL, RANGE_TO=%d, IP='%s' WHERE ID_COMUNE=%d;", subentro.From.Time.Unix(), subentro.To.Time.Unix(), subentro.IP.String, comune.Id)
		}
		//Date is not present
	} else {
		if subentro.PreferredDate.Valid {
			saveOrUpdateSQL = fmt.Sprintf("INSERT INTO SUBENTRO (RANGE_FROM,RANGE_TO,FINAL_DATE,IP,ID_COMUNE) VALUES(%d,%d,%d,'%s',%d);", subentro.From.Time.Unix(), subentro.To.Time.Unix(), subentro.PreferredDate.Time.Unix(), subentro.IP.String, comune.Id)
		} else {
			saveOrUpdateSQL = fmt.Sprintf("INSERT INTO SUBENTRO (RANGE_FROM,RANGE_TO,IP,ID_COMUNE) VALUES(%d,%d,'%s',%d);", subentro.From.Time.Unix(), subentro.To.Time.Unix(), subentro.IP.String, comune.Id)
		}
	}

	//log.Print(saveOrUpdateSQL)
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	stmt, err := tx.Prepare(saveOrUpdateSQL)

	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	tx.Commit()

}

// SearchAlert : SQL extract logic to get the alerts to be sent to SWH for multiple cases
// ANPR-80 (refactored ANPR-79 code)
func SearchAlerts(db *sql.DB, alertType string) []Alerting {
	whereClause := "1"
	switch alertType {
	case PROACTIVE:
		whereClause = "SUBENTRO.RANGE_TO BETWEEN strftime('%s') AND strftime('%s')+1036800  AND COMUNE.DATA_SUBENTRO is null AND (((SUBENTRO.RANGE_TO-strftime('%s')) / 86400) % 3 = 0)"
	case REACTIVE:
		whereClause = "SUBENTRO.RANGE_TO < strftime('%s')-86400  AND COMUNE.DATA_SUBENTRO is null"
	}

	var sqlbuffer bytes.Buffer
	sqlbuffer.WriteString(`SELECT 
		COMUNE.NAME,
		FORNITORE.NAME as FORNITORE,
		REFERENTI.EMAIL_REFERENTE as EMAIL,
		date(SUBENTRO.RANGE_FROM,'unixepoch') as DATE_FROM,
		date(SUBENTRO.RANGE_TO,'unixepoch') as DATE_TO
		FROM COMUNE
			INNER JOIN FORNITORE on FORNITORE.ID = COMUNE.ID_FORNITORE
			INNER JOIN SUBENTRO on SUBENTRO.ID_COMUNE = COMUNE.ID
			INNER JOIN REFERENTI on REFERENTI.ID = FORNITORE.ID
		WHERE ` + whereClause + ` 
		ORDER BY COMUNE.NAME;
		`)

	rows, err := db.Query(sqlbuffer.String())
	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s %s", err, sqlbuffer.String()))
	}
	defer rows.Close()
	alerts := []Alerting{}

	for rows.Next() {
		var alerting = Alerting{}

		if err := rows.Scan(&alerting.Name, &alerting.FornitoreName, &alerting.FornitoreEmail, &alerting.DateFrom, &alerting.DateTo); err != nil {
			panic(err)
		}

		alerts = append(alerts, alerting)

	}
	if err := rows.Err(); err != nil {
		panic(err)
	}

	return alerts
}
func SearchAnomalie(db *sql.DB, searchFilter AnomalieSearchFilter) []Anomalie {

	var sqlbuffer bytes.Buffer
	sqlbuffer.WriteString(`SELECT
		an.DESCRIZIONANOMALIA,
		co.CODICE_ISTAT,
		co.NAME,
		co.POPOLAZIONE,
		an.TIPOANOMALIA,
		SUM(an.NUMNEROANOMALIE),
		an.CLASSEANOMALIA
		FROM COMUNE co
	 		INNER JOIN  ANOMALIE an ON an.CODISTAT = co.CODICE_ISTAT WHERE  1
		`)
	var args []interface{}

	if searchFilter.CodiceIstat != "" {
		sqlbuffer.WriteString(" AND co.CODICE_ISTAT = ?")
		args = append(args, searchFilter.CodiceIstat)
	}

	//Exclusion conditios
	var exclusion = " "
	sqlbuffer.WriteString(exclusion)

	//log.Print(sqlbuffer.String())

	var groupString = "an.TIPOANOMALIA"
	sqlbuffer.WriteString(" GROUP BY ")
	sqlbuffer.WriteString(groupString)

	var orderString = "an.CLASSEANOMALIA"
	sqlbuffer.WriteString(" ORDER BY ")
	sqlbuffer.WriteString(orderString)

	sqlString := sqlbuffer.String()

	rows, err := db.Query(sqlString, args...)
	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s %s", err, sqlString))
	}
	defer rows.Close()
	anomalie := []Anomalie{}

	for rows.Next() {
		var anomalia = Anomalie{}

		if err := rows.Scan(&anomalia.Description, &anomalia.CodiceIstat, &anomalia.Name, &anomalia.Population, &anomalia.Code, &anomalia.Number, &anomalia.ClassAnomalia); err != nil {
			panic(err)
		}

		anomalie = append(anomalie, anomalia)

	}
	if err := rows.Err(); err != nil {
		panic(err)
	}

	return anomalie
}

func SearchComuni(db *sql.DB, searchFilter SearchFilter) []Comune {

	var sqlbuffer bytes.Buffer
	sqlbuffer.WriteString(`SELECT
		 co.ID,
		 co.CODICE_ISTAT,
		 co.NAME,
		 co.REGION,
		 co.PROVINCIA,
		 co.POPOLAZIONE,
		 co.POPOLAZIONE_AIRE,
		 co.POSTAZIONI,
		 co.LAT,
		 co.LON,
		 fn.ID,
		 fn.NAME,
		 su.RANGE_FROM,
		 su.RANGE_TO,
		 su.FINAL_DATE,
		 co.DATA_SUBENTRO,
		 co.DATA_ABILITAZIONE_TEST,
		 co.DATA_PRESUBENTRO,
		 co.ABILITAZIONE_PREFETTURA,
		 co.UTENTI_ABILITATI,
		 co.DATA_CONSEGNA_SC,
		 co.NUMERO_LETTORI,
		 co.IPPROVENIENZA,
		 co.EMAILPEC,
		 co.SC_CONSEGNATE,
		 co.DATA_RITIRO_SC,
		 co.DATA_PRIMO_PRESUBENTRO,
		 su.IP

		 FROM COMUNE co
	 		INNER JOIN  FORNITORE fn ON fn.ID = co.ID_FORNITORE
			LEFT OUTER JOIN SUBENTRO su ON co.ID = su.ID_COMUNE WHERE  1
		 `)
	var args []interface{}

	if searchFilter.Fornitore.Id > 0 {
		sqlbuffer.WriteString(" AND co.ID_FORNITORE = ?")
		args = append(args, searchFilter.Fornitore.Id)
	}
	if searchFilter.Fornitore.Name != "" {
		sqlbuffer.WriteString(" AND fn.NAME LIKE '%' || ? || '%'")
		args = append(args, searchFilter.Fornitore.Name)
	}
	if searchFilter.Comune.Id > 0 {
		sqlbuffer.WriteString(" AND co.ID = ?")
		args = append(args, searchFilter.Comune.Id)
	}
	if searchFilter.Comune.Name != "" {
		sqlbuffer.WriteString(" AND co.NAME LIKE '%' || ? || '%'")
		//sqlbuffer.WriteString(fmt.Sprintf("%s", searchFilter.Comune.Name))
		args = append(args, searchFilter.Comune.Name)
	}
	if searchFilter.Comune.CodiceIstat != "" {
		sqlbuffer.WriteString(" AND co.CODICE_ISTAT = ?")
		args = append(args, searchFilter.Comune.CodiceIstat)
	}
	//args = append(args, searchFilter.Comune.Name)
	if searchFilter.Comune.Subentro.From.Valid && searchFilter.Comune.Subentro.To.Valid {
		if searchFilter.Exclusion == nil {
			sqlbuffer.WriteString(" ")

		} else if searchFilter.Exclusion.ExclusionType != 1 {
			sqlbuffer.WriteString(" AND co.DATA_SUBENTRO IS NULL AND( su.RANGE_FROM >= ? AND su.RANGE_TO <= ?)")
			args = append(args, searchFilter.Comune.Subentro.From.Time.Unix(), searchFilter.Comune.Subentro.To.Time.Unix())

		} else {
			sqlbuffer.WriteString(" AND co.DATA_SUBENTRO IS NULL AND( su.FINAL_DATE >= ? AND su.FINAL_DATE <= ?)")
			args = append(args, searchFilter.Comune.Subentro.From.Time.Unix(), searchFilter.Comune.Subentro.To.Time.Unix())

		}

	}
	if searchFilter.Exclusion != nil && searchFilter.Exclusion.ExclusionType == 3 {
		sqlbuffer.WriteString(" AND co.DATA_SUBENTRO IS NULL")

	}

	//Exclusion conditios
	var exclusion = " "

	sqlbuffer.WriteString(exclusion)

	var orderString = "co.NAME"
	if searchFilter.Order != nil {

		if searchFilter.Order.OrderType == 1 {
			orderString = "fn.NAME"
		}
		if searchFilter.Order.OrderType == 2 {
			orderString = "su.FINAL_DATE ASC"
		}
		if searchFilter.Order.OrderType == 3 {
			orderString = "co.POPULATION DESC"
		}
		if searchFilter.Order.OrderType == 4 {
			orderString = "su.RANGE_FROM ASC"
		}

	}
	sqlbuffer.WriteString(" ORDER BY ")
	sqlbuffer.WriteString(orderString)
	//log.Print(sqlbuffer.String())
	rows, err := db.Query(sqlbuffer.String(), args...)

	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s %s", err, sqlbuffer.String()))
	}
	defer rows.Close()
	comuni := []Comune{}
	var rangeFrom sql.NullInt64
	var rangeTo sql.NullInt64
	var selectDate sql.NullInt64
	var dataSubentro sql.NullInt64

	var dataAbilitazione sql.NullInt64
	var dataPresubentro sql.NullInt64
	var dataConsegnaSC sql.NullInt64
	var dataRitiroSC sql.NullInt64
	var dataPrimoPresubentro sql.NullInt64

	for rows.Next() {

		var comune = Comune{}

		if err := rows.Scan(&comune.Id, &comune.CodiceIstat, &comune.Name, &comune.Region, &comune.Province, &comune.Population, &comune.PopulationAIRE, &comune.Postazioni, &comune.Lat, &comune.Lon, &comune.Fornitore.Id, &comune.Fornitore.Name, &rangeFrom, &rangeTo, &selectDate, &dataSubentro, &dataAbilitazione, &dataPresubentro, &comune.AbilitazionePrefettura, &comune.UtentiAbilitati, &dataConsegnaSC, &comune.NumeroLettori, &comune.IPProvenienza, &comune.EmailPec, &comune.SCConsegnate, &dataRitiroSC, &dataPrimoPresubentro, &comune.Subentro.IP); err != nil {
			panic(err)
		}
		if rangeFrom.Valid {

			comune.Subentro.From = null.NewTime(time.Unix(rangeFrom.Int64, 0), rangeFrom.Valid)
			comune.Subentro.To = null.NewTime(time.Unix(rangeTo.Int64, 0), rangeTo.Valid)
			comune.Subentro.PreferredDate = null.NewTime(time.Unix(selectDate.Int64, 0), selectDate.Valid)
		}
		if dataAbilitazione.Valid {
			comune.DataAbilitazione = null.NewTime(time.Unix(dataAbilitazione.Int64, 0), dataAbilitazione.Valid)
		}
		if dataPresubentro.Valid {
			comune.DataPresubentro = null.NewTime(time.Unix(dataPresubentro.Int64, 0), dataPresubentro.Valid)
		}
		if dataConsegnaSC.Valid {
			comune.DataConsegnaSm = null.NewTime(time.Unix(dataConsegnaSC.Int64, 0), dataConsegnaSC.Valid)
		}
		if dataSubentro.Valid {
			comune.DataSubentro = null.NewTime(time.Unix(dataSubentro.Int64, 0), dataSubentro.Valid)
		}

		if dataRitiroSC.Valid {
			comune.DataRitiroSm = null.NewTime(time.Unix(dataRitiroSC.Int64, 0), dataRitiroSC.Valid)
		}

		if dataPrimoPresubentro.Valid {
			comune.DataPrimoPresubentro = null.NewTime(time.Unix(dataPrimoPresubentro.Int64, 0), dataPrimoPresubentro.Valid)
		}
		//dataPrimoPresubentro
		comuni = append(comuni, comune)

	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	fmt.Printf("Query has returned %d comuni", len(comuni))

	return comuni
}

func InsertFornitori(db *sql.DB, fornitori []Fornitore) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	var sqlInsert = "INSERT INTO FORNITORE (NAME, URL) VALUES (?,?)"
	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(fornitori); i++ {

		var fornitore = fornitori[i]
		_, err = stmt.Exec(fornitore.Name, fornitore.Url)

		if err != nil {
			log.Print(err)
		}
	}

	defer stmt.Close()
	tx.Commit()
}
func InsertComuni(db *sql.DB, comuni []Comune) {

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	var sqlInsert = `insert into COMUNE(
			ID_FORNITORE,
			NAME,
			PROVINCIA,
			REGION,
			POPOLAZIONE,
			POPOLAZIONE_AIRE,
			CODICE_ISTAT,
			POSTAZIONI,
			LAT,
			LON,
			NOME_REFERENTE,
			COGNOME_REFERENTE,
			TELEFONO_REFERENTE,
			CELLULARE_REFERENTE,
			EMAIL_REFERENTE,
			INDIRIZZO_VIA,
			INDIRIZZO_CAP,
			INDIRIZZO_CIVICO,
			PEC_COMUNE,
			DATA_SUBENTRO,
			DATA_ABILITAZIONE_TEST,
			DATA_PRESUBENTRO,
			ABILITAZIONE_PREFETTURA,
			UTENTI_ABILITATI,
			DATA_CONSEGNA_SC,
			NUMERO_LETTORI,
			IPPROVENIENZA,
			EMAILPEC,
			SC_CONSEGNATE,
			DATA_RITIRO_SC

			)
				values(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	stmt, err := tx.Prepare(sqlInsert)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(comuni); i++ {
		var comune = comuni[i]
		_, err = stmt.Exec(comune.Fornitore.Id, comune.Name, comune.Province, comune.Region, comune.Population, comune.PopulationAIRE, comune.CodiceIstat, comune.Postazioni, comune.Lat, comune.Lon, comune.Responsible.Name, comune.Responsible.Surname, comune.Responsible.Phone, comune.Responsible.Mobile, comune.Responsible.Email, comune.Indirizzo.Via, comune.Indirizzo.Cap, comune.Indirizzo.Civico, comune.Indirizzo.Pec, comune.DataSubentro.Time.Unix(), comune.DataAbilitazione.Time.Unix(), comune.DataPresubentro.Time.Unix(), comune.AbilitazionePrefettura, comune.UtentiAbilitati, comune.DataConsegnaSm.Time.Unix(), comune.NumeroLettori, comune.IPProvenienza, comune.EmailPec, comune.SCConsegnate, comune.DataRitiroSm.Time.Unix())
		if err != nil {
			log.Print(err)
		}
	}

	defer stmt.Close()
	tx.Commit()
}
func IsPresent(db *sql.DB, sqlSelect string) bool {

	rows, err := db.Query(sqlSelect)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	var nResult int = 0
	rows.Next()
	rows.Scan(&nResult)

	return nResult > 0

}

// to avoid to resend mails on reloads
func SearchComment(db *sql.DB, comune Comune) []Comment {
	_, err := db.Begin()
	if err != nil {
		panic(err)
	}

	rows, err := db.Query("SELECT ID,AUTHOR, DATE, CONTENT FROM COMMENTO WHERE ID_COMUNE = ? ORDER BY DATE DESC", comune.Id)
	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s", err))
	}
	defer rows.Close()
	comments := []Comment{}
	for rows.Next() {
		var comment = Comment{}
		var dateTime int64

		if err := rows.Scan(&comment.Id, &comment.Author, &dateTime, &comment.Content); err != nil {
			panic(err)
		}
		comment.Date = null.NewTime(time.Unix(dateTime, 0), true)

		comments = append(comments, comment)
	}
	return comments
}
func SaveOrUpdateComment(db *sql.DB, comment Comment) {
	tx, err := db.Begin()
	if err != nil {

		panic(err)
	}
	var sqlString = "INSERT INTO COMMENTO (ID_COMUNE,AUTHOR,DATE,CONTENT ) VALUES (?,?,?,?)"
	var isInsert = true
	if comment.Id > 0 {
		sqlString = "UPDATE COMMENTO SET AUTHOR = ?,DATE = ?,CONTENT = ? WHERE ID =?"
		isInsert = false
	}
	stmt, err := tx.Prepare(sqlString)
	if err != nil {
		panic(err)
	}
	if isInsert {
		_, err = stmt.Exec(comment.Comune.Id, comment.Author, comment.Date.Time.Unix(), comment.Content)
	} else {
		_, err = stmt.Exec(comment.Author, comment.Date.Time.Unix(), comment.Content)

	}
	if err != nil {
		log.Print(err)
	}
	defer stmt.Close()
	tx.Commit()
}

func DeleteComment(db *sql.DB, comment Comment) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	stmt, err := tx.Prepare("DELETE FROM COMMENTO WHERE ID =? AND ID_COMUNE=?")
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(comment.Id, comment.Comune.Id)
	if err != nil {
		panic(err)
	}
	defer stmt.Close()
	tx.Commit()

}

func TimeStampOrNull(time null.Time) null.Int {

	if !time.Valid {
		return null.IntFromPtr(nil)
	}

	return null.IntFrom(time.Time.Unix())
}

func UpdateComuneCheckListDate(db *sql.DB, comuni []Comune) {

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare("UPDATE COMUNE SET DATA_SUBENTRO=?,DATA_ABILITAZIONE_TEST=?,DATA_PRESUBENTRO=?,ABILITAZIONE_PREFETTURA=?, UTENTI_ABILITATI=?,DATA_CONSEGNA_SC=?,NUMERO_LETTORI =?,IPPROVENIENZA =?, EMAILPEC=?, SC_CONSEGNATE=?,POSTAZIONI=?, DATA_RITIRO_SC=?, DATA_PRIMO_PRESUBENTRO=?, POPOLAZIONE=?, POPOLAZIONE_AIRE=? WHERE CODICE_ISTAT=?;")
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(comuni); i++ {
		var comune = comuni[i]

		_, err = stmt.Exec(
			TimeStampOrNull(comune.DataSubentro),
			TimeStampOrNull(comune.DataAbilitazione),
			TimeStampOrNull(comune.DataPresubentro),
			comune.AbilitazionePrefettura,
			comune.UtentiAbilitati,
			TimeStampOrNull(comune.DataConsegnaSm),
			comune.NumeroLettori,
			comune.EmailPec,
			comune.IPProvenienza,
			comune.SCConsegnate,
			comune.Postazioni,
			TimeStampOrNull(comune.DataRitiroSm),
			TimeStampOrNull(comune.DataPrimoPresubentro),
			comune.Population,
			comune.PopulationAIRE,
			comune.CodiceIstat,
		)
		if err != nil {
			panic(err)
		}

	}

	defer stmt.Close()
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated %d comuni from DataCheckList.xml", len(comuni))
}

func UpdateAnomalieSchedeSoggettoDate(db *sql.DB, anomalie []Anomalie) {

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}

	// first truncate the anomalie table...
	sttm, err := tx.Prepare("DELETE FROM ANOMALIE")
	_, err = sttm.Exec()
	if err != nil {
		panic(err)
	}

	stmt, err := tx.Prepare("INSERT INTO ANOMALIE (CODISTAT,DENOMINAZIONECOMUNE,DESCRIZIONANOMALIA,NUMNEROANOMALIE,TIPOANOMALIA,CLASSEANOMALIA)  VALUES (?,?,?,?,?,?);")
	if err != nil {
		panic(err)
	}

	for i := 0; i < len(anomalie); i++ {
		var anomalia = anomalie[i]

		_, err = stmt.Exec(
			anomalia.CodiceIstat,
			anomalia.Name,
			anomalia.Description,
			anomalia.Number,
			anomalia.TipoAnomalia,
			anomalia.ClassAnomalia)
		if err != nil {
			panic(err)
		}

	}

	defer stmt.Close()
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Updated %d anomalie from AnomalieSchedeSoggettoPreSubV.2.xml", len(anomalie))
}

// CheckAlertTable utility function to check for a specific alert type the last_date sent
// a bit like a NOSQL case squeezed int sqlite... :-/
func CheckAlertTable(db *sql.DB, alert string) time.Time {

	var sqlbuffer bytes.Buffer
	sqlbuffer.WriteString("SELECT LAST_PROCESSED_DATE FROM ALERTS WHERE ALERT_NAME = '" + alert + "'")

	rows, err := db.Query(sqlbuffer.String())
	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s %s", err, sqlbuffer.String()))
	}
	defer rows.Close()

	rows.Next()
	var lastTime time.Time

	if err := rows.Scan(&lastTime); err != nil {
		panic(err)
	}

	if err := rows.Err(); err != nil {
		panic(err)
	}

	return lastTime

}

// UpdateAlertsTable utility function to update the last_date of an executed process
// a bit like a NOSQL case squeezed int sqlite... :-/
func UpdateAlertsTable(db *sql.DB, alert string) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	sqlString := "UPDATE ALERTS SET LAST_PROCESSED_DATE = ? WHERE ALERT_NAME = ?"

	stmt, err := tx.Prepare(sqlString)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(time.Time.Unix(time.Now()), alert)

	if err != nil {
		log.Print(err)
	}
	defer stmt.Close()
	tx.Commit()
}

/**
Update the association between a comune and a fornitore
*/
func UpdateComuneFornitore(db *sql.DB, codiceIstat string, fornitore int) {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	sqlString := "UPDATE COMUNE SET ID_FORNITORE = ? WHERE CODICE_ISTAT = ?"

	stmt, err := tx.Prepare(sqlString)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec(fornitore, codiceIstat)

	if err != nil {
		log.Print(err)
	}
	defer stmt.Close()
	tx.Commit()
}
func GetSubentroInfo(db *sql.DB, aDate null.Time) SubentroInfo {
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	sqlString := "SELECT COUNT(ID), sum(POPOLAZIONE) from COMUNE INNER JOIN SUBENTRO on COMUNE.ID= SUBENTRO.ID_COMUNE AND SUBENTRO.FINAL_DATE=?"
	rows, err := db.Query(sqlString, aDate.Time.Unix())
	if err != nil {
		panic(fmt.Sprintf("STDDEV query error: %s %s", err, sqlString))
	}
	defer rows.Close()
	rows.Next()
	var subentroInfo = SubentroInfo{
		DateFinal: aDate,
		//Comuni:0
		//Population:0
	}
	//subentroInfo.DateFinal = aDate
	err = rows.Scan(&subentroInfo.Comuni, &subentroInfo.Population)
	if err != nil {
		log.Print(err, sqlString)
	}
	tx.Commit()
	return subentroInfo
}
func Close(db *sql.DB) {
	defer db.Close()
}
