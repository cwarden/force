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
	// Creation logic
	return &CustomTab{Path: path}, nil
}
