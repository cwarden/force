package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ForceCLI/force/config"
	"github.com/ForceCLI/force/lib/metadata"
)

// Structs for XML building
type Package struct {
	Xmlns   string     `xml:"xmlns,attr"`
	Types   []MetaType `xml:"types"`
	Version string     `xml:"version"`
}

type MetadataType string
type MetadataTypeDirectory string

type MetaType struct {
	Members []string     `xml:"members"`
	Name    MetadataType `xml:"name"`
}

func createPackageXml() Package {
	return Package{
		Version: strings.TrimPrefix(apiVersion, "v"),
		Xmlns:   "http://soap.sforce.com/2006/04/metadata",
	}
}

type PackageBuilder struct {
	metadata []metadata.Metadata
}

func (pb PackageBuilder) Size() int {
	return len(pb.metadata)
}

func NewPushBuilder() PackageBuilder {
	pb := PackageBuilder{}
	return pb
}

func NewFetchBuilder() PackageBuilder {
	pb := PackageBuilder{}
	return pb
}

// Build and return package.xml
func (pb PackageBuilder) PackageXml() []byte {
	p := createPackageXml()

	types := make(map[string][]string)

	for _, m := range pb.metadata {
		if members, ok := types[m.DeployedType()]; ok {
			types[m.DeployedType()] = append(members, m.Name())
		} else {
			types[m.DeployedType()] = []string{m.Name()}
		}
	}

	for k, v := range types {
		p.Types = append(p.Types, MetaType{Name: MetadataType(k), Members: v})
	}

	byteXml, _ := xml.MarshalIndent(p, "", "    ")
	byteXml = append([]byte(xml.Header), byteXml...)
	return byteXml
}

func (pb *PackageBuilder) PackageFiles() (ForceMetadataFiles, error) {
	f := make(ForceMetadataFiles)
	if len(pb.metadata) == 0 {
		return f, nil
	}
	f["package.xml"] = pb.PackageXml()
	for _, m := range pb.metadata {
		files, err := m.Files()
		if err != nil {
			return f, err
		}
		for name, content := range files {
			f[name] = content
		}
	}

	return f, nil
}

func (pb *PackageBuilder) AddMetadata(m metadata.Metadata) {
	pb.metadata = append(pb.metadata, m)
}

func (pb *PackageBuilder) AddMetadataType(metadataType string) error {
	metaFolder, err := pb.MetadataDir(metadataType)
	if err != nil {
		return fmt.Errorf("Could not get metadata directry: %w", err)
	}
	fmt.Println("Adding", metaFolder)
	return pb.AddDirectory(metaFolder)
}

func (pb *PackageBuilder) AddMetadataItem(metadataType string, name string) error {
	metaFolder, err := pb.MetadataDir(metadataType)
	if err != nil {
		return fmt.Errorf("Could not get metadata directry: %w", err)
	}
	if filePath, err := findMetadataPath(metaFolder, name); err != nil {
		return fmt.Errorf("Could not find path for %s of type %s: %w", name, metadataType, err)
	} else {
		return pb.Add(filePath)
	}
}

func (pb *PackageBuilder) Add(path string) error {
	f, err := os.Stat(path)
	if err != nil {
		return err
	}
	if f.Mode().IsDir() {
		return pb.AddDirectory(path)
	} else {
		return pb.AddFile(path)
	}
}

func (pb *PackageBuilder) AddFile(fpath string) error {
	if lwcJsTestFile.MatchString(fpath) {
		// If this is a JS test file, just ignore it entirely,
		// don't consider it bad.
		return nil
	}
	m, err := metadata.MetadataFromPath(fpath)
	if err != nil {
		return fmt.Errorf("Could not add file: %w", err)
	}
	pb.AddMetadata(m)
	return nil
}

// AddDirectory Recursively add files contained in provided directory
func (pb *PackageBuilder) AddDirectory(fpath string) error {
	files, err := ioutil.ReadDir(fpath)
	if err != nil {
		return err
	}

	for _, f := range files {
		dirOrFilePath := fpath + "/" + f.Name()
		if strings.HasPrefix(f.Name(), ".") {
			Log.Info("Ignoring hidden file: " + dirOrFilePath)
			continue
		}

		if f.IsDir() {
			if lwcJsTestDir.MatchString(dirOrFilePath) {
				// Normally malformed paths would indicate invalid metadata,
				// but LWC tests should never be deployed. We may want to consider this logic/behavior,
				// such that we don't call `addFile` on directories in some cases; if we could
				// avoid the addFile call on the __tests__ dir, we could avoid this check.
				continue
			}
			fmt.Println("Adding", dirOrFilePath)
			err := pb.AddDirectory(dirOrFilePath)
			if err != nil {
				return err
			}
			continue
		}

		fmt.Println("Adding", dirOrFilePath)
		err = pb.AddFile(dirOrFilePath)
		if err != nil {
			return err
		}

	}
	return err
}

func (pb *PackageBuilder) MetadataDir(metadataType string) (path string, err error) {
	sourceDir, err := config.GetSourceDir()
	if err != nil {
		return "", fmt.Errorf("Could not identify source directory: %w", err)
	}
	md := metadata.Registry.ByName(metadataType)
	if md == nil {
		return "", fmt.Errorf("Unknown metadata type: %s", metadataType)
	}
	return filepath.Join(sourceDir, string(md("").Dir())), nil
}

// Get the path to a metadata file from the source folder and metadata name
func findMetadataPath(folder string, metadataName string) (string, error) {
	info, err := os.Stat(folder)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Invalid directory %s", folder)
	}
	filePath := ""
	err = filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(f.Name())
		if err != nil {
			Log.Info("Error looking for metadata: " + err.Error())
			return nil
		}
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSuffix(rel, ext)) == strings.ToLower(metadataName) {
			filePath = path
		}
		return nil
	})
	if err != nil {
		Log.Info("Error looking for metadata: " + err.Error())
		return "", err
	}
	if filePath == "" {
		return "", fmt.Errorf("Failed to find %s in %s", metadataName, folder)
	}
	return filePath, nil
}

var lwcJsTestFile = regexp.MustCompile(".*\\.test\\.js$")
var lwcJsTestDir = regexp.MustCompile(fmt.Sprintf("%s__tests__$", regexp.QuoteMeta(string(os.PathSeparator))))
