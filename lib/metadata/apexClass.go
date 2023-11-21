package metadata

import (
	"fmt"
	"io/ioutil"
	"strings"
)

func init() {
	Registry.Register("ApexClass", isApexClass, createApexClass)
}

type ApexClass struct {
	Path string
}

func isApexClass(path string) bool {
	// Detection logic
	return false
}

func createApexClass(path string) (Metadata, error) {
	// Creation logic
	return &ApexClass{Path: path}, nil
}

func (t *ApexClass) DeployedType() string {
	return "ApexClass"
}

func (t *ApexClass) Name() string {
	return ComponentName(t.Path)
}

func (t *ApexClass) dir() string {
	return "classes"
}

func (t *ApexClass) Files() (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(t.Path, t.dir())] = fileContent

	class := strings.TrimSuffix(t.Path, "-meta.xml")
	fileContent, err = ioutil.ReadFile(class)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(class, t.dir())] = fileContent
	return files, nil
}
