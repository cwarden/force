package metadata

type BaseMetadata struct {
	path         string
	deployedType string
	dir          string
}

func (b *BaseMetadata) Name() string {
	return ComponentName(b.path)
}

func (b *BaseMetadata) DeployedType() string {
	return b.deployedType
}

func (b *BaseMetadata) Dir() string {
	return b.dir
}

func (b *BaseMetadata) Path() string {
	return b.path
}

func (b *BaseMetadata) UniqueId() string {
	return b.Path()
}

func (b *BaseMetadata) Files() (ForceMetadataFiles, error) {
	return metadataOnlyFile(b)
}

func (b *BaseMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[MakeRelativePath(b.Path(), b.Dir())] = b.Path()
	return paths
}
