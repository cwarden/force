package metadata

type CustomTab struct {
	BaseMetadata
}

func NewCustomTab(path string) Metadata {
	return &CustomTab{
		BaseMetadata: BaseMetadata{
			Path:         path,
			deployedType: "CustomTab",
			Dir:          "tabs",
		},
	}
}
