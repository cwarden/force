package metadata

type BaseMetadata struct {
	Path         string
	deployedType string
	Dir          string
}

func (b *BaseMetadata) Name() string {
	return ComponentName(b.Path)
}

func (b *BaseMetadata) DeployedType() string {
	return b.deployedType
}

func (b *BaseMetadata) dir() string {
	return b.Dir
}

func (b *BaseMetadata) path() string {
	return b.Path
}

func (b *BaseMetadata) Files() (ForceMetadataFiles, error) {
	return metadataOnlyFile(b)
}
