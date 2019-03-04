package main

import (
	"fmt"
	"github.com/ccontavalli/goutils/email"
	"github.com/ccontavalli/goutils/token"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// How does this work?
//
// 1) User lands on a page he has no access to.
// 2) Error page is shown, with form to enter email address, submitting to a server side api.
// 3) On form submission, generate email to the specified email address, with link.
// 4) On link click, create cookie to authenticate user.

type EmailAuthenticator struct {
	// Url to use in the submit form to cause an email to be sent.
	formsubmitpath string
	// Url to use in the email itself to validate the address.
	emailcallbackpath string
	// Name of the template to pass to email.MailSender to generate the validation email.
	template string
	// Object to send out email as.
	sender *email.MailSender
	// Encrypted cookie generator.
	tgenerator *token.TokenGenerator
	authz_mgr  *AuthorizationManager
}

func NewEmailAuthenticator(submitpath, callbackpath, template string, sender *email.MailSender, authz_mgr *AuthorizationManager, tsettings *token.TokenSettings) (*EmailAuthenticator, error) {
	if tsettings == nil {
		settings := token.DefaultTokenSettings()
		tsettings = &settings
	}
	tgenerator, err := token.NewTokenGenerator(*tsettings)
	if err != nil {
		return nil, err
	}
	return &EmailAuthenticator{submitpath, callbackpath, template, sender, tgenerator, authz_mgr}, nil
}

// Returns the URL and cookie to be used in the registration forms.
func (auth *EmailAuthenticator) GetFormUrl(c *gin.Context) (string, string, error) {
	requested_page := BuildUrlFromContext(c, "", "").String()
	cookie, err := auth.CreateFormSubmitCookie(requested_page)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%s", auth.formsubmitpath), cookie, nil
}

// Returns the URL to be used in emails.
func (auth *EmailAuthenticator) GetEmailUrl(c *gin.Context, email, page string) (string, error) {
	query, err := auth.CreateEmailValidationCookie(email, page)
	if err != nil {
		return "", err
	}
	return BuildUrlFromContext(c, auth.emailcallbackpath, "c="+url.QueryEscape(query)).String(), nil
}

// HTTP Handler for form submission.
func (auth *EmailAuthenticator) FormSubmitHandler(c *gin.Context) {
	data := struct {
		Email  string `form:"email" json:"email" binding:"required"`
		Cookie string `form:"cookie" json:"cookie" binding:"required"`
	}{}
	err := c.Bind(&data)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid form data - %v", err))
		return
	}

	page, err := auth.DecodeFormSubmitCookie(data.Cookie)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid redirect page - %v", err))
		return
	}

	if !auth.authz_mgr.EmailIsKnown(data.Email) {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid email address"))
		return
	}

	validateurl, err := auth.GetEmailUrl(c, data.Email, page)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Could not compute url - %v", err))
		return

	}
	log.Print(validateurl)

	err = auth.sender.Send(auth.template, struct{ Email, ValidateUrl string }{data.Email, validateurl}, data.Email)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Could not compute url - %v", err))
		return
	}

	c.String(http.StatusOK, "OK")
}

// HTTP Handler for validation.
func (auth *EmailAuthenticator) ValidateHandler(c *gin.Context) {
	cookie := c.Query("c")
	if cookie == "" {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid form data - missng c parameter - %s", c.Param("c")))
		return
	}

	email, url, err := auth.DecodeEmailValidationCookie(cookie)
	if err != nil {
		c.AbortWithError(http.StatusUnauthorized, fmt.Errorf("Invalid form data - %v", err))
		return
	}

	SetAuthenticatedUser(c, email)
	c.Redirect(http.StatusSeeOther, url)
}

func (auth *EmailAuthenticator) Logout(c *gin.Context) {
	ClearAuthenticatedUser(c)
	c.Redirect(http.StatusSeeOther, "/")
}

func (auth *EmailAuthenticator) CreateFormSubmitCookie(path string) (string, error) {
	query, err := auth.tgenerator.Generate(path, []string{})
	if err != nil {
		return "", err
	}
	return query, nil
}

func (auth *EmailAuthenticator) DecodeFormSubmitCookie(cookie string) (string, error) {
	page, _, err := auth.tgenerator.IsValid(cookie, []string{})
	if err != nil {
		return "", err
	}

	return page, nil
}

func (auth *EmailAuthenticator) CreateEmailValidationCookie(email, path string) (string, error) {
	if strings.Index(email, "|") >= 0 {
		return "", fmt.Errorf("Invalid character in email")
	}
	query, err := auth.tgenerator.Generate(email+"|"+path, []string{})
	if err != nil {
		return "", err
	}
	return query, nil
}

func (auth *EmailAuthenticator) DecodeEmailValidationCookie(cookie string) (string, string, error) {
	path, _, err := auth.tgenerator.IsValid(cookie, []string{})
	if err != nil {
		return "", "", err
	}
	paths := strings.SplitN(path, "|", 2)
	if len(paths) != 2 {
		return "", "", fmt.Errorf("Could not decode cookie separator")
	}
	return paths[0], paths[1], err
}
