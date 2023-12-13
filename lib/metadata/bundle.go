package metadata

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type BundledMetadata struct {
	BaseMetadata
}

func (t *BundledMetadata) Files() (ForceMetadataFiles, error) {
	return allFilesInFolder(t)
}

func (t *BundledMetadata) FilePaths() ForceMetadataFilePaths {
	paths := make(ForceMetadataFilePaths)
	dir := filepath.Dir(t.Path())
	contents, err := ioutil.ReadDir(dir)
	if err != nil {
		return paths
	}

	for _, f := range contents {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if lwcJsTestFile.MatchString(f.Name()) {
			// If this is a JS test file, just ignore it entirely,
			// don't consider it bad.
			continue
		}

		if f.IsDir() {
			continue
		}

		filePath := dir + string(os.PathSeparator) + f.Name()
		paths[RelativePath(filePath, t.Dir())] = filePath

	}
	return paths
}
