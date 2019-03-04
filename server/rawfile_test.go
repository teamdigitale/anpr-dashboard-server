package main

import (
	//"github.com/ccontavalli/goutils/templates"
	//"github.com/jordan-wright/email"
	"github.com/appleboy/gofight"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	//"gopkg.in/yaml.v2"
	//"encoding/json"
	//"net/url"
	//"strings"
	"os"
	"testing"
	// "net/http"
	//    "fmt"
)

func TestRawfile(t *testing.T) {
	assert := assert.New(t)
	gf := gofight.New()
	rf := NewRawFileHandler()
	uh := MockUrlHandler{}

	assert.NotNil(rf.Prepare("/file/that/does/not/exist", nil, nil))

	testfile := "./test/image.png"
	info, err := os.Lstat(testfile)

	assert.Nil(err)
	assert.Nil(rf.Prepare(testfile, info, &uh))

	assert.Equal(1, len(uh.Urls))
	assert.NotNil(uh.Urls[0].Renderer)
	assert.Equal(testfile, uh.Urls[0].Fspath)
	assert.Equal("image.png", uh.Urls[0].Name)

	RunHandlerWithCheck("rawfile", gf, func(c *gin.Context) {
		uh.Urls[0].Renderer.Render(c)
	}, func(resp gofight.HTTPResponse, req gofight.HTTPRequest) {
		//assert.Equal("a", resp.Header().Get("Content-Type"))
		//assert.Equal("b", resp.Header().Get("Content-Encoding"))
		//assert.Equal("c", resp.Header().Get("Vary"))
	})
}
