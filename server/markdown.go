package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/russross/blackfriday"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type MdHandler struct {
}

type PageProperties struct {
	Title    string
	Subtitle string

	Contacts []string

	FsPathConfig `yaml:",inline"`
}

type MdPage struct {
	content    []byte
	properties *PageProperties
}

func NewMdHandler() *MdHandler {
	return &MdHandler{}
}

func parseProperties(content []byte) (error, *PageProperties, []byte) {

	start := 0
	end := 0

	// Find --- marker, skip empty lines.
	for {
		if start >= len(content) {
			return nil, nil, content
		}

		// Find end of line.
		end = bytes.IndexByte(content[start:], '\n')
		if end < 0 {
			end = len(content)
		} else {
			end += start
		}

		// Did we find the marker?
		line := bytes.TrimSpace(content[start:end])
		start = end + 1

		if bytes.Equal(line, []byte("---")) {
			break
		}

		// Empty line? yes keep going, no return now.
		if !bytes.Equal(line, []byte("")) {
			return nil, nil, content
		}
	}

	ystart := start
	for start < len(content) {
		// Find end of line.
		end = bytes.IndexByte(content[start:], '\n')
		if end < 0 {
			end = len(content)
		} else {
			end += start
		}

		// Did we find the marker?
		line := bytes.TrimSpace(content[start:end])

		if bytes.Equal(line, []byte("---")) {
			break
		}
		start = end + 1
	}
	yend := start

	var mdprops PageProperties
	err := yaml.Unmarshal(content[ystart:yend], &mdprops)
	if err != nil {
		return err, nil, content
	}
	return nil, &mdprops, content[end+1 : len(content)]
}

func StripExtensions(name string) string {
	extindex := strings.Index(name, ".")
	if extindex < 0 {
		return name
	}
	return name[:extindex]
}

func (processor *MdHandler) Prepare(fspath string, file os.FileInfo, uh UrlHandler) error {
	content, err := ioutil.ReadFile(fspath)
	if err != nil {
		return err
	}

	err, mdprops, content := parseProperties(content)
	if err != nil {
		return err
	}
	var pc *FsPathConfig
	if mdprops != nil {
		pc = &mdprops.FsPathConfig
	}

	page := &MdPage{content, mdprops}
	return uh.RegisterFile(pc, fspath, StripExtensions(file.Name()), page)
}

func (mp *MdPage) Render(c *gin.Context) error {
	toc := template.HTML(mp.GetToc())
	content := template.HTML(mp.GetContent())
	properties := mp.GetProperties()
	vcs := &VcsProperties{}

	c.HTML(http.StatusOK, "document", gin.H{"index": toc, "content": content, "properties": properties, "vcs": &vcs})
	return nil
}

var commonExtensions int = 0 |
	blackfriday.EXTENSION_NO_INTRA_EMPHASIS |
	blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_FOOTNOTES |
	blackfriday.EXTENSION_HEADER_IDS |
	blackfriday.EXTENSION_AUTO_HEADER_IDS |
	blackfriday.EXTENSION_BACKSLASH_LINE_BREAK |
	blackfriday.EXTENSION_DEFINITION_LISTS
var commonHtmlFlags int = 0 |
	blackfriday.HTML_USE_XHTML |
	blackfriday.HTML_USE_SMARTYPANTS |
	blackfriday.HTML_SMARTYPANTS_DASHES |
	blackfriday.HTML_SMARTYPANTS_LATEX_DASHES |
	// blackfriday.HTML_SKIP_HTML |
	blackfriday.HTML_NOFOLLOW_LINKS

	// This is too aggressive, blocks here references, like #down-below.
	// blackfriday.HTML_SAFELINK |

	// Although useful, this gets very annoying:
	// turns text like dl 50/2016 into a fraction.
	// blackfriday.HTML_SMARTYPANTS_FRACTIONS |

func (page *MdPage) GetToc() string {
	// TODO: parse once, create both toc and index in one go.
	// TODO: creating toc could be done much more efficiently without parsing multiple times.
	renderer := blackfriday.HtmlRenderer(commonHtmlFlags|blackfriday.HTML_TOC|blackfriday.HTML_OMIT_CONTENTS, "", "")
	output := blackfriday.MarkdownOptions(page.content, renderer, blackfriday.Options{Extensions: commonExtensions})
	return string(output)
}

func (page *MdPage) GetContent() string {
	renderer := blackfriday.HtmlRenderer(commonHtmlFlags, "", "")
	output := blackfriday.MarkdownOptions(page.content, renderer, blackfriday.Options{Extensions: commonExtensions})
	return string(output)
}

func (page *MdPage) GetProperties() *PageProperties {
	if page.properties != nil {
		return page.properties
	}
	return &PageProperties{}
}
