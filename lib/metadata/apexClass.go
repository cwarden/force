package metadata

type ApexClass struct {
	BaseMetadata
}

func NewApexClass(path string) Metadata {
	return &ApexClass{
		BaseMetadata: BaseMetadata{
			Path:         path,
			deployedType: "ApexClass",
			Dir:          "classes",
		},
	}
}

func (t *ApexClass) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}
