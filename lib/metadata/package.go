package metadata

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Package struct {
	Path string
}

func (m *Package) path() string {
	return m.Path
}

func (m *Package) Files() (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(m.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[filepath.Base(m.Path)] = fileContent
	return files, nil
}

func NewPackage(path string) Deployable {
	return &Package{
		Path: path,
	}
}
