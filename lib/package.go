package lib

import "github.com/ForceCLI/force/lib/metadata"

type Format int

const (
	MetadataFormat Format = iota
	SourceFormat
)

func (f Format) String() string {
	return []string{"Metadata Format", "Source Format"}[f]
}

type PackageAble interface {
	Package() Package
}

type DeployablePackage struct {
	Package
	Files ForceMetadataFiles
}

type RetrievedPackage struct {
	Package
	Files  ForceMetadataFiles
	Root   string
	Format Format
}

func deployablePackageFromMetadataFile(path string) (DeployablePackage, error) {
	pkg := Package{}

	return DeployablePackage{
		Package: pkg,
	}, nil
}

func deployablePackageFromPath(path string) (DeployablePackage, error) {
	switch {
	/*
		case isDirWithRelatedMetadata(path):
			// Folder name
		case isDir(path):
			// Metadata directory
	*/
	case metadata.IsMetadata(path):
		// Standalone metadata, e.g. CustomTab, CustomApplication, etc.
	case hasRelatedMetadata(path):
		// Has separate metadata file, e.g. ApexClass
	}
	return DeployablePackage{}, nil
}
