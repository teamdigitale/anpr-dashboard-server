package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/ccontavalli/goutils/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"io/ioutil"
	"net/http"
	//"os"
)

func randToken() []byte {
	b := make([]byte, 32)
	rand.Read(b)
	return b
}

// OauthCredentials which stores google ids.
type OauthCredentials struct {
	Cid     string `json:"cid"`
	Csecret string `json:"csecret"`
}

type OauthAuthenticator struct {
	conftemplate *oauth2.Config
	cookiegen    *securecookie.SecureCookie
	callbackurl  string
}

// Returns a copy of the OauthConfig tweaked to support the current request.
// This is necessary to prevent multiple threads from modfying the same object.
func (oa *OauthAuthenticator) GetOauthConfigForContext(c *gin.Context) *oauth2.Config {
	newconf := *oa.conftemplate

	path := oa.callbackurl
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	newconf.RedirectURL = fmt.Sprintf("%s://%s%s", scheme, host, path)
	return &newconf
}

func loadOauthCredentials(filename string) (error, *OauthCredentials) {
	var cred OauthCredentials
	return config.ReadYamlConfigFromFile(filename, &cred), &cred
}

func NewOauthAuthenticator(filename string, callbackurl string, keys *SecureKeys) (*OauthAuthenticator, error) {
	err, creds := loadOauthCredentials(filename)
	if err != nil {
		return nil, err
	}
	return &OauthAuthenticator{
		conftemplate: &oauth2.Config{
			ClientID:     creds.Cid,
			ClientSecret: creds.Csecret,

			// This must be filled before use.
			// RedirectURL:  callbackurl,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
			},
			Endpoint: google.Endpoint,
		},
		cookiegen:   securecookie.New(keys.HashKey, keys.EncryptionKey),
		callbackurl: callbackurl,
	}, nil
}

type OauthDest struct {
	Token []byte
	Url   string
}

func (auth *OauthAuthenticator) GetUrl(c *gin.Context, destination string) (string, error) {
	// This is strictly not necessary, but verifies that the callback is invoked
	// from the same browser session that originated the redirect.
	session := sessions.Default(c)
	token, ok := session.Get("oauth-token").([]byte)
	if !ok || len(token) == 0 {
		token = randToken()
		session.Set("oauth-token", token)
		session.Save()
	}

	state, err := auth.cookiegen.Encode("oauth", OauthDest{token, destination})
	if err != nil {
		return "", err
	}
	return auth.GetOauthConfigForContext(c).AuthCodeURL(state), nil
}

func (auth *OauthAuthenticator) CallbackHandler(c *gin.Context) {
	session := sessions.Default(c)
	token, ok := session.Get("oauth-token").([]byte)
	if !ok || len(token) <= 0 {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid cookie oauth-token: %v %v", ok, token))
		return
	}

	var result OauthDest
	state := c.Query("state")
	err := auth.cookiegen.Decode("oauth", state, &result)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid result state: %s", err))
		return
	}

	if !bytes.Equal(token, result.Token) {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Token mismatch"))
		return
	}

	conf := auth.GetOauthConfigForContext(c)

	tok, err := conf.Exchange(oauth2.NoContext, c.Query("code"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	client := conf.Client(oauth2.NoContext, tok)
	email, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	defer email.Body.Close()
	data, _ := ioutil.ReadAll(email.Body)

	user := struct {
		Email string `json:"email"`
	}{}
	err = json.Unmarshal(data, &user)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	SetAuthenticatedUser(c, user.Email)

	c.Redirect(http.StatusSeeOther, result.Url)
}
