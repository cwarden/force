package metadata

import (
	"fmt"
	"io/ioutil"
)

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

func (t *ReportFolder) DeployedType() string {
	return "Report"
}

func (t *ReportFolder) Name() string {
	return ComponentName(t.Path)
}

func (t *ReportFolder) dir() string {
	return "reports"
}

func (t *ReportFolder) Files() (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(t.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(t.Path, t.dir())] = fileContent
	return files, nil
}
