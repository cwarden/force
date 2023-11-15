package metadata

func init() {
	Registry.Register("ReportFolder", isReportFolder, createReportFolder)
}

type ReportFolder struct {
	Path string
}

func isReportFolder(path string) bool {
	// Detection logic
	return false
}

func createReportFolder(path string) (Metadata, error) {
	// Creation logic
	return &ReportFolder{Path: path}, nil
}
