package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ExperienceBundleMetadata struct {
	BaseMetadata
}

func NewExperienceBundle(path string) Deployable {
	return &ExperienceBundleMetadata{
		BaseMetadata: BaseMetadata{
			path: path,
			dir:  "experiences",
		},
	}
}

func (t *ExperienceBundleMetadata) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}

func (t *ExperienceBundleMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[MakeRelativePath(t.Path(), t.Dir())] = t.Path()
	dir := strings.TrimSuffix(t.Path(), ".site-meta.xml")

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error getting Experience Bundle file: %w", err)
		}
		if info.IsDir() {
			return nil
		}
		paths[MakeRelativePath(path, t.Dir())] = path
		return nil
	})

	if err != nil {
		return paths
	}

	return paths
}
