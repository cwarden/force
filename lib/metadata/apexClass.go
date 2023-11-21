package metadata

func init() {
	Registry.Register("ApexClass", createApexClass)
}

type ApexClass struct {
	Path string
}

func createApexClass(path string) (Metadata, error) {
	return &ApexClass{Path: path}, nil
}

func (t *ApexClass) DeployedType() string {
	return "ApexClass"
}

func (t *ApexClass) Name() string {
	return ComponentName(t.Path)
}

func (t *ApexClass) dir() string {
	return "classes"
}

func (t *ApexClass) path() string {
	return t.Path
}

func (t *ApexClass) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}
