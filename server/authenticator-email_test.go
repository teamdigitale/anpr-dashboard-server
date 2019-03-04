package main

import (
	//"github.com/ccontavalli/goutils/templates"
	//"github.com/jordan-wright/email"
	"github.com/appleboy/gofight"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	//"gopkg.in/yaml.v2"
	"net/url"
	"strings"
	"testing"
	// "net/http"
	//    "fmt"
)

func TestAuthtenticatorMethods(t *testing.T) {
	assert := assert.New(t)

	ea, err := NewEmailAuthenticator("/form-submit", "/validate-email", "email", nil, nil, nil)
	assert.Nil(err)
	assert.NotNil(ea)

	cookie, err := ea.CreateEmailValidationCookie("foo,,PASS!Blah", "/bar,/pass,foo%20puck")
	assert.Nil(err)
	username, path, err := ea.DecodeEmailValidationCookie(cookie)
	assert.Nil(err)
	assert.Equal("foo,,PASS!Blah", username)
	assert.Equal("/bar,/pass,foo%20puck", path)

	cookie, err = ea.CreateFormSubmitCookie("fuffa,,pappa")
	assert.Nil(err)
	value, err := ea.DecodeFormSubmitCookie(cookie)
	assert.Equal("fuffa,,pappa", value)
	assert.NotEqual(cookie, value)
}

func TestAuthtenticatorWeb(t *testing.T) {
	assert := assert.New(t)
	gf := gofight.New()

	ea, err := NewEmailAuthenticator("/form-submit", "/validate-email", "email", nil, nil, nil)
	assert.Nil(err)
	assert.NotNil(ea)

	RunHandler("validate-email", gf, func(c *gin.Context) {
		path, err := ea.GetEmailUrl(c, "foo@gmail.com", "/test/path?q=mytest")

		assert.Nil(err)
		assert.True(strings.HasPrefix(path, "http://127.0.0.1/validate-email?c="))

		parsed, err := url.Parse(path)
		assert.Nil(err)

		assert.Equal("127.0.0.1", parsed.Host)
		assert.Equal("/validate-email", parsed.Path)

		values := parsed.Query()
		assert.True(strings.HasPrefix(values.Get("c"), "0:"))
	})

	RunHandler("form-submit", gf, func(c *gin.Context) {
		path, cookie, err := ea.GetFormUrl(c)
		assert.Nil(err)
		assert.Equal("/form-submit", path)
		assert.True(strings.HasPrefix(cookie, "0:"))
	})
}
