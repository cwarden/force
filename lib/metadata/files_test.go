package metadata_test

import (
	"testing"

	. "github.com/ForceCLI/force/lib/metadata"
)

func TestRelativePath(t *testing.T) {
	tests := []struct {
		fullPath   string
		relativeTo string
		want       string
	}{
		{"/path/to/special/file/myfile.cls", "special", "special/file/myfile.cls"},
		{"/path/to/special/dir/myfile.cls", "nonexistent", ""},
		{"/path/to/special/dir/myfile.cls", "dir", "dir/myfile.cls"},
		{"/path/to/special/dir/myfile.cls", "path", "path/to/special/dir/myfile.cls"},
		{"/special", "special", "special"},
	}

	for _, tc := range tests {
		got := MakeRelativePath(tc.fullPath, tc.relativeTo)
		if got != tc.want {
			t.Errorf("RelativePath(%q, %q) = %q, want %q", tc.fullPath, tc.relativeTo, got, tc.want)
		}
	}
}

func TestComponentName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/path/to/special/file/myfile.cls", "myfile"},
		{"/path/to/special/file/myfile.cls-meta.xml", "myfile"},
		{"/path/to/file.txt", "file"},
		{"/path/.hiddenfile", ".hiddenfile"},
		{"/path/to/.hiddenfile.ext", ".hiddenfile"},
		{"/noextensionfile", "noextensionfile"},
	}

	for _, tc := range tests {
		got := ComponentName(tc.path)
		if got != tc.want {
			t.Errorf("ComponentName(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestFolderedComponentName(t *testing.T) {
	tests := []struct {
		path       string
		relativeTo string
		want       string
	}{
		{"/path/to/reports/MyFolder-meta.xml", "reports", "MyFolder"},
		{"/path/to/reports/MyFolder/MyReport.report", "reports", "MyFolder/MyReport"},
		{"/path/to/reports/MyFolder/SubFolder-meta.xml", "reports", "MyFolder/SubFolder"},
		{"/path/to/reports/MyFolder/SubFolder/MyReport.report", "reports", "MyFolder/SubFolder/MyReport"},
	}

	for _, tc := range tests {
		got := FolderedComponentName(tc.path, tc.relativeTo)
		if got != tc.want {
			t.Errorf("FolderedComponentName(%q, %q) = %q, want %q", tc.path, tc.relativeTo, got, tc.want)
		}
	}
}
