package metadata

import (
	"fmt"
	"path/filepath"

	fmd "github.com/ForceCLI/force-md/general"
	object "github.com/ForceCLI/force-md/objects"
	"github.com/ForceCLI/force-md/objects/recordtype"
)

type RecordType struct {
	CustomObjectComponent
}

func (m *RecordType) AddTo(o fmd.Metadata) (fmd.Metadata, error) {
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
	f, err := recordtype.Open(m.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not load field: %w", err)
	}
	obj.RecordTypes = append(obj.RecordTypes, f.RecordType)
	return obj, nil
}

func NewRecordType(path string) Deployable {
	return &RecordType{
		CustomObjectComponent: CustomObjectComponent{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: "RecordType",
				dir:          "objects",
			},
		},
	}
}
