package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"net/url"
	"strings"
)

type User string
type APIKey string

type Credentials struct {
	User   *User
	APIKey APIKey
}

func (cred *Credentials) AsString() string {
	user := ""
	if cred.User != nil {
		user = string(*cred.User)
	}

	return fmt.Sprintf("User: %s / APIKey: %s", user, cred.APIKey)
}

func (auth *User) GetEmail() string {
	return string(*auth)
}

func (auth *User) GetDomain() string {
	email := string(*auth)
	i := strings.LastIndex(email, "@")
	if i < 0 {
		return ""
	}

	return email[i:]
}

func SetAuthenticatedUser(c *gin.Context, email string) {
	session := sessions.Default(c)
	session.Set("id", email)
	session.Save()
}

func ClearAuthenticatedUser(c *gin.Context) {
	session := sessions.Default(c)
	session.Delete("id")
	session.Save()
}

func GetAuthenticatedUser(c *gin.Context) *User {
	session := sessions.Default(c)
	id, ok := session.Get("id").(string)
	if !ok {
		return nil
	}
	return (*User)(&id)
}

func GetAPIKey(c *gin.Context) APIKey {
	apikey := c.Request.Header.Get("APIKey")
	return (APIKey)(apikey)
}

func GetCredentials(c *gin.Context) *Credentials {
	return &Credentials{User: GetAuthenticatedUser(c), APIKey: GetAPIKey(c)}
}

// Returns the URL of the request in the gin.Context. If path is
// not the empty string, overrides the path as specified.
func BuildUrlFromContext(c *gin.Context, path string, query string) *url.URL {
	url := c.Request.URL
	url.Opaque = ""
	url.Host = c.Request.Host
	if path != "" {
		url.Path = path
		url.RawPath = ""
		url.RawQuery = ""
		url.Fragment = ""
	}
	if query != "" {
		url.RawQuery = query
	}
	url.Scheme = "http"
	if c.Request.TLS != nil {
		url.Scheme = "https"
	}

	return url
}
