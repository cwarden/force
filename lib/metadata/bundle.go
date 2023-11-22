package metadata

type BundledMetadata struct {
	BaseMetadata
}

func (t *BundledMetadata) Files() (ForceMetadataFiles, error) {
	return allFilesInFolder(t)
}
