package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path"
	"time"

	"github.com/ccontavalli/goutils/email"
	"github.com/ccontavalli/goutils/gin/gtemplates"
	"github.com/ccontavalli/goutils/templates"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/acme/autocert"
	"gopkg.in/yaml.v2"
)

var gConfigfile = flag.String("config-file", "", "Config file. Either specify a config file, or listen-on with root.")

var gListenonHttp = flag.String("http-listen-on", "127.0.0.1:3000", "Address and port # to listen on for http requests.")
var gListenonHttps = flag.String("https-listen-on", "", "Address and port # to listen on for https requests.")

var gRoot = flag.String("root", "", "Directory to scan for documents to export. Additional path to export.")
var gWtemplates = flag.String("web-templates", "templates", "Directory to scan for templates.")
var gEtemplates = flag.String("email-templates", "emails", "Directory to scan for templates.")
var gStaticContent = flag.String("static-content", "static", "Static content directory.")
var gMode = flag.String("mode", "release", "Use to set the gin mode - release, or debug")

var gOauthCreds = flag.String("oauth-creds", "creds/oauth-creds.yaml", "JSON file containing OAUTH credentials - obtained from the identity provider.")
var gEmailCreds = flag.String("email-creds", "creds/email-creds.yaml", "JSON file containing secrets used to send out emails")
var gCookiekeys = flag.String("cookie-creds", "creds/cookie-creds.json", "JSON file containing keys to encrypt and verify cookies. If none is specified, one is created.")

// to drive the alerting system
const (
	PROACTIVE string = "ProactiveEmailAlert"
	REACTIVE  string = "ReactiveEmailAlert"
	TESTENV   string = "sandbox"
)

type MyData struct {
	Title string
}

type SecureKeys struct {
	HashKey       []byte
	EncryptionKey []byte
}

// ReadOrCreateKey Reads a key from a file, or creates a new one and stores it in a file.
// Returns error if it can't succeed in generating or storing a new key.
func ReadOrCreateKey(path string) (*SecureKeys, error) {
	file, err := ioutil.ReadFile(path)
	if err == nil {
		var keys SecureKeys
		err = json.Unmarshal(file, &keys)
		if err == nil {
			return &keys, nil
		}
	}
	log.Printf("GENERATING NEW KEYS - %s - %v\n", path, err)

	keys := SecureKeys{securecookie.GenerateRandomKey(32), securecookie.GenerateRandomKey(32)}
	blob, err := json.Marshal(keys)
	if err != nil {
		return nil, err
	}

	err = ioutil.WriteFile(path, blob, 0600)
	return &keys, err
}

func redirect(w http.ResponseWriter, req *http.Request) {
	// The host string _may_ in	clude the port number.
	// So it could be test.machine:3000, for example, or [1::12]:3000, for IPv6.
	host, _, err := net.SplitHostPort(req.Host)
	if err != nil {
		host = req.Host
	}

	// TODO: the https server may be on a non standard port. This port may also
	// not have anything to do with the port the server is listening on (in the case
	// of NAT devices or load balancers in front of the server - for example).
	// This function should take a parameter to determine which https to redirect to.
	to := "https://" + host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		to += "?" + req.URL.RawQuery
	}
	http.Redirect(w, req, to, http.StatusMovedPermanently)
}

func showTemplate(name string, c *gin.Context) {
	tmpl, err := template.ParseFiles(path.Join(*gWtemplates, name))
	if err != nil {
		panic("could not parse template")
	}
	buffer := bytes.Buffer{}
	bufwriter := bufio.NewWriter(&buffer)
	err = tmpl.Execute(bufwriter, MyData{"My Website"})
	if err != nil {
		panic("could not execute template")
	}
	bufwriter.Flush()
	c.Data(http.StatusOK, "text/html", buffer.Bytes())
}

func main() {

	// In order to chech on the production server when the server daemon restart!
	log.Printf("DASHBOARD RESTARTED @ %v-%v-%v %v:%v", time.Now().Year(), time.Now().Month(), time.Now().Day(), time.Now().Hour(), time.Now().Minute())

	flag.Parse()

	keys, err := ReadOrCreateKey(*gCookiekeys)
	if err != nil {
		log.Fatal("Could not create keys - ", err)
	}

	googleOauth, err := NewOauthAuthenticator(*gOauthCreds, "/auth/google/callback", keys)
	if err != nil {
		log.Fatal("Could not initialize authenticator - ", err)
	}

	var config *ServerConfig
	if *gConfigfile != "" {
		config, err = NewServerConfigFromFile(*gConfigfile)
		if err != nil {
			log.Fatal("Could not read configuration file - ", err)
		}
		log.Printf("LOADED CONFIG FILE %s", *gConfigfile)
	}

	urloptions := DefaultUrlManagerOptions()
	if config != nil && config.Options != nil {
		urloptions = *config.Options.Merge(urloptions)
	} else {
		if *gRoot == "" {
			log.Fatal("You must specify --root flag - when no config file or no Options are provided")
		}
	}
	if *gRoot != "" {
		// The "nil" here means that no specific config was supplied, just use the default config.
		urloptions.Paths[*gRoot] = nil
	}

	storageOptions := DefaultStorageOptions()
	if config != nil && config.StorageOptions != nil {
		storageOptions = config.StorageOptions.Merge(storageOptions)
	}

	authorizationManager := NewAuthorizationManager(config.Groups)

	htmlRender, err := templates.NewStaticTemplatesFromDir(nil, *gWtemplates, nil)
	if err != nil {
		log.Fatal("Could not initialize web templates - ", err)
	}

	emailRender, err := templates.NewStaticTemplatesFromDir(nil, *gEtemplates, nil)
	if err != nil {
		log.Fatal("Could not initialize email templates - ", err)
	}

	emailSender, err := email.NewMailSenderFromConfigFile(*gEmailCreds, yaml.Unmarshal, emailRender)
	if err != nil {
		log.Fatal("Could not initialize email sender - ", err)
	}

	emailAuthenticator, err := NewEmailAuthenticator("/auth/email/send-validation", "/auth/email/validate", "validation", emailSender, authorizationManager, nil)
	if err != nil {
		log.Fatal("Could not initialize email validator - ", err)
	}

	urlmanager, err := NewUrlManager(urloptions, UrlManagerAuthenticators{googleauth: googleOauth, emailauth: emailAuthenticator}, authorizationManager, htmlRender)
	if err != nil {
		log.Fatal("Could not initialize urlmanager - ", err)
	}

	storageManager, err := NewStorageManager(storageOptions, authorizationManager, emailSender, "updatecomuni")
	if err != nil {
		log.Fatal("Could not initialize storage manager - ", err)
	}
	defer storageManager.Close()

	_, err = NewAlertManager(storageOptions, config, emailSender, map[string]string{REACTIVE: "alert_outofdate", PROACTIVE: "alert_closetodate"})
	if err != nil {
		log.Fatal("Could not initialize alert manager - ", err)
	}

	// Routing section

	gin.SetMode(*gMode)

	router := gin.Default()

	//router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.HTMLRender = (*gtemplates.HTMLRender)(htmlRender)

	store := cookie.NewStore(keys.HashKey, keys.EncryptionKey)
	router.Use(sessions.Sessions("goospel", store))
	router.Use(gin.Recovery())

	// Static URLS take precedence over any automatically computed URL.
	// Note that static URLs cannot be updated at run time. Need restart.
	router.StaticFile("/favicon.ico", path.Join(*gStaticContent, "favicon.ico"))
	router.Static("/static", *gStaticContent)

	//router.GET("/auth/google/callback", googleOauth.CallbackHandler)
	router.POST("/auth/email/send-validation", emailAuthenticator.FormSubmitHandler)
	router.GET("/auth/email/validate", emailAuthenticator.ValidateHandler)
	router.GET("/auth/email/logout", emailAuthenticator.Logout)

	router.POST("/comune/search", urlmanager.OnlyLoggedHandler(storageManager.Search))
	router.POST("/comune/status", urlmanager.OnlyLoggedHandler(storageManager.Status))
	router.PUT("/comune/update", urlmanager.OnlyLoggedHandler(storageManager.Update))

	router.GET("/downloadpianosubentro", urlmanager.OnlyLoggedHandler(storageManager.GetPianoSubentro))
	router.PUT("/uploadpianosubentro", urlmanager.OnlyLoggedHandler(storageManager.PutPianoSubentro))

	router.POST("/comune/searchComments", urlmanager.OnlyLoggedHandler(storageManager.SearchComments))
	router.PUT("/comune/updateComment", urlmanager.OnlyLoggedHandler(storageManager.SaveOrUpdateComment))
	router.POST("/comune/deleteComment", urlmanager.OnlyLoggedHandler(storageManager.DeleteComment))

	if *gMode == "debug" {
		router.GET("/debug", urlmanager.Debug)
		router.GET("/panic", func(c *gin.Context) { panic("silvestro vive") })
	}

	// Show the main Subentro page
	router.GET("/subentro", urlmanager.OnlyLoggedHandler(func(c *gin.Context) {
		showTemplate("subentro.tmpl", c)
	}))

	router.GET("/", urlmanager.OnlyLoggedHandler(func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/subentro")
	}))

	// Show a simple page with the progresses of "subentri", ordered by Comune
	router.GET("/progressi", urlmanager.OnlyLoggedHandler(func(c *gin.Context) {
		showTemplate("progressi.tmpl", c)
	}))

	// Show a status page for a single Comune
	router.GET("/status", urlmanager.OnlyLoggedHandler(func(c *gin.Context) {
		showTemplate("status.tmpl", c)
	}))
	router.GET("/api/comune/:codice_istat", func(c *gin.Context) {
		codiceIstat := c.Param("codice_istat")
		storageManager.SearchComuniByCodiceIstat(c, codiceIstat)
	})

	router.POST("/api/comune/updateFornitore", urlmanager.OnlyLoggedHandler(storageManager.UpdateFornitore))
	router.GET("/api/getSubentroInfo/*date", urlmanager.OnlyLoggedHandler(storageManager.GetSubentroInfo))
	// Install a not found handler. The not found handler uses an internal
	// router created and updated dynamically from the list of documents to
	// render.
	router.NoRoute(urlmanager.Process)
	//router.Use(gzip.Gzip(gzip.DefaultCompression))

	// If both http and https, http always redirects to https.
	if *gListenonHttps != "" {
		if config == nil || len(config.Hostnames) <= 0 {
			log.Fatal("When using https, you must specify a config file with a list of hostnames")
		}

		go http.ListenAndServe(*gListenonHttp, http.HandlerFunc(redirect))

		m := autocert.Manager{
			Cache:      autocert.DirCache(urloptions.SSLCacheDir),
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(config.Hostnames...),
		}
		s := &http.Server{
			Addr:      *gListenonHttps,
			Handler:   router,
			TLSConfig: &tls.Config{GetCertificate: m.GetCertificate},
		}
		s.ListenAndServeTLS("", "")

		router.Run(*gListenonHttps)
	} else {
		router.Run(*gListenonHttp)
	}
}
