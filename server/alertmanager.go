package main

import (
	"database/sql"
	"log"
	"strings"
	"time"

	"github.com/ccontavalli/goutils/email"
	"github.com/teamdigitale/anpr-dashboard-server/sqlite"
)

type AlertManager struct {
	sender                     *email.MailSender
	template                   map[string]string
	config                     *ServerConfig
	db                         *sql.DB
	fornitori_acls             map[string]ACL
	notify_email               []string
	periodic_routines_interval time.Duration
}

func NewAlertManager(options *StorageOptions, config *ServerConfig, sender *email.MailSender, template map[string]string) (*AlertManager, error) {

	//log.Printf("Alert Options:\n%v", *sender)

	manager := AlertManager{
		sender:                     sender,
		template:                   template,
		config:                     config,
		db:                         sqlite.OpenDB(options.DatabasePath),
		fornitori_acls:             options.FornitoriACLs,
		notify_email:               options.NotifyEmail,
		periodic_routines_interval: 24 * 60 * 60 * time.Second,
	}

	go manager.RunPeriodicRoutines()

	return &manager, nil

}

// SendindEmailAlert : this function will run periodically and send to the relative FORNITORE
// an email alert about a SUBENTRO close (10 days before) OR over to its planned deadline
// ANPR-80
func (manager *AlertManager) SendingEmailAlert(allAlerts []string) {

	for _, t := range allAlerts {

		lastRun := sqlite.CheckAlertTable(manager.db, t).Unix()

		todayTS := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()).Unix()

		log.Printf(t+" timesheet: on db %v, today %v", lastRun, todayTS)

		if lastRun < todayTS {

			alerting := sqlite.SearchAlerts(manager.db, t)

			for _, v := range alerting {
				log.Printf("MAIL ALERT | %v | %s | %s | %s | %s | %s", todayTS, t, v.Name, v.FornitoreName, v.DateTo, v.FornitoreEmail)
				revDateTo := strings.Split(v.DateTo, "-")
				DateToHuman := revDateTo[2] + "-" + revDateTo[1] + "-" + revDateTo[0]

				if (manager.config.StorageOptions.Environment == TESTENV) || (v.FornitoreEmail == "") {
					v.FornitoreEmail = strings.Join(manager.config.StorageOptions.NotifyEmail, ",")
				}

				err := manager.sender.Send(manager.template[t],
					struct{ NomeComune, DateTo, FornitoreName string }{v.Name, DateToHuman, v.FornitoreName},
					v.FornitoreEmail)
				if err != nil {
					log.Printf("Could not send email for "+t+" notice: %s", err)
				}

				if manager.config.StorageOptions.Environment == TESTENV {
					break
				}
			}

			sqlite.UpdateAlertsTable(manager.db, t)
		}
	}
}

// RunPeriodicRoutines : this is the core of the chronos loop
func (manager *AlertManager) RunPeriodicRoutines() {

	var allAlerts []string

	for k := range manager.template {
		allAlerts = append(allAlerts, k)
	}

	for {
		manager.SendingEmailAlert(allAlerts)
		time.Sleep(manager.periodic_routines_interval)
	}
}
