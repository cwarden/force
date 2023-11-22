package metadata

import (
	"fmt"
	"io/ioutil"
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

func metadataOnlyFile(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(m.path())
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(m.path(), m.Dir())] = fileContent
	return files, nil
}

func metadataAndContentFiles(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(m.path())
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(m.path(), m.Dir())] = fileContent

	class := strings.TrimSuffix(m.path(), "-meta.xml")
	fileContent, err = ioutil.ReadFile(class)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(class, m.Dir())] = fileContent
	return files, nil
}
