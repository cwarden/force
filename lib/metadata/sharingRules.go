package metadata

func init() {
	Registry.Register("SharingRules", createSharingRules)
}

type SharingRules struct {
	Path string
}

func (t *SharingRules) DeployedType() string {
	return "SharingRules"
}

func (t *SharingRules) Name() string {
	return ComponentName(t.Path)
}

func (t *SharingRules) dir() string {
	return "sharingRules"
}

func (t *SharingRules) path() string {
	return t.Path
}

func (t *SharingRules) Files() (ForceMetadataFiles, error) {
	return metadataOnlyFile(t)
}

func createSharingRules(path string) (Metadata, error) {
	return &SharingRules{Path: path}, nil
}
