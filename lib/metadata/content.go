package metadata

import (
	"os"
	"path/filepath"
	"strings"
)

// Metadata with separate content files: ApexClass, ApexTrigger,
// WaveDataflow, WaveRecipe, Static Resources
type ContentMetadata struct {
	BaseMetadata
}

func (t *ContentMetadata) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}

func (b *ContentMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[MakeRelativePath(b.Path(), b.Dir())] = b.Path()

	content := strings.TrimSuffix(b.Path(), "-meta.xml")
	if _, err := os.Stat(content); err == nil {
		paths[MakeRelativePath(content, b.Dir())] = content
		return paths
	}
	contentGlob := strings.TrimSuffix(b.Path(), ".resource-meta.xml") + ".*"
	if matches, _ := filepath.Glob(contentGlob); len(matches) == 2 {
		for _, m := range matches {
			if m == b.Path() {
				continue
			}
			paths[MakeRelativePath(content, b.Dir())] = m
			return paths
		}
	}
	return paths
}
