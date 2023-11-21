package metadata

import (
	"fmt"
	"io/ioutil"
)

func init() {
	Registry.Register("CustomTab", isCustomTab, createCustomTab)
}

type CustomTab struct {
	Path string
}

func isCustomTab(path string) bool {
	// Detection logic
	return false
}

func (t *CustomTab) DeployedType() string {
	return "CustomTab"
}

func (t *CustomTab) Name() string {
	return ComponentName(t.Path)
}

func (t *CustomTab) dir() string {
	return "tabs"
}

func (t *CustomTab) Files() (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(t.Path, t.dir())] = fileContent
	return files, nil
}

func createCustomTab(path string) (Metadata, error) {
	// Get the file contents
	// Get the path relative to the tabs directory
	// Normalize file extension
	return &CustomTab{Path: path}, nil
}
