package metadata

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Package struct {
	path string
}

func (m *Package) Path() string {
	return m.path
}

func (m *Package) UniqueId() string {
	return m.Path()
}

func (m *Package) Files() (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(m.path)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[filepath.Base(m.path)] = fileContent
	return files, nil
}

func NewPackage(path string) Deployable {
	return &Package{
		path: path,
	}
}
func (p *Package) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[filepath.Base(p.Path())] = p.Path()
	return paths
}
