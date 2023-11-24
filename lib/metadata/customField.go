package metadata

import (
	"fmt"
	"path/filepath"

	fmd "github.com/ForceCLI/force-md/general"
	object "github.com/ForceCLI/force-md/objects"
	field "github.com/ForceCLI/force-md/objects/field"
)

type CustomField struct {
	CustomObjectComponent
}

func (m *CustomField) AddTo(o fmd.Metadata) (fmd.Metadata, error) {
	var obj *object.CustomObject
	var err error
	if o == nil {
		objectPath := filepath.Dir(filepath.Dir(m.Path))
		objectMetaPath := fmt.Sprintf("%s/%s.object-meta.xml", objectPath, m.parentObjectName())
		obj, err = object.Open(objectMetaPath)
		if err != nil {
			return nil, fmt.Errorf("Could not initialize parent object %s: %w", objectMetaPath, err)
		}
	} else if z, ok := o.(*object.CustomObject); ok {
		obj = z
	} else {
		return nil, fmt.Errorf("Expecting a CustomObject")
	}
	f, err := field.Open(m.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not load field: %w", err)
	}
	obj.Fields = append(obj.Fields, f.Field)
	return obj, nil
}

func NewCustomField(path string) Deployable {
	return &CustomField{
		CustomObjectComponent: CustomObjectComponent{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: "CustomField",
				dir:          "objects",
			},
		},
	}
}
