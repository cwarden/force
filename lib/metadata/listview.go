package metadata

import (
	"fmt"
	"path/filepath"

	fmd "github.com/ForceCLI/force-md/general"
	object "github.com/ForceCLI/force-md/objects"
	"github.com/ForceCLI/force-md/objects/listview"
)

type ListView struct {
	CustomObjectComponent
}

func (m *ListView) AddTo(o fmd.Metadata) (fmd.Metadata, error) {
	var obj *object.CustomObject
	var err error
	if o == nil {
		objectPath := filepath.Dir(filepath.Dir(m.path))
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
	f, err := listview.Open(m.path)
	if err != nil {
		return nil, fmt.Errorf("Could not load field: %w", err)
	}
	obj.ListViews = append(obj.ListViews, f.ListView)
	return obj, nil
}

func NewListView(path string) Deployable {
	return &ListView{
		CustomObjectComponent: CustomObjectComponent{
			BaseMetadata: BaseMetadata{
				path:         path,
				deployedType: "ListView",
				dir:          "objects",
			},
		},
	}
}
