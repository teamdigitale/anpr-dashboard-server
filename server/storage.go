package main

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html/charset"

	"github.com/ccontavalli/goutils/email"
	"github.com/gin-gonic/gin"
	"github.com/teamdigitale/anpr-dashboard-server/sqlite"
	"gopkg.in/guregu/null.v3"
)

var (
	columnTitles = []string{
		"id_comune",
		"codice_istat",
		"nome_comune",
		"prima_data_subentro",
		"ultima_data_subentro",
		"data_subentro_preferita",
	}
	columnTitlesForNotify = []string{
		"id_comune",
		"codice_istat",
		"nome_comune",
		"fornitore",
		"prima_data_subentro",
		"ultima_data_subentro",
		"data_subentro_preferita",
	}
	detailedColumnTitles = []string{
		"id_comune",
		"codice_istat",
		"nome_comune",
		"provincia",
		"regione",
		"popolazione",
		"popolazione_aire",
		"postazioni",
		"lat",
		"lon",
		"fornitore",
		"prima_data_subentro",
		"ultima_data_subentro",
		"data_subentro_preferita",
		"data_consegna_sc",
		"data_ritiro_sc",
		"data_subentro",
		"data_primo_presubentro",
		"data_ultimo_presubentro",
	}
)

type ACL struct {
	Readers []GroupID
	Writers []GroupID
}

func (acl ACL) GetReaders() []GroupID {
	return acl.Readers
}

func (acl ACL) GetWriters() []GroupID {
	return acl.Writers
}

type StorageOptions struct {
	DatabasePath               string
	FornitoriACLs              map[string]ACL
	NotifyEmail                []string
	Environment                string
	CheckDataListPath          string
	AnomalieSchedeSoggettoPath string
}

func (options *StorageOptions) Merge(source *StorageOptions) *StorageOptions {
	merged := StorageOptions{}
	if options.DatabasePath != "" {
		merged.DatabasePath = options.DatabasePath
	} else {
		merged.DatabasePath = source.DatabasePath
	}

	merged.FornitoriACLs = options.FornitoriACLs

	if len(options.NotifyEmail) != 0 {
		merged.NotifyEmail = options.NotifyEmail
	} else {
		merged.NotifyEmail = source.NotifyEmail
	}

	if options.CheckDataListPath != "" {
		merged.CheckDataListPath = options.CheckDataListPath
	} else {
		merged.CheckDataListPath = source.CheckDataListPath
	}

	if options.AnomalieSchedeSoggettoPath != "" {
		merged.AnomalieSchedeSoggettoPath = options.AnomalieSchedeSoggettoPath
	} else {
		merged.AnomalieSchedeSoggettoPath = source.AnomalieSchedeSoggettoPath
	}

	return &merged
}

//Ignore this function if already defined in the config file
func DefaultStorageOptions() *StorageOptions {
	options := StorageOptions{
		DatabasePath:               "sqlite.db",
		NotifyEmail:                []string{"team-anpr-changes@teamdigitale.governo.it", "specifiche.anpr@sogei.it"},
		CheckDataListPath:          "DatiCheckListV2.xml",
		AnomalieSchedeSoggettoPath: "AnomalieSchedeSoggettoPreSubV.2.xml",
	}
	return &options
}

type StorageManager struct {
	db             *sql.DB
	authz_mgr      *AuthorizationManager
	fornitori_acls map[string]ACL

	email_sender          *email.MailSender
	notify_email          []string
	notify_email_template string

	periodic_routines_interval  time.Duration
	checkDatalist_path          string
	AnomalieSchedeSoggetto_path string
}

func NewStorageManager(options *StorageOptions, authz_mgr *AuthorizationManager, email_sender *email.MailSender, notify_email_template string) (*StorageManager, error) {
	//log.Printf("Storage Options:\n%v", *options)

	manager := StorageManager{
		db:             sqlite.OpenDB(options.DatabasePath),
		authz_mgr:      authz_mgr,
		fornitori_acls: options.FornitoriACLs,

		email_sender:          email_sender,
		notify_email:          options.NotifyEmail,
		notify_email_template: notify_email_template,

		periodic_routines_interval:  60 * 60 * time.Second,
		checkDatalist_path:          options.CheckDataListPath,
		AnomalieSchedeSoggetto_path: options.AnomalieSchedeSoggettoPath,
	}

	go manager.RunPeriodicRoutines()

	return &manager, nil
}

func (manager *StorageManager) Close() {
	sqlite.Close(manager.db)
}

func (manager *StorageManager) getACLComune(comune sqlite.Comune) ACL {
	acl, is_set := manager.fornitori_acls[comune.Fornitore.Name]
	if !is_set {
		return ACL{}
	}
	return acl
}

func (manager *StorageManager) notifyUpdateComuni(credentials *Credentials, comuni []sqlite.Comune) {

	buffer := &bytes.Buffer{}
	csv_writer := csv.NewWriter(buffer)
	csv_writer.Write(columnTitlesForNotify)
	for _, comune := range comuni {
		csv_writer.Write(comune.AsCSVRecordForNotify())
	}
	csv_writer.Flush()
	log.Printf("Received update from %s :\n%s", credentials.AsString(), buffer.String())

	err := manager.email_sender.Send(manager.notify_email_template, struct{ Credentials, Change string }{credentials.AsString(), buffer.String()}, manager.notify_email...)
	if err != nil {
		log.Printf("Could not send email for update notice: %s", err)
	}
}

func (manager *StorageManager) notifyUpdateComment(credentials *Credentials, comment sqlite.Comment, updateType string) {

	change := fmt.Sprintf("%s\n%s", updateType, comment.AsString())
	log.Printf("Received update from %s :\n%s", credentials.AsString(), change)

	err := manager.email_sender.Send(manager.notify_email_template, struct{ Credentials, Change string }{credentials.AsString(), change}, manager.notify_email...)
	if err != nil {
		log.Printf("Could not send email for update notice: %s", err)
	}
}

type StorageResult struct {
	Result   string            `json:"result"`
	Error    string            `json:"error"`
	Data     []sqlite.Comune   `json:"data"`
	Anomalia []sqlite.Anomalie `json:"anomalia"`
	Comments []sqlite.Comment  `json:"comment"`
}

type StorageFEResult struct {
	Result string              `json:"result"`
	Error  string              `json:"error"`
	Data   []sqlite.ComuneInfo `json:"data"`
}

func (manager *StorageManager) SearchComuniByCodiceIstat(ctx *gin.Context, codice_istat string) {
	res := StorageFEResult{
		//
	}
	s :=
		sqlite.SearchFilter{
			Comune: sqlite.Comune{
				CodiceIstat: codice_istat,
			},
		}
	comuni := sqlite.SearchComuni(manager.db, s)
	if len(comuni) != 0 {

		//comune.Fornitore = Fornitore{}
		res.Data = append(res.Data, (comuni[0]).ToComuneInfo())
	}
	res.Result = "ok"
	ctx.JSON(200, res)
}

func (manager *StorageManager) UpdateFornitore(ctx *gin.Context) {
	credentials := GetCredentials(ctx)
	res := StorageResult{
		Result: "nok",
	}

	if !manager.authz_mgr.IsAdmin(credentials) {
		ctx.JSON(403, res)
		return
	}
	comuneFornitore := sqlite.ComuneFornitore{}
	err := ctx.BindJSON(&comuneFornitore)
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}
	sqlite.UpdateComuneFornitore(manager.db, comuneFornitore.CodiceIstat, comuneFornitore.FornitoreID)
	res.Result = "ok"
	ctx.JSON(200, res)

}
func (manager *StorageManager) Search(ctx *gin.Context) {
	credentials := GetCredentials(ctx)
	res := StorageResult{
		Result: "nok",
	}

	filter := sqlite.SearchFilter{}
	//fmt.Printf("%s", ctx.Request.Body) << EVIL ON PRODUCTION
	err := ctx.BindJSON(&filter)
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}

	comuni := sqlite.SearchComuni(manager.db, filter)

	for _, comune := range comuni {
		acl := manager.getACLComune(comune)
		if manager.authz_mgr.HasReadAccess(acl, credentials) || manager.authz_mgr.IsAdmin(credentials) {
			res.Data = append(res.Data, comune)
		} else {

		}
	}

	res.Result = "ok"
	ctx.JSON(200, res)
}

func (manager *StorageManager) Status(ctx *gin.Context) {

	res := StorageResult{
		Result: "nok",
	}

	filter := sqlite.AnomalieSearchFilter{}
	err := ctx.BindJSON(&filter)
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}

	anomalie := sqlite.SearchAnomalie(manager.db, filter)

	for _, anomalia := range anomalie {
		res.Anomalia = append(res.Anomalia, anomalia)
	}

	log.Print(res.Anomalia)

	res.Result = "ok"
	ctx.JSON(200, res)
}

func (manager *StorageManager) SearchComments(ctx *gin.Context) {
	res := StorageResult{
		Result: "nok",
	}
	comune := sqlite.Comune{}
	//fmt.Printf("%s", ctx.Request.Body)
	err := ctx.BindJSON(&comune)
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}
	acl := manager.getACLComune(comune)
	if manager.authz_mgr.HasReadAccess(acl, GetCredentials(ctx)) {
		var comments = sqlite.SearchComment(manager.db, comune)

		res.Data = append(res.Data, comune)
		res.Comments = comments
	}

	res.Result = "ok"
	ctx.JSON(200, res)
}

func (manager *StorageManager) GetSubentroInfo(ctx *gin.Context) {
	dateString := ctx.Param("date")[1:]
	date := NullTimeFromString(dateString)
	info := sqlite.GetSubentroInfo(manager.db, date)
	log.Printf("Info retrieved %v for date %v, param %s", info, date, dateString)
	ctx.JSON(200, info)
}
func (manager *StorageManager) SaveOrUpdateComment(ctx *gin.Context) {
	res := StorageResult{
		Result: "nok",
	}
	comment := sqlite.Comment{}
	err := ctx.BindJSON(&comment)
	if err != nil {
		res.Error = err.Error()
		log.Print(err)
		ctx.JSON(400, res)
		return
	}
	comune := comment.Comune
	acl := manager.getACLComune(comune)
	credentials := GetCredentials(ctx)
	log.Print("comune", comune)
	if manager.authz_mgr.HasWriteAccess(acl, credentials) {
		log.Print("Sanitize and save comment ", comment)
		comment.Date = null.NewTime(time.Now(), true)
		p := bluemonday.UGCPolicy()
		comment.Author = fmt.Sprintf("%s", *credentials.User)
		comment.Content = p.Sanitize(comment.Content)
		sqlite.SaveOrUpdateComment(manager.db, comment)
		manager.notifyUpdateComment(credentials, comment, "added")
	}
	res.Result = "ok"
	ctx.JSON(200, res)

}
func (manager *StorageManager) DeleteComment(ctx *gin.Context) {
	res := StorageResult{
		Result: "nok",
	}
	comment := sqlite.Comment{}
	err := ctx.BindJSON(&comment)
	if err != nil {
		res.Error = err.Error()
		log.Print(err)
		ctx.JSON(400, res)
		return
	}
	comune := comment.Comune
	acl := manager.getACLComune(comune)
	credentials := GetCredentials(ctx)
	log.Print("comune", comune)
	if manager.authz_mgr.HasWriteAccess(acl, credentials) {
		sqlite.DeleteComment(manager.db, comment)
		manager.notifyUpdateComment(credentials, comment, "deleted")
	}
	res.Result = "ok"
	ctx.JSON(200, res)
}
func (manager *StorageManager) Update(ctx *gin.Context) {

	res := StorageResult{
		Result: "nok",
	}

	given_comune := sqlite.Comune{}
	err := ctx.BindJSON(&given_comune)
	if err != nil {
		res.Error = err.Error()
		log.Print(err)
		ctx.JSON(400, res)
		return
	}

	filter := sqlite.SearchFilter{Comune: given_comune}
	filter.Comune.Subentro.From.Valid = false
	filter.Comune.Subentro.To.Valid = false

	search_results := sqlite.SearchComuni(manager.db, filter)
	if len(search_results) != 1 {
		res.Error = "Comune not found"
		log.Print(err)
		ctx.JSON(404, res)
		return
	}

	comune := search_results[0]

	credentials := GetCredentials(ctx)
	acl := manager.getACLComune(comune)

	//Check authorization on the single item (prevent request forgery)
	if !manager.authz_mgr.IsAdmin(credentials) && !manager.authz_mgr.HasWriteAccess(acl, credentials) {
		res.Error = "permission denied"
		ctx.JSON(403, res)
		return
	}

	_, err = given_comune.Subentro.IsWellFormed(&comune)
	if err != nil {
		res.Error = fmt.Sprintf("ERRORE: %s", err.Error())
		log.Print(err)
		ctx.JSON(400, res)
		return
	}
	if given_comune.Subentro.PreferredDate.Valid {

		info := sqlite.GetSubentroInfo(manager.db, comune.Subentro.PreferredDate)
		log.Printf("Check info of subentro for date %v, numero comuni:%v, popolazione:%v", given_comune.Subentro.PreferredDate, info.Comuni, info.Population)

		var limit int64 = 50
		var comuniInfoValue int64

		if info.Comuni.Valid {
			comuniInfoValue = info.Comuni.Int64
		}

		if comuniInfoValue > limit {
			res.Error = fmt.Sprintf("ERRORE: limite raggiunto di %d comuni nella data indicata", limit)
			log.Print(err)
			ctx.JSON(400, res)
			return
		}
	}

	comune.Subentro = given_comune.Subentro

	sqlite.SaveOrUpdateSubentro(manager.db, comune)
	manager.notifyUpdateComuni(credentials, []sqlite.Comune{comune})

	res.Result = "ok"
	ctx.JSON(200, res)
}
func (manager *StorageManager) GetPianoSubentro(ctx *gin.Context) {

	detailed_records, _ := strconv.ParseBool(ctx.Query("detailed_records"))

	buffer := &bytes.Buffer{}
	csv_writer := csv.NewWriter(buffer)

	if detailed_records {
		csv_writer.Write(detailedColumnTitles)
	} else {
		csv_writer.Write(columnTitles)
	}

	filter := sqlite.SearchFilter{}
	comuni := sqlite.SearchComuni(manager.db, filter)
	credentials := GetCredentials(ctx)

	for _, comune := range comuni {
		acl := manager.getACLComune(comune)
		if manager.authz_mgr.HasReadAccess(acl, credentials) || manager.authz_mgr.IsAdmin(credentials) || manager.authz_mgr.IsAdminReader(credentials) {
			if detailed_records {
				//log.Print("Writing %s", comune.AsDetailedCSVRecord())
				csv_writer.Write(comune.AsDetailedCSVRecord())
			} else {
				csv_writer.Write(comune.AsCSVRecord())
			}
		}
	}

	csv_writer.Flush()

	ctx.Header("Content-Description", "File Transfer")
	ctx.Header("Content-Disposition", "attachment; filename=pianosubentro.csv")
	ctx.Data(200, "text/csv", buffer.Bytes())
}
func (manager *StorageManager) PutPianoSubentro(ctx *gin.Context) {

	res := StorageResult{
		Result: "nok",
	}

	file_header, err := ctx.FormFile("csv")
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}
	csv_file, err := file_header.Open()
	if err != nil {
		res.Error = err.Error()
		ctx.JSON(400, res)
		return
	}

	credentials := GetCredentials(ctx)

	csv_reader := csv.NewReader(csv_file)
	comuni := []sqlite.Comune{}
	error_msgs := []string{}
	num_row := 0
	for {
		row, err := csv_reader.Read()
		num_row++
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err.Error())
		}
		if reflect.DeepEqual(row, columnTitles) {
			continue
		}

		given_comune, err := sqlite.NewComuneFromCSVRecord(row)
		if err != nil {
			error_msgs = append(error_msgs, fmt.Sprintf("ERRORE(riga %d): %s", num_row, err.Error()))
			continue
		}

		if !given_comune.Subentro.HasValidDates() {
			// Skip CSV rows that are empty.
			continue
		}
		//given_comune.Subentro = sqlite.Subentro{}

		filter := sqlite.SearchFilter{
			Comune:    *given_comune,
			Fornitore: sqlite.Fornitore{},
			//Exclusion: nil,
		}

		search_results := sqlite.SearchComuni(manager.db, filter)
		if len(search_results) != 1 {
			error_msgs = append(error_msgs, fmt.Sprintf("ERRORE(riga %d): comune not found", num_row))
			continue
		}

		comune := search_results[0]

		acl := manager.getACLComune(comune)
		if !manager.authz_mgr.HasWriteAccess(acl, credentials) {
			error_msgs = append(error_msgs, fmt.Sprintf("ERRORE(riga %d): non autorizzato a modificare comune", num_row))
			continue
		}

		_, err = given_comune.Subentro.IsWellFormed(&comune)
		if err != nil {
			error_msgs = append(error_msgs, fmt.Sprintf("ERRORE(riga %d): %s", num_row, err.Error()))
			continue
		}

		comune.Subentro = given_comune.Subentro
		if comune.Subentro.From.Valid {
			comuni = append(comuni, comune)
		}
	}

	if len(error_msgs) == 0 {
		log.Printf("Comuni da modificare: %d", len(comuni))
		for _, comune := range comuni {
			log.Printf("Save update for comune %s ", comune.Name)
			sqlite.SaveOrUpdateSubentro(manager.db, comune)
		}
		res.Result = "ok"
		res.Error = fmt.Sprintf("'%s' uploaded", file_header.Filename)
		ctx.JSON(200, res)
		return
	}
	manager.notifyUpdateComuni(credentials, comuni)

	res.Error = strings.Join(error_msgs, "\n")
	ctx.JSON(400, res)
}

type ComuneState struct {
	CodiceIstat             string `xml:"CODICEISTAT"`
	SiglaProvincia          string `xml:"SIGLAPROVINCIA"`
	SiglaPrefettura         string `xml:"SIGLAPREFETTURA"`
	Name                    string `xml:"DENOMINAZIONE"`
	DataSubentro            string `xml:"DATASUBENTRO"`
	NumeroPostazioni        string `xml:"NUMEROPOSTAZIONI"`
	DataPreSubentro         string `xml:"DATAPRESUBENTRO"`
	AbilitazionePrefettura  string `xml:"ABILITAZIONEPREFETTURA"`
	Popolazione             string `xml:"NUM_APR"`
	PopolazioneAIRE         string `xml:"NUM_AIRE"`
	PopolazioneIstat        string `xml:"ABITANTIISTAT"`
	DataConsegnaSC          string `xml:"DATACONSEGNASC"`
	NumeroLettoriConsegnati string `xml:"NUMEROLETTORICONSEGNATI"`
	DataAbilitazione        string `xml:"DATAABILITAZIONETEST"`
	UtentiAbilitati         string `xml:"UTENTIABILITATI"`
	IPProvenienza           string `xml:"IPPROVENIENZA"`
	EmailPec                string `xml:"EMAILPEC"`
	SCConsegnate            string `xml:"NUMEROSMARTCARDCONSEGNATE"`
	DataRitiroSmartCard     string `xml:"DATARITIROSCPREFETTURA"`
	DataPrimoPreSubentro    string `xml:"DATAPRIMOPRESUBENTRO"`
}

type AnomaliaState struct {
	CodiceIstat   string `xml:"CODISTAT"`
	Name          string `xml:"DENOMINAZIONECOMUNE"`
	Description   string `xml:"DESCRIZIONANOMALIA"`
	Number        int    `xml:"NUMNEROANOMALIE"`
	ClassAnomalia string `xml:"CLASSEANOMALIA"`
	TipoAnomalia  string `xml:"TIPOANOMALIA"`
}

type ComuneCheckList struct {
	Rows []ComuneState `xml:"ROW"`
}

type AnomaliaCheckList struct {
	Rows []AnomaliaState `xml:"ROW"`
}

func NullTimeFromString(date string) null.Time {
	aNullTime, err := time.Parse("02/01/2006", date)
	if err == nil {
		return null.TimeFrom(aNullTime)
	} else {

		return null.TimeFromPtr(nil)
	}
}

func NullIntFromString(anInt string) null.Int {

	aNullInt, err := strconv.ParseInt(anInt, 10, 64)
	if err == nil {
		return null.IntFrom(aNullInt)
	} else {
		return null.IntFromPtr(nil)
	}

}

func NullBoolFromString(aString string) null.Bool {

	if aString == "SI" {
		return null.BoolFrom(true)
	} else {
		return null.BoolFrom(false)
	}

}

/**
Conver a XML row into a comue for data required to update
*/
func (state *ComuneState) ToComune() sqlite.Comune {
	//log.Printf("Codice Istat %s, DataSubentro %s,  DataPreSubentro: %s,DataConsegnaSC; %s", state.CodiceIstat, state.DataSubentro, state.DataPreSubentro, state.DataConsegnaSC)
	var comune sqlite.Comune
	comune.DataSubentro = NullTimeFromString(state.DataSubentro)
	comune.DataConsegnaSm = NullTimeFromString(state.DataConsegnaSC)
	comune.DataPresubentro = NullTimeFromString(state.DataPreSubentro)
	comune.DataAbilitazione = NullTimeFromString(state.DataAbilitazione)
	comune.NumeroLettori = NullIntFromString(state.NumeroLettoriConsegnati)
	comune.Postazioni = NullIntFromString(state.NumeroPostazioni)
	comune.CodiceIstat = state.CodiceIstat
	comune.UtentiAbilitati = NullIntFromString(state.UtentiAbilitati)
	comune.AbilitazionePrefettura = NullBoolFromString(state.AbilitazionePrefettura)
	comune.IPProvenienza = NullBoolFromString(state.IPProvenienza)
	comune.EmailPec = NullBoolFromString(state.EmailPec)
	comune.SCConsegnate = NullIntFromString(state.SCConsegnate)
	comune.DataRitiroSm = NullTimeFromString(state.DataRitiroSmartCard)
	comune.DataPrimoPresubentro = NullTimeFromString(state.DataPrimoPreSubentro)

	comune.PopulationAIRE, _ = strconv.Atoi(state.PopolazioneAIRE)

	//In case the specific comune is not in ANPR, the fall back to the istat population
	population, _ := strconv.Atoi(state.Popolazione)
	if population == 0 {
		population, _ = strconv.Atoi(state.PopolazioneIstat)
	}
	comune.Population = population
	return comune
}

/**
Conver a XML row into a comue for data required to update
*/
func (state *AnomaliaState) ToAnomalia() sqlite.Anomalie {

	var anomalia sqlite.Anomalie
	anomalia.CodiceIstat = state.CodiceIstat
	anomalia.Description = state.Description
	anomalia.Name = state.Name
	anomalia.Number = state.Number
	anomalia.ClassAnomalia = state.ClassAnomalia
	anomalia.TipoAnomalia = state.TipoAnomalia

	return anomalia
}

func ParseXML(fd *os.File, data interface{}) error {
	decoder := xml.NewDecoder(fd)
	decoder.CharsetReader = func(s string, r io.Reader) (io.Reader, error) { return charset.NewReader(r, s) }

	return decoder.Decode(data)
}

func (manager *StorageManager) UpdateComuniCheckList() {
	fd, err := os.Open(manager.checkDatalist_path)
	if err != nil {
		log.Printf("WARNING: Not updating subentro state: %s", err.Error())
		return
	}

	state := ComuneCheckList{}
	err = ParseXML(fd, &state)
	if err != nil {
		log.Printf("WARNING: Not updating subentro state: %s", err.Error())
		return
	}

	var comuni = []sqlite.Comune{}
	for _, v := range state.Rows {

		comuni = append(comuni, v.ToComune())
	}

	sqlite.UpdateComuneCheckListDate(manager.db, comuni)

}

func (manager *StorageManager) UpdateAnomalieSchedeSoggetto() {

	// TODO: ORIGINALLY IT WAS ONCE A WEEK ON MONDAY... not anymore
	/*
		t := time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006")
		i := strings.Index(t, "Mon")
		if i == -1 {
			return
		}
	*/

	fd, err := os.Open(manager.AnomalieSchedeSoggetto_path)
	if err != nil {
		log.Printf("WARNING: Not updating anomalie state: %s", err.Error())
		return
	}

	state := AnomaliaCheckList{}
	err = ParseXML(fd, &state)
	if err != nil {
		log.Printf("WARNING: Not updating anomalie state: %s", err.Error())
		return
	}

	var anomalie = []sqlite.Anomalie{}
	for _, v := range state.Rows {

		anomalie = append(anomalie, v.ToAnomalia())
	}

	sqlite.UpdateAnomalieSchedeSoggettoDate(manager.db, anomalie)

}
func (manager *StorageManager) RunPeriodicRoutines() {
	for {
		manager.UpdateComuniCheckList()
		manager.UpdateAnomalieSchedeSoggetto()
		time.Sleep(manager.periodic_routines_interval)
	}
}
