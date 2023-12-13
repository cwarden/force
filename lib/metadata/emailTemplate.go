package metadata

import "strings"

// Email templates are like FolderedMetadata and ContentMetadata.  They are
// deployed with paths and have separate -meta.xml files.
type EmailTemplateMetadata struct {
	FolderedMetadata
}

func (b *EmailTemplateMetadata) Name() string {
	return FolderedComponentName(b.path, b.dir)
}

func (t *EmailTemplateMetadata) Files() (ForceMetadataFiles, error) {
	return metadataAndContentFiles(t)
}

func (b *EmailTemplateMetadata) Paths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	paths[RelativePath(b.Path(), b.Dir())] = b.Path()
	content := strings.TrimSuffix(b.Path(), "-meta.xml")
	paths[RelativePath(content, b.Dir())] = content
	return paths
}

func NewEmailTemplate(path string) Deployable {
	return &EmailTemplateMetadata{
		FolderedMetadata: FolderedMetadata{
			BaseMetadata: BaseMetadata{
				path:         path,
				deployedType: "EmailTemplate",
				dir:          "email",
			},
		},
	}
}
