package metadata

func init() {
	Registry.Register("CustomTab", isCustomTab, createCustomTab)
}

type CustomTab struct {
	Path string
}

func isCustomTab(path string) bool {
	// Detection logic
	return false
}

func createCustomTab(path string) (Metadata, error) {
	// Get the file contents
	// Get the path relative to the tabs directory
	// Normalize file extension
	return &CustomTab{Path: path}, nil
}
