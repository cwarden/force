package sf

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

func newProjectCmd() *cobra.Command {
	return newGroup("project", "Work with projects, such as deploy and retrieve metadata.",
		newGroup("convert", "Convert between source format and metadata format.",
			newProjectConvertSourceCmd(),
			newProjectConvertMdapiCmd(),
		),
		newGroup("generate", "Generate a manifest.", newProjectGenerateManifestCmd()),
		newGroup("list", "List ignored files.", newProjectListIgnoredCmd()),
		newGroup("deploy", "Deploy metadata to an org from your local project.",
			newProjectDeployStartCmd(false),
			newProjectDeployStartCmd(true),
			newProjectDeployQuickCmd(),
			newProjectDeployReportCmd(),
			newProjectDeployResumeCmd(),
			newProjectDeployCancelCmd(),
		),
		newGroup("retrieve", "Retrieve metadata from an org to your local project.",
			newProjectRetrieveStartCmd(),
		),
	)
}

// projectAPIVersion returns the version written to generated manifests:
// sourceApiVersion from sfdx-project.json, or the API version in use.
func projectAPIVersion(project *sfProject) string {
	if project != nil && project.SourceApiVersion != "" {
		return project.SourceApiVersion
	}
	return lib.ApiVersionNumber()
}

// relativePath returns path relative to the working directory, as sf prints
// file paths.
func relativePath(path string) string {
	wd, err := os.Getwd()
	if err != nil {
		return path
	}
	relative, err := filepath.Rel(wd, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

// collectSource gathers the components named by --source-dir, --metadata
// and --manifest.
func collectSource(cmd *cobra.Command, project *sfProject, sourceDirFlag string) (*sourceSet, error) {
	set := newSourceSet()
	if project != nil {
		set.ignore = loadForceIgnore(project.Root)
	}
	sourceDirs := splitList(cmd, sourceDirFlag)
	metadata := splitList(cmd, "metadata")
	manifest, _ := cmd.Flags().GetString("manifest")
	for _, dir := range sourceDirs {
		if err := set.addPath(dir); err != nil {
			return nil, err
		}
	}
	if manifest != "" {
		parsed, err := readManifest(manifest)
		if err != nil {
			return nil, err
		}
		metadata = append(metadata, parsed.metadataNames()...)
	}
	if len(metadata) > 0 {
		if err := set.addMetadata(project, metadata); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func newProjectConvertSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "source",
		Short:       "Convert source-formatted files into metadata that you can deploy using Metadata API.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Source"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "package-name"); err != nil {
				return err
			}
			project, err := loadProject()
			if err != nil {
				return err
			}
			set, err := collectSource(cmd, project, "source-dir")
			if err != nil {
				return err
			}
			if len(set.files) == 0 {
				rootDir, _ := cmd.Flags().GetString("root-dir")
				if rootDir == "" {
					rootDir = project.defaultPackageDir()
				}
				if err := set.addPath(rootDir); err != nil {
					return err
				}
			}
			outputDir, _ := cmd.Flags().GetString("output-dir")
			if outputDir == "" {
				outputDir = fmt.Sprintf("metadataPackage_%d", time.Now().UnixMilli())
			}
			location, err := filepath.Abs(outputDir)
			if err != nil {
				return err
			}
			for path, data := range set.deployFiles(projectAPIVersion(project)) {
				target := filepath.Join(location, filepath.FromSlash(path))
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(target, data, 0o644); err != nil {
					return err
				}
			}
			return emit(cmd, map[string]string{"location": location}, func(w io.Writer) {
				fmt.Fprintf(w, "Source was successfully converted to Metadata API format and written to the location: %s\n", location)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	cmd.Flags().StringP("root-dir", "r", "", "Source directory other than the default package to convert.")
	cmd.Flags().StringP("output-dir", "d", "", "Output directory to store the Metadata API–formatted files in.")
	cmd.Flags().StringP("package-name", "n", "", "Name of the package to associate with the metadata-formatted files.")
	cmd.Flags().StringP("manifest", "x", "", "Path to the manifest (package.xml) file that specifies the metadata types to convert.")
	cmd.Flags().StringArrayP("source-dir", "p", nil, "Paths to the local source files to convert.")
	cmd.Flags().StringArrayP("metadata", "m", nil, "Metadata component names to convert.")
	return cmd
}

type fileResult struct {
	FullName string `json:"fullName"`
	Type     string `json:"type"`
	FilePath string `json:"filePath"`
	State    string `json:"state"`
}

// writeSourceFormat writes metadata format files under root in source
// format and returns one entry for each file written, with paths relative
// to the working directory.  The files are converted in a temporary
// directory first, so that each can be compared with the file it replaces.
func writeSourceFormat(root string, files lib.ForceMetadataFiles, query lib.ForceMetadataQuery) ([]fileResult, error) {
	staging, err := os.MkdirTemp("", "sf-source-format")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(staging)
	write := lib.SourceFormatEntryWriter(staging, query)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := write(name, strings.NewReader(string(files[name]))); err != nil {
			return nil, err
		}
	}
	var results []fileResult
	err = filepath.Walk(staging, func(staged string, info os.FileInfo, err error) error {
		if err != nil || !info.Mode().IsRegular() {
			return err
		}
		relative, err := filepath.Rel(staging, staged)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(staged)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		state := "Add"
		if existing, err := os.ReadFile(target); err == nil {
			state = "Changed"
			if sha256.Sum256(existing) == sha256.Sum256(data) {
				state = "Unchanged"
			}
		}
		if state != "Unchanged" {
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(target, data, 0o644); err != nil {
				return err
			}
		}
		if metadataType, fullName := componentOfPath(relative); metadataType != "" {
			results = append(results, fileResult{FullName: fullName, Type: metadataType, FilePath: relativePath(target), State: state})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		if a.FullName != b.FullName {
			return a.FullName < b.FullName
		}
		// sf lists the -meta.xml file of a component before its content
		// file.
		aMeta, bMeta := strings.HasSuffix(a.FilePath, "-meta.xml"), strings.HasSuffix(b.FilePath, "-meta.xml")
		if aMeta != bMeta {
			return aMeta
		}
		return a.FilePath < b.FilePath
	})
	return results, nil
}

// wildcardQuery requests every component of the types that the source
// format writer handles by name.
var wildcardQuery = lib.ForceMetadataQuery{
	{Name: []string{"CustomObject"}, Members: []string{"*"}},
	{Name: []string{"CustomObjectTranslation"}, Members: []string{"*"}},
}

func newProjectConvertMdapiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "mdapi",
		Short:       "Convert metadata retrieved via Metadata API into the source format used in Salesforce DX projects.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Mdapi"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "manifest", "metadata-dir", "metadata"); err != nil {
				return err
			}
			rootDir, _ := cmd.Flags().GetString("root-dir")
			if rootDir == "" {
				return missingFlagError("root-dir")
			}
			info, err := os.Stat(rootDir)
			if err != nil {
				return newError("InvalidFlagPathError", "Invalid path '%s' for flag 'root-dir': No such file or directory.", rootDir)
			}
			if !info.IsDir() {
				return newError("InvalidFlagPathError", "Invalid path '%s' for flag 'root-dir': Expected a directory but found a file.", rootDir)
			}
			project, err := loadProject()
			if err != nil {
				return err
			}
			outputDir, _ := cmd.Flags().GetString("output-dir")
			if outputDir == "" {
				outputDir = project.defaultPackageDir()
			}
			files, err := readMetadataDir(rootDir)
			if err != nil {
				return err
			}
			query := wildcardQuery
			if manifest, err := readManifest(filepath.Join(rootDir, "package.xml")); err == nil {
				query = manifest.query()
			}
			components := map[string]bool{}
			for path := range files {
				if metadataType, fullName := componentOfPath(path); metadataType != "" {
					components[metadataType+":"+fullName] = true
				}
			}
			stop := startSpinner(cmd, fmt.Sprintf("Converting %d metadata components", len(components)))
			target, err := filepath.Abs(filepath.Join(outputDir, "main", "default"))
			if err != nil {
				return err
			}
			results, err := writeSourceFormat(target, files, query)
			if err != nil {
				stop("failed")
				return err
			}
			stop("done")
			if results == nil {
				results = []fileResult{}
			}
			return emit(cmd, results, func(w io.Writer) {
				if len(results) == 0 {
					fmt.Fprintln(w, "No metadata found to convert")
					return
				}
				rows := make([][]string, 0, len(results))
				for _, result := range results {
					rows = append(rows, []string{result.State, result.FullName, result.Type, result.FilePath})
				}
				printTable(w, "", []string{"STATE", "FULL NAME", "TYPE", "PROJECT PATH"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	cmd.Flags().StringP("root-dir", "r", "", "Root directory that contains the Metadata API–formatted metadata.")
	cmd.Flags().StringP("output-dir", "d", "", "Directory to store your files in after they're converted to source format.")
	cmd.Flags().StringP("manifest", "x", "", "File path to manifest (package.xml) of metadata types to convert.")
	cmd.Flags().StringArrayP("metadata-dir", "p", nil, "Root of directory or zip file of metadata formatted files to convert.")
	cmd.Flags().StringArrayP("metadata", "m", nil, "Metadata component names to convert.")
	return cmd
}

// stage is one line of the progress display of deploy and retrieve.
type stage struct {
	label   string
	state   string // "", "done", "skipped", "failed"
	elapsed time.Duration
	details []string
}

// formatElapsed formats a duration as the progress display does:
// milliseconds below one second, then seconds, then minutes and seconds.
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	minutes := int(d.Minutes())
	return fmt.Sprintf("%dm %.2fs", minutes, d.Seconds()-float64(minutes*60))
}

// wrapLine breaks text at spaces so that no line is wider than width.  The
// space at a break starts the next line, as in sf's output.
func wrapLine(text string, width int) []string {
	var lines []string
	for len([]rune(text)) > width {
		runes := []rune(text)
		cut := width
		for cut > 0 && runes[cut] != ' ' {
			cut--
		}
		if cut == 0 {
			cut = width
		}
		lines = append(lines, string(runes[:cut]))
		text = string(runes[cut:])
	}
	return append(lines, text)
}

// writeStages prints the last frame of the progress display sf shows for
// deploy and retrieve.  sf redraws the frame while it waits; this prints it
// once.
func writeStages(w io.Writer, title, subtitle string, stages []stage, info [][2]string) {
	const width = 50
	side := (width - len([]rune(title)) - 2) / 2
	rule := strings.Repeat("─", side)
	fmt.Fprintf(w, "\n %s %s %s\n\n", rule, title, rule)
	if subtitle != "" {
		for _, line := range wrapLine(subtitle, 79) {
			fmt.Fprintf(w, " %s\n", line)
		}
	}
	fmt.Fprintln(w)
	for _, s := range stages {
		switch s.state {
		case "done":
			fmt.Fprintf(w, " ✔ %s %s\n", s.label, formatElapsed(s.elapsed))
		case "failed":
			fmt.Fprintf(w, " ✘ %s %s\n", s.label, formatElapsed(s.elapsed))
		case "skipped":
			fmt.Fprintf(w, " ◯ %s - Skipped\n", s.label)
		default:
			fmt.Fprintf(w, " ◼ %s\n", s.label)
		}
		for _, detail := range s.details {
			fmt.Fprintf(w, "   ▸ %s\n", detail)
		}
	}
	fmt.Fprintln(w)
	for _, item := range info {
		fmt.Fprintf(w, " %s: %s\n", item[0], item[1])
	}
	fmt.Fprintln(w)
}

func newProjectGenerateManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "manifest",
		Short:       "Create a project manifest that lists the metadata components you want to deploy or retrieve.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ManifestGenerate"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "from-org", "include-packages"); err != nil {
				return err
			}
			metadata := splitList(cmd, "metadata")
			sourceDirs := splitList(cmd, "source-dir")
			name, _ := cmd.Flags().GetString("name")
			manifestType, _ := cmd.Flags().GetString("type")
			if name != "" && manifestType != "" {
				return &Error{Message: fmt.Sprintf("The following errors occurred:\n  --name=%s cannot also be provided when using --type\n  --type=%s cannot also be provided when using --name\nSee more help with --help", name, manifestType), ExitCode: 2}
			}
			if len(metadata) == 0 && len(sourceDirs) == 0 {
				return &Error{Message: "provided flags must include at least one of: metadata,source-dir,from-org", ExitCode: 1}
			}
			fileNames := map[string]string{"": "package.xml", "package": "package.xml", "pre": "destructiveChangesPre.xml", "post": "destructiveChangesPost.xml", "destroy": "destructiveChanges.xml"}
			fileName, ok := fileNames[manifestType]
			if !ok {
				return &Error{Message: fmt.Sprintf("Expected --type=%s to be one of: pre, post, destroy, package\nSee more help with --help", manifestType), ExitCode: 1}
			}
			if name != "" {
				fileName = name
				if !strings.HasSuffix(fileName, ".xml") {
					fileName += ".xml"
				}
			}
			project, err := loadProject()
			if err != nil {
				return err
			}
			// Components named with --metadata are listed as given; the
			// components of --source-dir are read from the files.
			types := map[string][]string{}
			add := func(metadataType, member string) {
				if !containsString(types[metadataType], member) {
					types[metadataType] = append(types[metadataType], member)
				}
			}
			for _, entry := range metadata {
				metadataType, member, hasMember := strings.Cut(entry, ":")
				if !hasMember {
					member = "*"
				}
				add(metadataType, member)
			}
			if len(sourceDirs) > 0 {
				set := newSourceSet()
				set.ignore = loadForceIgnore(project.Root)
				for _, dir := range sourceDirs {
					if err := set.addPath(dir); err != nil {
						return err
					}
				}
				for metadataType, members := range set.types {
					for _, member := range members {
						add(metadataType, member)
					}
				}
			}
			for _, excluded := range splitList(cmd, "excluded-metadata") {
				delete(types, excluded)
			}
			outputDir, _ := cmd.Flags().GetString("output-dir")
			path := fileName
			if outputDir != "" {
				if err := os.MkdirAll(outputDir, 0o755); err != nil {
					return err
				}
				path = filepath.Join(outputDir, fileName)
			}
			if err := os.WriteFile(path, packageXML(types, projectAPIVersion(project)), 0o644); err != nil {
				return err
			}
			result := newOrderedMap()
			if outputDir != "" {
				absolute, _ := filepath.Abs(path)
				result.Set("path", absolute)
			} else {
				result.Set("path", path)
			}
			result.Set("name", fileName)
			return emit(cmd, result, func(w io.Writer) {
				if outputDir != "" {
					fmt.Fprintf(w, "successfully wrote %s to %s\n", fileName, outputDir)
					return
				}
				fmt.Fprintf(w, "successfully wrote %s\n", fileName)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	cmd.Flags().StringArrayP("metadata", "m", nil, "Names of metadata components to include in the manifest.")
	cmd.Flags().StringArrayP("source-dir", "p", nil, "Paths to the local source files to include in the manifest.")
	cmd.Flags().StringP("name", "n", "", "Name of a custom manifest file to create.")
	cmd.Flags().StringP("type", "t", "", "Type of manifest to create; the type determines the name of the created file.")
	cmd.Flags().StringArrayP("include-packages", "c", nil, "Package types (managed, unlocked) whose metadata is included in the manifest; by default, metadata in managed and unlocked packages is excluded.")
	cmd.Flags().StringArray("excluded-metadata", nil, "Metadata types to exclude when building a manifest from an org.")
	cmd.Flags().String("from-org", "", "Username or alias of the org that contains the metadata components from which to build a manifest.")
	cmd.Flags().StringP("output-dir", "d", "", "Directory to save the created manifest.")
	return cmd
}

func newProjectListIgnoredCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "ignored",
		Short:       "Check your local project package directories for forceignored files.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Ignored"},
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := loadProject()
			if err != nil {
				return err
			}
			ignore := loadForceIgnore(project.Root)
			var paths []string
			if sourceDir, _ := cmd.Flags().GetString("source-dir"); sourceDir != "" {
				if _, err := os.Stat(sourceDir); err != nil {
					return newError("invalidSourceDir", "File or directory '%s' doesn't exist in your project. Specify one that exists and rerun the command.", sourceDir)
				}
				paths = []string{sourceDir}
			} else {
				for _, directory := range project.PackageDirectories {
					paths = append(paths, directory.Path)
				}
			}
			ignored := []string{}
			for _, path := range paths {
				target := path
				if !filepath.IsAbs(target) && cmd.Flags().Changed("source-dir") == false {
					target = filepath.Join(project.Root, path)
				}
				files, err := ignore.ignoredFiles(target)
				if err != nil {
					return err
				}
				ignored = append(ignored, files...)
			}
			sort.Strings(ignored)
			result := newOrderedMap()
			result.Set("ignoredFiles", ignored)
			return emit(cmd, result, func(w io.Writer) {
				if len(ignored) == 0 {
					fmt.Fprintf(w, "No ignored files found in paths:\n%s\n", strings.Join(paths, "\n"))
					return
				}
				fmt.Fprintf(w, "Found the following ignored files:\n%s\n", strings.Join(ignored, "\n"))
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("source-dir", "p", "", "File or directory of files that the command checks for foreceignored files.")
	return cmd
}
