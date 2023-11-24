package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	fmd "github.com/ForceCLI/force-md/general"
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
	metadata map[string]metadata.Deployable
}

func (pb PackageBuilder) Size() int {
	return len(pb.metadata)
}

func NewPushBuilder() PackageBuilder {
	pb := PackageBuilder{}
	pb.metadata = make(map[string]metadata.Deployable)
	return pb
}

func NewFetchBuilder() PackageBuilder {
	pb := PackageBuilder{}
	pb.metadata = make(map[string]metadata.Deployable)
	return pb
}

// Build and return package.xml
func (pb PackageBuilder) PackageXml() []byte {
	p := createPackageXml()

	types := make(map[string][]string)

	for _, d := range pb.metadata {
		if m, ok := d.(metadata.DeployableMetadata); ok {
			if members, ok := types[m.DeployedType()]; ok {
				types[m.DeployedType()] = append(members, m.Name())
			} else {
				types[m.DeployedType()] = []string{m.Name()}
			}
		}
	}

	for k, v := range types {
		p.Types = append(p.Types, MetaType{Name: MetadataType(k), Members: v})
	}

	byteXml, _ := xml.MarshalIndent(p, "", "    ")
	byteXml = append([]byte(xml.Header), byteXml...)
	return byteXml
}

type PendingMerge map[string][]metadata.MergeableMetadata

func merge(toMerge PendingMerge) (ForceMetadataFiles, error) {
	f := make(ForceMetadataFiles)
	for parentPath, m := range toMerge {
		var parent fmd.Metadata
		var err error
		for _, child := range m {
			parent, err = child.AddTo(parent)
			if err != nil {
				return nil, fmt.Errorf("Could not merge %s: %w", child.Name(), err)
			}
		}
		parentData, err := xml.MarshalIndent(parent, "", "    ")
		if err != nil {
			return nil, fmt.Errorf("Could not serialize %s: %w", parentPath, err)
		}

		f[parentPath] = parentData
	}
	return f, nil
}

func (pb *PackageBuilder) PackageFiles() (ForceMetadataFiles, error) {
	f := make(ForceMetadataFiles)
	if len(pb.metadata) == 0 {
		return f, nil
	}
	f["package.xml"] = pb.PackageXml()
	toMerge := make(PendingMerge)
	for _, m := range pb.metadata {
		if merge, ok := m.(metadata.MergeableMetadata); ok {
			// Group metadata, e.g. CustomFields, that needs to be merged by
			// parent path
			parentPath := merge.ParentPath()
			toMerge[parentPath] = append(toMerge[parentPath], merge)
			continue
		} else {
		}
		files, err := m.Files()
		if err != nil {
			return f, err
		}
		for name, content := range files {
			f[name] = content
		}
	}
	merged, err := merge(toMerge)
	if err != nil {
		return f, err
	}
	for name, content := range merged {
		f[name] = content
	}

	return f, nil
}

func (pb *PackageBuilder) AddMetadata(m metadata.Deployable) {
	pb.metadata[m.UniqueId()] = m
}

func (pb *PackageBuilder) AddMetadataType(metadataType string) error {
	metaFolder, err := pb.MetadataDir(metadataType)
	if err != nil {
		return fmt.Errorf("Could not get metadata directry: %w", err)
	}
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
	m, err := metadata.DeployableFromPath(fpath)
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
			err := pb.AddDirectory(dirOrFilePath)
			if err != nil {
				return err
			}
			continue
		}

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
	deployable := md("")
	if m, ok := deployable.(metadata.DeployableMetadata); ok {
		return filepath.Join(sourceDir, m.Dir()), nil
	} else {
		return "", fmt.Errorf("Unknown metadata type: %s", metadataType)
	}
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
