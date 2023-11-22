package metadata

type DeployableMetadata interface {
	Deployable
	DeployedType() string
	Name() string
	Dir() string
	path() string
}

// destructiveChanges.xml is deployable, but is not metadata
type Deployable interface {
	Files() (ForceMetadataFiles, error)
}

type DeployableCreateFunc func(path string) Deployable
type MetadataCreateFunc func(path string) DeployableMetadata

type deployableTypeRegistry struct {
	createFuncs map[string]DeployableCreateFunc
}

var Registry = &deployableTypeRegistry{
	createFuncs: make(map[string]DeployableCreateFunc),
}

func (r *deployableTypeRegistry) ByName(name string) DeployableCreateFunc {
	return r.createFuncs[name]
}

func (r *deployableTypeRegistry) Register(metadataType string, createFunc DeployableCreateFunc) {
	r.createFuncs[metadataType] = createFunc
}

func (r *deployableTypeRegistry) RegisterBaseType(metadataType, dir string) {
	r.createFuncs[metadataType] = createBaseMetadataFunc(metadataType, dir)
}

func (r *deployableTypeRegistry) RegisterFolderedType(metadataType, dir string) {
	r.createFuncs[metadataType] = createFolderedMetadataFunc(metadataType, dir)
}

func (r *deployableTypeRegistry) RegisterBundledType(metadataType, dir string) {
	r.createFuncs[metadataType] = createBundledMetadataFunc(metadataType, dir)
}

// Folder types are have their own metadata type, e.g. ReportFolder, but are
// deployed as the type of their contents, e.g. Report
func (r *deployableTypeRegistry) RegisterFolderType(metadataType, deployedType, dir string) {
	r.createFuncs[metadataType] = createFolderedMetadataFunc(deployedType, dir)
}

func createBaseMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &BaseMetadata{
			Path:         path,
			deployedType: deployedType,
			dir:          dir,
		}
	}
}

func createFolderedMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &FolderedMetadata{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: deployedType,
				dir:          dir,
			},
		}
	}
}

func createBundledMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &BundledMetadata{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: deployedType,
				dir:          dir,
			},
		}
	}
}
