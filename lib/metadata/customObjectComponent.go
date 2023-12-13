package metadata

import (
	"fmt"
	"path/filepath"
)

type CustomObjectComponent struct {
	BaseMetadata
}

func (m *CustomObjectComponent) Name() string {
	return fmt.Sprintf("%s.%s", m.parentObjectName(), ComponentName(m.path))
}

func (m *CustomObjectComponent) parentObjectName() string {
	return filepath.Base(filepath.Dir(filepath.Dir(m.path)))
}

func (m *CustomObjectComponent) ParentPath() string {
	return fmt.Sprintf("objects/%s.object", m.parentObjectName())
}
