package vm

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ListCachedISOs returns downloaded ISO files from the vmctl ISO cache.
func ListCachedISOs() ([]string, error) {
	entries, err := os.ReadDir(ISODir())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".iso") {
			continue
		}
		paths = append(paths, filepath.Join(ISODir(), name))
	}
	sort.Strings(paths)
	return paths, nil
}
