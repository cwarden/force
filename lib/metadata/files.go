package metadata

import (
	"path/filepath"
	"strings"
)

func ComponentName(path string) string {
	name := strings.TrimSuffix(path, "-meta.xml")
	if filepath.Base(name) == filepath.Ext(name) {
		return filepath.Base(name)
	}
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
}

func RelativePath(fullpath, relativeTo string) string {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(fullpath)

	// Find the index of the relativeTo part
	idx := strings.Index(normalizedPath, relativeTo)
	if idx == -1 {
		return ""
	}

	// Slice the string from the found index
	relativePath := normalizedPath[idx:]

	return relativePath
}
