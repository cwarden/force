package metadata

func init() {
	Registry.Register("CustomTab", createCustomTab)
}

type CustomTab struct {
	Path string
}

func (t *CustomTab) DeployedType() string {
	return "CustomTab"
}

func (t *CustomTab) Name() string {
	return ComponentName(t.Path)
}

func (t *CustomTab) dir() string {
	return "tabs"
}

func (t *CustomTab) path() string {
	return t.Path
}

func (t *CustomTab) Files() (ForceMetadataFiles, error) {
	return metadataOnlyFile(t)
}

func createCustomTab(path string) (Metadata, error) {
	return &CustomTab{Path: path}, nil
}
