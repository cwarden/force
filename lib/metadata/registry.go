package metadata

type DeployableMetadata interface {
	Deployable
	DeployedType() string
	Name() string
	Dir() string
	Path() string
}

type FetchableMetadata interface {
	// Map relative paths used by Metadata API to filesystem paths
	Paths() ForceMetadataFilePaths
}

// destructiveChanges.xml is deployable, but is not metadata
type Deployable interface {
	Files() (ForceMetadataFiles, error)
	Paths() ForceMetadataFilePaths
	UniqueId() string
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
	if _, ok := r.createFuncs[metadataType]; ok {
		panic("Duplicate metadata registration for " + metadataType)
	}
	r.createFuncs[metadataType] = createFunc
}

func (r *deployableTypeRegistry) RegisterBaseType(metadataType, dir string) {
	r.Register(metadataType, createBaseMetadataFunc(metadataType, dir))
}

func (r *deployableTypeRegistry) RegisterFolderedType(metadataType, dir string) {
	r.Register(metadataType, createFolderedMetadataFunc(metadataType, dir))
}

func (r *deployableTypeRegistry) RegisterContentType(metadataType, dir string) {
	r.Register(metadataType, createContentMetadataFunc(metadataType, dir))
}

func (r *deployableTypeRegistry) RegisterBundledType(metadataType, dir string) {
	r.Register(metadataType, createBundledMetadataFunc(metadataType, dir))
}

// Folder types are have their own metadata type, e.g. ReportFolder, but are
// deployed as the type of their contents, e.g. Report
func (r *deployableTypeRegistry) RegisterFolderType(metadataType, deployedType, dir string) {
	r.createFuncs[metadataType] = createFolderedMetadataFunc(deployedType, dir)
}

func createBaseMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &BaseMetadata{
			path:         path,
			deployedType: deployedType,
			dir:          dir,
		}
	}
}

func createFolderedMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &FolderedMetadata{
			BaseMetadata: BaseMetadata{
				path:         path,
				deployedType: deployedType,
				dir:          dir,
			},
		}
	}
}

func createContentMetadataFunc(deployedType, dir string) DeployableCreateFunc {
	return func(path string) Deployable {
		return &ContentMetadata{
			BaseMetadata: BaseMetadata{
				path:         path,
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
				path:         path,
				deployedType: deployedType,
				dir:          dir,
			},
		}
	}
}
