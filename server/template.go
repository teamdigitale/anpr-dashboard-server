package main

import (
	"bytes"
	"github.com/ccontavalli/goutils/templates"
	"github.com/gin-gonic/gin"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
)

type TemplateHandler struct {
	parent *templates.StaticTemplates
}

func NewTemplateHandler(base *templates.StaticTemplates) *TemplateHandler {
	return &TemplateHandler{base}
}

type TemplateRenderer template.Template

func (self *TemplateRenderer) Render(c *gin.Context) error {
	buffer := bytes.Buffer{}
	err := (*template.Template)(self).ExecuteTemplate(&buffer, "start", struct{}{})
	if err != nil {
		return err
	}

	c.Data(http.StatusOK, "text/html", buffer.Bytes())
	return nil
}

func (th *TemplateHandler) Prepare(fspath string, file os.FileInfo, uh UrlHandler) error {
	content, err := ioutil.ReadFile(fspath)
	if err != nil {
		return err
	}

	loader := templates.NewStaticTemplatesFromParent(th.parent)
	name, err := loader.Parse(fspath, content)
	if err != nil {
		return err
	}

	template := loader.Get(name)
	return uh.RegisterFile(nil, fspath, file.Name(), (*TemplateRenderer)(template))
}
