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

func HasRelatedMetadata(path string) bool {
	return !IsMetadata(path) && IsMetadata(path+"-meta.xml")
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
