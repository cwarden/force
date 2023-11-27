package metadata

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ForceCLI/force-md/objecttranslation"
	"github.com/ForceCLI/force-md/objecttranslation/field"
)

// The entire CustomObjectTranslation is deployed if any of the sub-components
// are deployed
type CustomObjectTranslationComponent struct {
	BaseMetadata
}

func (m *CustomObjectTranslationComponent) Name() string {
	if !strings.HasSuffix(m.path(), "-meta.xml") {
		return ComponentName(m.Path)
	}
	return filepath.Base(filepath.Dir(m.Path))
}

func (m *CustomObjectTranslationComponent) UniqueId() string {
	if !strings.HasSuffix(m.path(), "-meta.xml") {
		return m.Path
	}
	return filepath.Dir(m.Path)
}

func (m *CustomObjectTranslationComponent) Files() (ForceMetadataFiles, error) {
	// If it's in metadata format, deploy as-is
	if !strings.HasSuffix(m.path(), "-meta.xml") {
		return metadataOnlyFile(m)
	}

	// Combine the objectTranslation and fieldTranslation metadata
	files := make(ForceMetadataFiles)
	dir := filepath.Dir(m.path())
	objectTranslationMetadata := dir + string(os.PathSeparator) + m.Name() + ".objectTranslation-meta.xml"
	objectTranslation, err := objecttranslation.Open(objectTranslationMetadata)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize parent object tranlation %s: %w", objectTranslationMetadata, err)
	}

	// Any any field translations
	fieldPattern := dir + string(os.PathSeparator) + "*.fieldTranslation-meta.xml"
	fieldFields, err := filepath.Glob(fieldPattern)
	if err != nil {
		return nil, fmt.Errorf("Could not find field translations: %w", err)
	}
	for _, f := range fieldFields {
		fieldTranslation, err := field.Open(f)
		if err != nil {
			return nil, fmt.Errorf("Could not initialize field translations: %w", err)
		}
		objectTranslation.Fields = append(objectTranslation.Fields, fieldTranslation.Field)
	}

	objectTranslationData, err := xml.MarshalIndent(objectTranslation, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("Could not serialize %s: %w", m.Name(), err)
	}
	files[RelativePath(dir+".objectTranslation", m.Dir())] = objectTranslationData
	return files, nil
}

func NewCustomObjectTranslationComponent(path string) Deployable {
	return &CustomObjectTranslationComponent{
		BaseMetadata: BaseMetadata{
			Path:         path,
			deployedType: "CustomObjectTranslation",
			dir:          "objectTranslations",
		},
	}
}
