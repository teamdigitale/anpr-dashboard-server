package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ccontavalli/goutils/misc"
	"github.com/ccontavalli/goutils/scanner"
	"github.com/ccontavalli/goutils/templates"
	"github.com/gin-gonic/gin"
	iradix "github.com/hashicorp/go-immutable-radix"
)

type FileState struct {
	fspath string

	config   *FsPathConfig
	renderer PageRenderer
}

type DirState struct {
	fspath string
	root   string

	urlfiles map[string]*FileState
}

type PageRenderer interface {
	Render(c *gin.Context) error
}

type TypeHandler interface {
	Prepare(fspath string, file os.FileInfo, uh UrlHandler) error
}

type WebLocation struct {
	IndexScore int

	Index *FileState
	Dir   *DirState
}

type UrlManagerOptions struct {
	FsPathConfigFileName string
	DefaultFsPathConfig  *FsPathConfig

	// Time to wait before rescanning the tree.
	Sleep time.Duration

	// Directory where to store SSL certificates and caches.
	SSLCacheDir string

	// Paths to scan for content.
	// The key is a file system path containing files to export.
	Paths map[string]*FsPathConfig

	// Paths to map to remote servers.
	// The key is a remote url to export as a local url.
	Urls map[string]*ProxyPathConfig
}

type ScanError struct {
	FsPath string
	Error  string
}
type UrlSet struct {
	tree   *iradix.Tree
	errors []ScanError
}

type UrlManagerAuthenticators struct {
	googleauth *OauthAuthenticator
	githubauth *OauthAuthenticator
	emailauth  *EmailAuthenticator
}

type UrlManager struct {
	options        UrlManagerOptions
	authenticators UrlManagerAuthenticators
	authz_mgr      *AuthorizationManager

	SuffixHandler  map[string]TypeHandler
	DefaultHandler TypeHandler

	ulock  sync.RWMutex
	urlset *UrlSet
}

func DefaultUrlManagerOptions() UrlManagerOptions {
	options := UrlManagerOptions{
		FsPathConfigFileName: ".goospel.config",
		DefaultFsPathConfig: (&FsPathConfig{}).Merge(FsPathConfig{
			AccessConfig:       AccessConfig{Viewers: []GroupID{"!", "."}},
			Indexes:            []string{"index.html", "README.md", "INDEX.md", ".html", ".md"},
			Skip:               []string{".git", ".svn", ".cvs", ".aws", ".hg"},
			StrippedExtensions: []string{".html", ".shtml"},
		}),
		Sleep:       60 * time.Second,
		SSLCacheDir: "sslcache",
		Paths:       make(map[string]*FsPathConfig),
		Urls:        make(map[string]*ProxyPathConfig),
	}

	return options
}

// Returns a new configuration resulting by merging this config with the supplied one.
// This object (uo) is the one that takes priority.
func (uo *UrlManagerOptions) Merge(source UrlManagerOptions) *UrlManagerOptions {
	result := UrlManagerOptions{}
	if uo.FsPathConfigFileName != "" {
		result.FsPathConfigFileName = uo.FsPathConfigFileName
	} else {
		result.FsPathConfigFileName = source.FsPathConfigFileName
	}
	if uo.SSLCacheDir != "" {
		result.SSLCacheDir = uo.SSLCacheDir
	} else {
		result.SSLCacheDir = source.SSLCacheDir
	}
	if uo.DefaultFsPathConfig != nil {
		result.DefaultFsPathConfig = uo.DefaultFsPathConfig
	} else {
		result.DefaultFsPathConfig = source.DefaultFsPathConfig
	}
	if uo.Sleep != 0 {
		result.Sleep = uo.Sleep
	} else {
		result.Sleep = source.Sleep
	}

	result.Paths = make(map[string]*FsPathConfig)
	for key, value := range source.Paths {
		result.Paths[key] = value
	}
	for key, value := range uo.Paths {
		result.Paths[key] = value
	}
	result.Urls = make(map[string]*ProxyPathConfig)
	for key, value := range source.Urls {
		result.Urls[key] = value
	}
	for key, value := range uo.Urls {
		result.Urls[key] = value
	}
	return &result
}

func (r *UrlManager) RequireAuthentication(c *gin.Context, template string) {
	request := c.Request.URL.String()
	gauth := ""
	var err error
	if r.authenticators.googleauth != nil {
		gauth, err = r.authenticators.googleauth.GetUrl(c, request)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
	}

	eauth := ""
	ecookie := ""
	if r.authenticators.emailauth != nil {
		eauth, ecookie, err = r.authenticators.emailauth.GetFormUrl(c)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return

		}
	}

	props := PageProperties{}
	vcs := VcsProperties{}
	c.HTML(http.StatusForbidden, template, gin.H{"properties": &props, "vcs": &vcs, "gauth": gauth, "eauth": eauth, "ecookie": ecookie})
}

func NewUrlManager(options UrlManagerOptions, authenticators UrlManagerAuthenticators, authz_mgr *AuthorizationManager, templates *templates.StaticTemplates) (*UrlManager, error) {
	manager := UrlManager{
		options:        options,
		authenticators: authenticators,
		authz_mgr:      authz_mgr,
	}

	manager.SuffixHandler = make(map[string]TypeHandler)
	// Register page handlers.
	manager.SuffixHandler[".md"] = NewMdHandler()
	manager.SuffixHandler[".tpl.html"] = NewTemplateHandler(templates)
	manager.DefaultHandler = NewRawFileHandler()

	go func() {
		for {
			manager.Update()
			time.Sleep(manager.options.Sleep)
		}
	}()
	return &manager, nil
}

func (um *UrlManager) Process(c *gin.Context) {
	// Path is already unescaped, but we need to get rid of any ../.././
	// or similar that might escape the DocumentRoot.
	original_path := c.Request.URL.Path
	clean_path := []byte(path.Clean(original_path))

	// urlset.tree is constant, we use pointer swapping to update it.
	// Garbage collector is thread safe, so we only need to grab a reference
	// under lock, to prevent the swap from happening while accessing the pointer.
	um.ulock.RLock()
	treeref := um.urlset.tree
	um.ulock.RUnlock()

	match, it, found := treeref.Root().LongestPrefix(clean_path)
	if !found {
		// FIXME: page not found at all error
		return
	}

	wp := it.(*WebLocation)

	// This is the path relative to what was inserted in the prefix tree.
	sub_path := bytes.Trim(bytes.TrimPrefix(clean_path, match), "/")

	// Before even checking if the user is allowed to access the path,
	// make sure that there is a trailing slash, so relative paths in the
	// html, js or css files work as expected.
	if len(sub_path) == 0 && len(wp.Dir.urlfiles) > 0 && original_path[len(original_path)-1] != '/' {
		c.Redirect(http.StatusMovedPermanently, string(clean_path)+"/")
		return
	}

	filestate := wp.Index
	if len(sub_path) != 0 {
		filestate = wp.Dir.urlfiles[string(sub_path)]
	}
	if filestate == nil || filestate.renderer == nil {
		// FIXME: path not found
		return
	}

	// Check that the user can access the file and/or directory.
	credentials := GetCredentials(c)
	if !um.authz_mgr.HasReadAccess(filestate.config, credentials) {
		if !um.authz_mgr.IsPubliclyViewable(filestate.config) && credentials.User == nil {
			um.RequireAuthentication(c, "error-must-login")
		} else {
			um.RequireAuthentication(c, "error-not-authorized")
		}
		return
	}

	filestate.renderer.Render(c)
}

func (um *UrlManager) OnlyLoggedHandler(h func(*gin.Context)) func(*gin.Context) {
	return func(c *gin.Context) {
		credentials := GetCredentials(c)
		if credentials.User != nil || credentials.APIKey != "" {
			h(c)
			return
		}
		um.RequireAuthentication(c, "error-must-login")
		return
	}
}

func (um *UrlManager) Debug(c *gin.Context) {
	um.ulock.RLock()
	writer := bytes.Buffer{}
	// writer := bufio.NewWriter(&buffer)

	if len(um.urlset.errors) > 0 {
		fmt.Fprintf(&writer, "Errors:\n")

		for i, se := range um.urlset.errors {
			fmt.Fprintf(&writer, "  [%d] - %s: %s\n", i, se.FsPath, se.Error)
		}
	}

	fmt.Fprintf(&writer, "Paths:\n")
	it := um.urlset.tree.Root().Iterator()
	for {
		prefix, node, found := it.Next()
		if !found {
			break
		}

		fmt.Fprintf(&writer, "  prefix: %s\n", prefix)

		wl := node.(*WebLocation)
		fmt.Fprintf(&writer, "    index: score %d - %p\n", wl.IndexScore, wl.Index)
		if wl.Index != nil {
			fmt.Fprintf(&writer, "      fspath: %s\n", wl.Index.fspath)
			fmt.Fprintf(&writer, "      config: %p - %v\n", wl.Index.config, wl.Index.config)
			fmt.Fprintf(&writer, "      renderer: %p\n", wl.Index.renderer)
		}
		fmt.Fprintf(&writer, "    dir: %p\n", wl.Dir)

		if wl.Dir != nil {
			fmt.Fprintf(&writer, "      directory: %s\n", wl.Dir.fspath)
			fmt.Fprintf(&writer, "      root: %s\n", wl.Dir.root)
			fmt.Fprintf(&writer, "      urls:\n")

			for url, fstate := range wl.Dir.urlfiles {
				fmt.Fprintf(&writer, "        url: %s %p - %+v\n", url, fstate, fstate)
				if fstate != nil {
					fmt.Fprintf(&writer, "         config: %+v\n", fstate.config)
				}
			}
		}
	}
	um.ulock.RUnlock()

	// writer.Flush()
	c.Data(http.StatusOK, "text/plain", writer.Bytes())
}

type UrlHandler interface {
	GetConfig() *FsPathConfig
	RegisterFile(pc *FsPathConfig, fspath, name string, renderer PageRenderer) error
}

type urlHandler struct {
	txn    *iradix.Txn
	dir    *DirState
	config *FsPathConfig
}

func (uh *urlHandler) GetConfig() *FsPathConfig {
	return uh.config
}

// Registers a file to be served via web.
// pc is an optional config to be used specifically for this file.
// fspath is the path on the filesystem where the file can be read.
// filename is the desired name for the file.
// renderer is the callback to be used to render the file.
func (uh *urlHandler) RegisterFile(pc *FsPathConfig, fspath, name string, renderer PageRenderer) error {
	// Pseudo code:
	// 1) determine which FsPathConfig to use, and if they are compatible.
	// 2) for the fspath, compute an urlpath to use?
	// 3) register the file for all RegisterAs parameters, and urlpath above.
	// 4) elect the directory index

	// MakeName returns a directory name.
	// urlpath := MakeName(fspath)

	// Note that:
	// - uh.dir.config -> is the config for the directory currently scanned.
	// - pc -> is the config specific to this file, extracted from the file (if any).
	if pc == nil {
		pc = uh.config
	} else {
		// TODO: check that the configs are compatible?
		pc = pc.Merge(*uh.config)
	}

	dirname := strings.TrimPrefix(path.Dir(fspath), uh.dir.root)
	filename := path.Base(fspath)

	var wl *WebLocation
	dh, found := uh.txn.Get([]byte(dirname))

	if found {
		wl = dh.(*WebLocation)
	} else {
		wl = &WebLocation{math.MaxInt32, nil, uh.dir}
		uh.txn.Insert([]byte(dirname), wl)
	}
	// Create the file state.
	fstate := FileState{fspath, pc, renderer}

	// Check if this file should be used as an index.
	score := StringSuffixIndex(pc.Indexes, filename)
	if score < 0 {
		score = math.MaxInt32
	}
	if score < wl.IndexScore {
		wl.Index = &fstate
		wl.IndexScore = score
	}

	if wl.Dir.urlfiles == nil {
		wl.Dir.urlfiles = make(map[string]*FileState)
	}

	for {
		extension := filepath.Ext(name)
		if len(extension) == 0 {
			break
		}
		if !misc.SortedHasString(pc.StrippedExtensions, extension) {
			break
		}
		name = name[:len(name)-len(extension)]
	}

	if _, ok := wl.Dir.urlfiles[name]; ok {
		return fmt.Errorf("FILE REGISTRATION - %s under %s is already registered", name, dirname)
	}

	wl.Dir.urlfiles[name] = &fstate
	//log.Print("REGISTERING FILE ", fspath, " ", dirname, " ", filename, " ", name, " ", wl.State.urlfiles, " ", uh.dir.root)
	return nil
}

func (um *UrlManager) Update() {
	urlset := UrlSet{iradix.New(), make([]ScanError, 0)}

	txn := urlset.tree.Txn()

	defaultconfig := um.options.DefaultFsPathConfig
	for root, config := range um.options.Paths {
		root := filepath.Clean(root)

		if config == nil {
			config = defaultconfig
		} else {
			config = config.Merge(*defaultconfig)
		}

		uh := urlHandler{txn, &DirState{root, root, nil}, config}

		scanner.ScanTree(root, &uh,
			func(gstate interface{}, fspath string, info os.FileInfo, files []os.FileInfo) (interface{}, error) {
				state := gstate.(*urlHandler)

				if info != nil && misc.SortedHasString(state.config.Skip, info.Name()) {
					return nil, fmt.Errorf("Skipping directory %s - %s by config", fspath, info.Name())
				}

				newdir := *state.dir
				newdir.fspath = fspath
				newdir.urlfiles = nil

				newstate := *state
				newstate.dir = &newdir

				configfile := Find(files, um.options.FsPathConfigFileName)
				if configfile != nil {
					newconfig, err := NewFsPathConfigFromFile(path.Join(fspath, configfile.Name()))
					if err != nil {
						return nil, err
					}
					newstate.config = newconfig.Merge(*state.config)
				}

				return &newstate, nil
			},
			func(gstate interface{}, path string, file os.FileInfo) error {
				uh := gstate.(*urlHandler)

				handler := um.DefaultHandler
				for suffix, lhandler := range um.SuffixHandler {
					if strings.HasSuffix(path, suffix) {
						handler = lhandler
						break
					}
				}

				return handler.Prepare(path, file, uh)
			},
			func(gstate interface{}, fspath string, err error) {
				log.Printf("ERROR %s, %s", fspath, err)
				urlset.errors = append(urlset.errors, ScanError{fspath, err.Error()})
			},
		)
	}

	urlset.tree = txn.Commit()

	um.ulock.Lock()
	um.urlset = &urlset
	um.ulock.Unlock()
}
