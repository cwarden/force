package metadata

func init() {
	Registry.Register("ReportFolder", createReportFolder)
}

type ReportFolder struct {
	Path string
}

func createReportFolder(path string) (Metadata, error) {
	// Creation logic
	return &ReportFolder{Path: path}, nil
}

func (t *ReportFolder) DeployedType() string {
	return "Report"
}

func (t *ReportFolder) Name() string {
	return ComponentName(t.Path)
}

func (t *ReportFolder) dir() string {
	return "reports"
}

func (t *ReportFolder) path() string {
	return t.Path
}

func (t *ReportFolder) Files() (ForceMetadataFiles, error) {
	return metadataOnlyFile(t)
}
