package metadata

import (
	"path/filepath"
	"strings"
)

type FolderedMetadata struct {
	BaseMetadata
}

func FolderedComponentName(path, relativeTo string) string {
	path = strings.TrimSuffix(path, "-meta.xml")
	normalizedPath := filepath.ToSlash(path)

	// Find the index of the relativeTo part
	index := strings.Index(normalizedPath, relativeTo)
	if index == -1 {
		return ""
	}

	// Calculate the start position of the relative path
	relativeStart := index + len(relativeTo)
	if relativeStart >= len(normalizedPath) {
		// The relativeTo is at the end of fullpath, no relative path
		return ""
	}

	// Extract and return the relative path
	relativePath := normalizedPath[relativeStart+1:] // +1 to skip the leading '/'

	return strings.TrimSuffix(relativePath, filepath.Ext(path))
}

func (b *FolderedMetadata) Name() string {
	return FolderedComponentName(b.path, b.dir)
}
