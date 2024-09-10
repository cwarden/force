package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ComponentName(path string) string {
	name := strings.TrimSuffix(path, "-meta.xml")
	if filepath.Base(name) == filepath.Ext(name) {
		return filepath.Base(name)
	}
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
}

func MakeRelativePath(fullpath, relativeTo string) string {
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

func metadataOnlyFile(m DeployableMetadata) (ForceMetadataFiles, error) {
	return metadataAndContentFiles(m)
}

func metadataAndContentFiles(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	for relative, fullPath := range m.Paths() {
		fileContent, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("Could not read file %s: %w", fullPath, err)
		}
		files[relative] = fileContent
	}
	return files, nil
}

var lwcJsTestFile = regexp.MustCompile(".*\\.test\\.js$")
