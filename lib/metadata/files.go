package metadata

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ComponentName(path string) string {
	name := strings.TrimSuffix(path, "-meta.xml")
	if filepath.Base(name) == filepath.Ext(name) {
		return filepath.Base(name)
	}
	return strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
}

func MakeRelativePath(fullpath, relativeTo string) string {
	// Normalize the path to use forward slashes
	normalizedPath := filepath.ToSlash(fullpath)

	// Find the index of the relativeTo part
	idx := strings.Index(normalizedPath, relativeTo)
	if idx == -1 {
		return ""
	}

	// Slice the string from the found index
	relativePath := normalizedPath[idx:]

	return relativePath
}

func metadataOnlyFile(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	fileContent, err := ioutil.ReadFile(m.Path())
	if err != nil {
		return nil, fmt.Errorf("Could not read metadata: %w", err)
	}
	files[RelativePath(m.Path(), m.Dir())] = fileContent
	return files, nil
}

func metadataAndContentFiles(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	for relative, fullPath := range m.Paths() {
		fileContent, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("Could not read file %s: %w", fullPath, err)
		}
		files[relative] = fileContent
	}
	return files, nil
}

func allFilesInFolder(m DeployableMetadata) (ForceMetadataFiles, error) {
	files := make(ForceMetadataFiles)
	dir := filepath.Dir(m.Path())
	contents, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
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
		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("Could not read metadata: %w", err)
		}
		files[MakeRelativePath(filePath, m.Dir())] = fileContent

	}
	return files, nil
}

var lwcJsTestFile = regexp.MustCompile(".*\\.test\\.js$")
