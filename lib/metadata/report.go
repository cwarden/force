package metadata

type Report struct {
	FolderedMetadata
}

func NewReport(path string) Metadata {
	return &Report{
		FolderedMetadata: FolderedMetadata{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: "Report",
				Dir:          "reports",
			},
		},
	}
}
