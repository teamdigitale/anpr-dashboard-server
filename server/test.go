package main

import (
	"encoding/json"
	"github.com/appleboy/gofight"
	"github.com/gin-gonic/gin"
)

// Return a pair of pony keys to be used in tests.
func GetTestKeys() *SecureKeys {
	var keys SecureKeys
	err := json.Unmarshal([]byte(`{"HashKey":"t9NlPSaMIU1SItEwR5FDBlorB8i35XBm28n9/po2xtk=","EncryptionKey":"SWujubjwykIH3t3aPwcRcjnekJH1k5ZACdscElZSjq8="}`), &keys)
	if err != nil {
		panic("Could not decode hard coded keys")
	}
	return &keys
}

// Provides a gin gonic context to run tests under.
func RunHandler(path string, gf *gofight.RequestConfig, handler func(c *gin.Context)) {
	RunHandlerWithCheck(path, gf, handler, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {})
}

func RunHandlerWithCheck(path string, gf *gofight.RequestConfig, handler func(c *gin.Context), checker func(r gofight.HTTPResponse, rq gofight.HTTPRequest)) {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.GET("/test-"+path, func(c *gin.Context) {
		c.Request.Host = "127.0.0.1"
		handler(c)
	})

	gf.GET("/test-"+path).Run(router, checker)
}

type MockUrl struct {
	Config       *FsPathConfig
	Fspath, Name string
	Renderer     PageRenderer
}

type MockUrlHandler struct {
	Config FsPathConfig
	Urls   []MockUrl
}

func (uh *MockUrlHandler) GetConfig() *FsPathConfig {
	return &uh.Config
}

func (uh *MockUrlHandler) RegisterFile(pc *FsPathConfig, fspath, name string, renderer PageRenderer) error {
	uh.Urls = append(uh.Urls, MockUrl{pc, fspath, name, renderer})
	return nil
}
