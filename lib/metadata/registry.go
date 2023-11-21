package metadata

import "fmt"

type Metadata interface {
	DeployedType() string
	Name() string
	Files() (ForceMetadataFiles, error)
	dir() string
}

type MetadataTypeFunc func(path string) bool
type MetadataCreateFunc func(path string) (Metadata, error)

type metadataTypeRegistry struct {
	detectFuncs map[string]MetadataTypeFunc
	createFuncs map[string]MetadataCreateFunc
}

var Registry = &metadataTypeRegistry{
	detectFuncs: make(map[string]MetadataTypeFunc),
	createFuncs: make(map[string]MetadataCreateFunc),
}

func (r *metadataTypeRegistry) Register(metadataType string, detectFunc MetadataTypeFunc, createFunc MetadataCreateFunc) {
	r.detectFuncs[metadataType] = detectFunc
	r.createFuncs[metadataType] = createFunc
}

func (r *metadataTypeRegistry) CreateMetadata(path string) (Metadata, error) {
	for metadataType, detectFunc := range r.detectFuncs {
		if detectFunc(path) {
			createFunc, exists := r.createFuncs[metadataType]
			if !exists {
				return nil, fmt.Errorf("no create function registered for metadata type: %s", metadataType)
			}
			return createFunc(path)
		}
	}
	return nil, fmt.Errorf("no suitable metadata type found for path: %s", path)
}
