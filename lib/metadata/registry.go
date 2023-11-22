package metadata

type Metadata interface {
	DeployedType() string
	Name() string
	Files() (ForceMetadataFiles, error)
	Dir() string
	path() string
}

type MetadataTypeFunc func(path string) bool
type MetadataCreateFunc func(path string) Metadata

type metadataTypeRegistry struct {
	createFuncs map[string]MetadataCreateFunc
}

var Registry = &metadataTypeRegistry{
	createFuncs: make(map[string]MetadataCreateFunc),
}

func (r *metadataTypeRegistry) ByName(name string) MetadataCreateFunc {
	return r.createFuncs[name]
}

func (r *metadataTypeRegistry) Register(metadataType string, createFunc MetadataCreateFunc) {
	r.createFuncs[metadataType] = createFunc
}

func (r *metadataTypeRegistry) RegisterBaseType(metadataType, dir string) {
	r.createFuncs[metadataType] = createBaseMetadataFunc(metadataType, dir)
}

func (r *metadataTypeRegistry) RegisterFolderedType(metadataType, dir string) {
	r.createFuncs[metadataType] = createFolderedMetadataFunc(metadataType, dir)
}

// Folder types are have their own metadata type, e.g. ReportFolder, but are
// deployed as the type of their contents, e.g. Report
func (r *metadataTypeRegistry) RegisterFolderType(metadataType, deployedType, dir string) {
	r.createFuncs[metadataType] = createFolderedMetadataFunc(deployedType, dir)
}

func createBaseMetadataFunc(deployedType, dir string) MetadataCreateFunc {
	return func(path string) Metadata {
		return &BaseMetadata{
			Path:         path,
			deployedType: deployedType,
			dir:          dir,
		}
	}
}

func createFolderedMetadataFunc(deployedType, dir string) MetadataCreateFunc {
	return func(path string) Metadata {
		return &FolderedMetadata{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: deployedType,
				dir:          dir,
			},
		}
	}
}
