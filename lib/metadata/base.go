package metadata

import (
	"path/filepath"
	"strings"
)

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
	return metadataAndContentFiles(b)
}

func (b *BaseMetadata) DeployedName() string {
	name := strings.TrimSuffix(b.Path(), "-meta.xml")
	deployedName := b.Dir() + "/" + filepath.Base(name)
	return deployedName
}

func (b *BaseMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[b.DeployedName()] = b.Path()
	return paths
}
