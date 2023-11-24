package metadata

import (
	"fmt"
	"path/filepath"

	fmd "github.com/ForceCLI/force-md/general"
	object "github.com/ForceCLI/force-md/objects"
	"github.com/ForceCLI/force-md/objects/validationrule"
)

type ValidationRule struct {
	CustomObjectComponent
}

func (m *ValidationRule) AddTo(o fmd.Metadata) (fmd.Metadata, error) {
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
	f, err := validationrule.Open(m.Path)
	if err != nil {
		return nil, fmt.Errorf("Could not load field: %w", err)
	}
	obj.ValidationRules = append(obj.ValidationRules, f.Rule)
	return obj, nil
}

func NewValidationRule(path string) Deployable {
	return &ValidationRule{
		CustomObjectComponent: CustomObjectComponent{
			BaseMetadata: BaseMetadata{
				Path:         path,
				deployedType: "ValidationRule",
				dir:          "objects",
			},
		},
	}
}
