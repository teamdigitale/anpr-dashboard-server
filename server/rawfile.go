package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type RawFileHandler struct {
}

type RawFile struct {
	fspath  string
	modtime time.Time

	ctype string

	content    *bytes.Reader
	compressed *bytes.Reader
}

// FIXME: move into a configuration parameter
var gl_SkipCompressionFor = regexp.MustCompile(`gif|png|jpeg|mpeg|mp4|quicktime|zip|rar|compressed|java-archive|xz`)

func NewRawFileHandler() *RawFileHandler {
	return &RawFileHandler{}
}

func (rh *RawFileHandler) Prepare(fspath string, file os.FileInfo, uh UrlHandler) error {
	content, err := ioutil.ReadFile(fspath)
	if err != nil {
		return err
	}

	ctype := mime.TypeByExtension(filepath.Ext(file.Name()))
	if ctype == "" {
		ctype = http.DetectContentType(content)
	}

	// FIXME: move compression support outside rawfile, so all handlers can benefit.
	var compressed *bytes.Reader
	if !gl_SkipCompressionFor.MatchString(ctype) {
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		writer.Write(content)
		writer.Close()
		if len(buffer.Bytes()) < len(content) {
			compressed = bytes.NewReader(buffer.Bytes())
		} else {
			// Really, if a file is small, the outcome of the compression might not
			// depend on the format at all. There might just be too little data to make
			// any conclusion.
			if len(content) >= 10240 {
				err = fmt.Errorf("Recommend adding MIME type %s to list of non-compressable types", ctype)
			}
		}
	}

	renderer := &RawFile{fspath, file.ModTime(), ctype, bytes.NewReader(content), compressed}
	return uh.RegisterFile(nil, fspath, file.Name(), renderer)
}

func acceptsGzip(request *http.Request) bool {
	accepts := request.Header.Get("Accept-Encoding")
	index := strings.Index(accepts, "gzip")
	if index < 0 {
		return false
	}

	left := accepts[index+len("gzip"):]
	if !strings.HasPrefix(left, ";q=0") {
		return true
	}
	left = left[len(";q=0"):]
	for i := 0; ; i++ {
		if i >= len(left) {
			return true
		}
		if left[i] == '.' {
			continue
		}

		if left[i] < '0' && left[i] > '9' {
			return true
		}
		if left[i] != '0' {
			return false
		}
	}

	// Never reached
	return false
}

func (rf *RawFile) Render(c *gin.Context) error {
	c.Writer.Header().Set("Content-Type", rf.ctype)

	if rf.compressed != nil && acceptsGzip(c.Request) {
		c.Writer.Header().Set("Content-Encoding", "gzip")
		c.Writer.Header().Set("Vary", "Accept-Encoding")
		http.ServeContent(c.Writer, c.Request, rf.fspath, rf.modtime, rf.compressed)
	} else {
		http.ServeContent(c.Writer, c.Request, rf.fspath, rf.modtime, rf.content)
	}
	return nil
}
