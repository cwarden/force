package metadata

import "strings"

// Metadata with separate content files: ApexClass, ApexTrigger,
// WaveDataflow, WaveRecipe
type ContentMetadata struct {
	BaseMetadata
}

func (t *ContentMetadata) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}

func (b *ContentMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[RelativePath(b.Path(), b.Dir())] = b.Path()
	content := strings.TrimSuffix(b.Path(), "-meta.xml")
	paths[RelativePath(content, b.Dir())] = content
	return paths
}
