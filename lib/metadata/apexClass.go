package metadata

func init() {
	Registry.Register("ApexClass", isApexClass, createApexClass)
}

type ApexClass struct {
	Path string
}

func isApexClass(path string) bool {
	// Detection logic
	return false
}

func createApexClass(path string) (Metadata, error) {
	// Creation logic
	return &ApexClass{Path: path}, nil
}
