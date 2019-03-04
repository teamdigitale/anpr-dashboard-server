package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func StringSuffixIndex(names []string, tofind string) int {
	for index, name := range names {
		if strings.HasSuffix(tofind, name) {
			return index
		}
	}
	return -1
}

// Uses some sort of heuristic to make up a reasonable name for the file.
//
// Rules:
//   - ccontavalli/ssh-ident/docs/README.md -> ssh-ident?
//   - ccontavalli/ssh-ident/docs/test.md -> ssh-ident-test?
//   - ccontavalli/README.md -> ccontavalli?
//   - ccontavalli/test.md -> ccontavalli-test?
//   - ccontavalli/docs/e-procurement/README.md -> e-procurement?
func MakeName(rel string) string {
	elements := strings.Split(rel, string(os.PathSeparator))
	suffix := ""
	filename := elements[len(elements)-1]
	elements = elements[:len(elements)-1]

	if filename != "README.md" {
		suffix = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	prefix := ""
	for i := len(elements) - 1; i >= 0; i-- {
		element := elements[i]
		if i == 0 || strings.Contains(element, "-") {
			prefix = element
			break
		}
	}

	result := ""
	if prefix != "" {
		if suffix != "" {
			result = prefix + "-" + suffix
		} else {
			result = prefix
		}
	} else {
		result = suffix
	}
	return strings.ToLower(result)
}

func Find(files []os.FileInfo, name string) os.FileInfo {
	result := sort.Search(len(files), func(i int) bool { return files[i].Name() >= name })
	if result < len(files) && files[result].Name() == name {
		return files[result]
	}

	return nil
}
