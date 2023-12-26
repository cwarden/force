package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ForceCLI/config"
)

var Config = config.NewConfig("force")

var sourceDirs = []string{
	"src",
	"metadata",
}

// IsSourceDir returns a boolean indicating that dir is actually a Salesforce
// source directory.
func IsSourceDir(dir string) bool {
	if _, err := os.Stat(dir); err == nil {
		return true
	}
	return false
}

// GetSourceDir returns a rooted path name of the Salesforce source directory,
// relative to the current directory. GetSourceDir will look for a source
// directory in the nearest subdirectory. If no such directory exists, it will
// look at its parents, assuming that it is within a source directory already.
func GetSourceDir() (dir string, err error) {
	base, err := os.Getwd()
	if err != nil {
		return
	}

	// Look down to our nearest subdirectories
	for _, src := range sourceDirs {
		if len(src) > 0 {
			dir = filepath.Join(base, src)
			if IsSourceDir(dir) {
				return
			}
		}
	}

	// Check the current directory and then start looking up at our parents.
	// When dir's parent is identical, it means we're at the root.  If we blow
	// past the actual root, we should drop to the next section of code
	for dir != filepath.Dir(dir) {
		dir = filepath.Dir(dir)
		if isSFDXProject(dir) {
			return getRootFromSFDXProject(dir)
		}
		for _, src := range sourceDirs {
			adir := filepath.Join(dir, src)
			if IsSourceDir(adir) {
				dir = adir
				return
			}
		}
	}

	// No source directory found, create a src directory and a symlinked "metadata"
	// directory for backward compatibility and return that.
	dir = filepath.Join(base, "src")
	err = os.Mkdir(dir, 0777)
	symlink := filepath.Join(base, "metadata")
	os.Symlink(dir, symlink)
	dir = symlink
	return
}

func isSFDXProject(dir string) bool {
	file := filepath.Join(dir, "sfdx-project.json")
	if _, err := os.Stat(file); err == nil {
		return true
	}
	return false
}

func getRootFromSFDXProject(dir string) (string, error) {
	if !isSFDXProject(dir) {
		return "", errors.New("no sfdx-project.json file found")
	}
	path := filepath.Join(dir, "sfdx-project.json")
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("Unable to read file: %w", err)
	}
	var project SFDXProject
	err = json.Unmarshal(file, &project)
	if err != nil {
		return "", fmt.Errorf("Unable to parse file: %w", err)
	}
	for _, p := range project.PackageDirectories {
		if p.Default {
			packageDir := filepath.Join(dir, p.Path)
			defaultDir := filepath.Join(packageDir, "main", "default")
			fmt.Println("will check", packageDir)
			fmt.Println("will check", defaultDir)
			if _, err := os.Stat(defaultDir); err == nil {
				return defaultDir, nil
			} else if _, err := os.Stat(packageDir); err == nil {
				return packageDir, nil
			} else {
				return "", fmt.Errorf("could not find package directory")
			}
		}
	}
	return "", errors.New("no default directory found")
}
