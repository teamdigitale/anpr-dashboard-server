package main

import (
	"github.com/ccontavalli/goutils/config"
	"github.com/ccontavalli/goutils/misc"
	"sort"
	//"log"
)

type AccessConfig struct {
	// Groups allowed to view the page / url / path.
	Viewers []GroupID
}

func (ac AccessConfig) GetReaders() []GroupID {
	return ac.Viewers
}

func (ac AccessConfig) GetWriters() []GroupID {
	return make([]GroupID, 0)
}

type GuessRegistrar struct {
}
type RegexpRegistrar struct {
	Match   string
	Replace string
}
type StaticRegistrar struct {
	BaseUrl string
}

// Configuration to be used for this path / file / directory.
type FsPathConfig struct {
	AccessConfig `yaml:",inline"`

	// Index files to use in this directory.
	Indexes []string
	// Pattern of files and/or directories to skip. Regexpps are ok.
	Skip []string
	// Extensions to strip.
	StrippedExtensions []string

	// How to register this path and subpaths.
	GuessRegistrar  *GuessRegistrar
	RegexpRegistrar []*RegexpRegistrar
	StaticRegistrar []*StaticRegistrar
}

func mergeAndSortOverrideableList(base, overlay []string) []string {
	result := mergeOverrideableList(base, overlay)
	sort.Strings(result)
	return misc.SortedDedup(result)
}

func mergeOverrideableList(base, overlay []string) []string {
	if len(overlay) >= 1 && overlay[0] == "=" {
		return overlay[1:]
	}

	if len(base) >= 1 && base[0] == "!" && len(overlay) >= 1 {
		return overlay
	}

	return append(base, overlay...)
}

// Returns a new configuration resulting by merging this config with the supplied one.
// This object (pc) is the one that takes priority.
func (pc *FsPathConfig) Merge(source FsPathConfig) *FsPathConfig {
	result := FsPathConfig{}

	if len(pc.Viewers) != 0 {
		result.Viewers = pc.Viewers
	} else {
		result.Viewers = source.Viewers
	}
	result.Skip = mergeAndSortOverrideableList(source.Skip, pc.Skip)
	result.StrippedExtensions = mergeAndSortOverrideableList(source.StrippedExtensions, pc.StrippedExtensions)
	result.Indexes = mergeOverrideableList(source.Indexes, pc.Indexes)

	return &result
}

func NewFsPathConfigFromFile(filename string) (*FsPathConfig, error) {
	result := FsPathConfig{}
	err := config.ReadYamlConfigFromFile(filename, &result)
	return &result, err
}

type ProxyPathConfig struct {
	AccessConfig `yaml:",inline"`

	// URL to forward the request to.
	BaseUrl []string
}

type ServerConfig struct {
	// List of hostnames to obtain a certificate for.
	Hostnames []string

	// Options for the UrlManager.
	Options *UrlManagerOptions

	StorageOptions *StorageOptions

	Groups []Group
}

func NewServerConfigFromFile(filename string) (*ServerConfig, error) {
	result := ServerConfig{}
	err := config.ReadYamlConfigFromFile(filename, &result)
	return &result, err
}
