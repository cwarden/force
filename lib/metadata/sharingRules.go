package metadata

type SharingRules struct {
	BaseMetadata
}

func NewSharingRules(path string) Metadata {
	return &SharingRules{
		BaseMetadata: BaseMetadata{
			Path:         path,
			deployedType: "SharingRules",
			Dir:          "sharingRules",
		},
	}
}
