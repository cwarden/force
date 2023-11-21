package metadata

type Metadata interface {
	DeployedType() string
	Name() string
	Files() (ForceMetadataFiles, error)
	dir() string
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

func (r *metadataTypeRegistry) Register(metadataType string, createFunc MetadataCreateFunc) {
	r.createFuncs[metadataType] = createFunc
}
