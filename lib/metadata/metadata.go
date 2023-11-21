package metadata

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

var NotXMLError = errors.New("Could not parse as XML")
var MetadataFileNotFound = errors.New("Could not identify metadata type")

type FilePath = string
type ForceMetadataFiles map[FilePath][]byte

func HasRelatedMetadata(path string) bool {
	return !IsMetadata(path) && IsMetadata(path+"-meta.xml")
}

type MetadataType string

// If the file in path contains metadata, return it.  Otherwise, try to find
// the corresponding file that contains metadata.
func metadataFileFromPath(path string) (string, error) {
	if IsMetadata(path) {
		return path, nil
	}
	if IsMetadata(path + "-meta.xml") {
		return path + "-meta.xml", nil
	}
	return "", fmt.Errorf("%w: %s", MetadataFileNotFound, path)
}

func MetadataFromPath(path string) (Metadata, error) {
	path, err := metadataFileFromPath(path)
	if err != nil {
		return nil, err
	}
	_, err = os.Stat(path)
	if err != nil {
		return nil, err
	}
	element, err := getRootElementName(path)
	if err == NotXMLError {
		return nil, err
	}
	if f, ok := Registry.createFuncs[element]; ok {
		return f(path), nil
	}
	return nil, fmt.Errorf("Could not find metadata")
}

func IsMetadata(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	element, err := getRootElementName(path)
	if err == NotXMLError {
		return false
	}
	if _, ok := Registry.createFuncs[element]; ok {
		return ok
	}
	return false
}

func getRootElementName(file string) (string, error) {
	xmlData, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("Could read XML file: %w", err)
	}

	decoder := xml.NewDecoder(ioutil.NopCloser(bytes.NewReader(xmlData)))

	for {
		t, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", NotXMLError
		}

		switch element := t.(type) {
		case xml.StartElement:
			// Return the name of the root element
			return element.Name.Local, nil
		}
	}
	return "", NotXMLError
}
