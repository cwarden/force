package metadata

// Metadata with separate content files: ApexClass, ApexTrigger,
// WaveDataflow, WaveRecipe
type ContentMetadata struct {
	BaseMetadata
}

func (t *ContentMetadata) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}
