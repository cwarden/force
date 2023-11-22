package metadata

type ApexTrigger struct {
	BaseMetadata
}

func NewApexTrigger(path string) Metadata {
	return &ApexTrigger{
		BaseMetadata: BaseMetadata{
			Path:         path,
			deployedType: "ApexTrigger",
			dir:          "triggers",
		},
	}
}

func (t *ApexTrigger) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}
