package sf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// deployPollInterval is the time between checks of a deployment.  Tests
// shorten it.
var deployPollInterval = time.Second

// Fields of a deploy result that sf reports as numbers and as arrays.  The
// SOAP response gives every value as text and gives a single element where
// there is one entry.
var (
	deployNumberFields = map[string]bool{
		"numberComponentErrors": true, "numberComponentsDeployed": true, "numberComponentsTotal": true,
		"numberTestErrors": true, "numberTestsCompleted": true, "numberTestsTotal": true,
		"columnNumber": true, "lineNumber": true, "numFailures": true, "numTestsRun": true,
		"totalTime": true, "time": true, "numLocations": true, "numLocationsNotCovered": true,
	}
	deployArrayFields = map[string]bool{
		"componentSuccesses": true, "componentFailures": true, "successes": true, "failures": true,
		"codeCoverage": true, "codeCoverageWarnings": true, "flowCoverage": true, "flowCoverageWarnings": true,
		"locationsNotCovered": true,
	}
)

// typedDeployValue converts the text values of a SOAP deploy result to the
// JSON types sf reports.
func typedDeployValue(key string, value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		if v == "true" {
			return true
		}
		if v == "false" {
			return false
		}
		if deployNumberFields[key] {
			if number, err := strconv.ParseFloat(v, 64); err == nil {
				return json.Number(strconv.FormatFloat(number, 'f', -1, 64))
			}
		}
		return v
	case *orderedMap:
		for _, childKey := range v.keys {
			child := typedDeployValue(childKey, v.values[childKey])
			if deployArrayFields[childKey] {
				if _, isList := child.([]interface{}); !isList {
					child = []interface{}{child}
				}
			}
			v.values[childKey] = child
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = typedDeployValue(key, item)
		}
		return v
	}
	return value
}

// deployState holds the status of a deployment and the response it came
// from.
type deployState struct {
	status lib.ForceCheckDeploymentStatusResult
	raw    *orderedMap
}

func checkDeploy(org *targetOrg, id string) (*deployState, error) {
	status, body, err := org.Force.Metadata.CheckDeployStatusXML(id)
	if err != nil {
		return nil, apiError(err)
	}
	state := &deployState{status: status, raw: newOrderedMap()}
	if results, err := soapResults(body); err == nil && len(results) > 0 {
		state.raw, _ = typedDeployValue("", results[0]).(*orderedMap)
	}
	if state.raw == nil {
		state.raw = newOrderedMap()
	}
	if details, ok := state.raw.values["details"].(*orderedMap); ok {
		for _, key := range []string{"componentSuccesses", "componentFailures"} {
			if _, present := details.Get(key); !present {
				details.Set(key, []interface{}{})
			}
		}
	}
	return state, nil
}

func deployURL(org *targetOrg, id string) string {
	inner := "/changemgmt/monitorDeploymentsDetails.apexp?asyncId=" + id + "&retURL=" + url.QueryEscape("/changemgmt/monitorDeployment.apexp")
	return org.Fields.String("instanceUrl") + "/lightning/setup/DeployStatus/page?address=" + url.QueryEscape(inner)
}

// deployFileResults lists the local files of the components in a deploy
// result, the way sf reports them in "files".
func deployFileResults(state *deployState, source *sourceSet) []*orderedMap {
	results := []*orderedMap{}
	add := func(fullName, componentType, fileState, fileName string, extra func(*orderedMap)) {
		if fileName == "package.xml" || componentType == "" {
			return
		}
		var paths []string
		if source != nil {
			paths = source.localPaths(fileName)
		}
		if len(paths) == 0 {
			paths = []string{fileName}
		}
		for _, path := range paths {
			item := newOrderedMap()
			item.Set("fullName", fullName)
			item.Set("type", componentType)
			item.Set("state", fileState)
			item.Set("filePath", path)
			if extra != nil {
				extra(item)
			}
			results = append(results, item)
		}
	}
	for _, success := range state.status.Details.ComponentSuccesses {
		fileState := "Unchanged"
		switch {
		case success.Deleted:
			fileState = "Deleted"
		case success.Created:
			fileState = "Created"
		case success.Changed:
			fileState = "Changed"
		}
		add(success.FullName, success.ComponentType, fileState, success.FileName, nil)
	}
	for _, failure := range state.status.Details.ComponentFailures {
		failure := failure
		add(failure.FullName, failure.ComponentType, "Failed", failure.FileName, func(item *orderedMap) {
			item.Set("error", failure.Problem)
			item.Set("problemType", failure.ProblemType)
			if failure.LineNumber > 0 {
				item.Set("lineNumber", failure.LineNumber)
				item.Set("columnNumber", failure.ColumnNumber)
			}
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		a, b := results[i], results[j]
		if a.values["type"] != b.values["type"] {
			return text(a.values["type"]) < text(b.values["type"])
		}
		if a.values["fullName"] != b.values["fullName"] {
			return text(a.values["fullName"]) < text(b.values["fullName"])
		}
		return text(a.values["filePath"]) < text(b.values["filePath"])
	})
	return results
}

// deployJSON returns the result sf reports for a deployment: the fields of
// the Metadata API result and the local files.
func deployJSON(org *targetOrg, state *deployState, source *sourceSet, zipSize, zipFileCount int) *orderedMap {
	result := newOrderedMap()
	for _, key := range state.raw.keys {
		result.Set(key, state.raw.values[key])
	}
	result.Set("files", deployFileResults(state, source))
	if zipSize > 0 {
		result.Set("zipSize", zipSize)
		result.Set("zipFileCount", zipFileCount)
	}
	result.Set("deployUrl", deployURL(org, state.status.Id))
	return result
}

// writeDeployTables prints the tables sf prints after a deployment: the
// deployed or validated source, the component failures and a summary of
// the tests.
func writeDeployTables(w io.Writer, state *deployState, source *sourceSet, concise bool) {
	files := deployFileResults(state, source)
	var successes, failures []*orderedMap
	for _, file := range files {
		if file.values["state"] == "Failed" {
			failures = append(failures, file)
		} else {
			successes = append(successes, file)
		}
	}
	if !concise && len(successes) > 0 && state.status.Status != "Failed" {
		title := "Deployed Source"
		if state.status.CheckOnly {
			title = "Validated Source"
		}
		rows := make([][]string, 0, len(successes))
		for _, file := range successes {
			rows = append(rows, []string{text(file.values["state"]), text(file.values["fullName"]), text(file.values["type"]), relativePath(text(file.values["filePath"]))})
		}
		fmt.Fprintln(w)
		printTable(w, paint(w, style{"\x1b[1m\x1b[34m", "\x1b[39m\x1b[22m"}, title), []string{"State", "Name", "Type", "Path"}, rows)
	}
	if state.status.Status != "Succeeded" && len(state.status.Details.ComponentFailures) > 0 {
		componentFailures := append([]lib.ComponentFailure{}, state.status.Details.ComponentFailures...)
		sort.SliceStable(componentFailures, func(i, j int) bool {
			a, b := componentFailures[i], componentFailures[j]
			if a.ProblemType != b.ProblemType {
				return a.ProblemType < b.ProblemType
			}
			if a.FullName != b.FullName {
				return a.FullName < b.FullName
			}
			return a.LineNumber < b.LineNumber
		})
		rows := make([][]string, 0, len(componentFailures))
		for _, failure := range componentFailures {
			location := ""
			if failure.LineNumber > 0 {
				location = fmt.Sprintf("%d:%d", failure.LineNumber, failure.ColumnNumber)
			}
			rows = append(rows, []string{failure.ComponentType, failure.FullName, failure.Problem, location})
		}
		fmt.Fprintln(w)
		printTable(w, paint(w, styleRed, fmt.Sprintf("Component Failures [%d]", len(rows))), []string{"Type", "Name", "Problem", "Line:Column"}, rows)
	}
	tests := state.status.Details.RunTestResult
	if tests.NumberOfTestsRun > 0 {
		testFailures := tests.TestFailures
		if len(testFailures) > 0 {
			rows := make([][]string, 0, len(testFailures))
			for _, failure := range testFailures {
				rows = append(rows, []string{failure.Name, failure.MethodName, failure.Message, failure.StackTrace})
			}
			fmt.Fprintln(w)
			printTable(w, paint(w, styleRed, fmt.Sprintf("Test Failures [%d]", len(rows))), []string{"Name", "Method", "Message", "Stacktrace"}, rows)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, paint(w, style{"\x1b[1m\x1b[34m", "\x1b[39m\x1b[22m"}, "Test Results Summary"))
		fmt.Fprintf(w, "Passing: %d\n", tests.NumberOfTestsRun-tests.NumberOfFailures)
		fmt.Fprintf(w, "Failing: %d\n", tests.NumberOfFailures)
		fmt.Fprintf(w, "Total: %d\n", tests.NumberOfTestsRun)
		if tests.TotalTime > 0 {
			fmt.Fprintf(w, "Time: %v\n", tests.TotalTime)
		}
	}
}

// deployProgress prints the last frame of the progress display for a
// deployment.
func deployProgress(w io.Writer, org *targetOrg, title, subtitle string, state *deployState, queued bool, prepare, elapsed time.Duration) {
	status := state.status
	stages := []stage{
		{label: "Preparing", state: "done", elapsed: prepare},
		{label: "Waiting for the org to respond", state: "skipped"},
		{label: "Deploying Metadata", state: "done", elapsed: elapsed - prepare},
		{label: "Running Tests", state: "skipped"},
		{label: "Updating Source Tracking", state: "skipped"},
		{label: "Done", state: "done"},
	}
	if queued {
		stages[2].state = "skipped"
	} else {
		if status.NumberComponentsTotal > 0 {
			stages[2].details = []string{fmt.Sprintf("Components: %d/%d (%d%%)", status.NumberComponentsDeployed, status.NumberComponentsTotal, status.NumberComponentsDeployed*100/status.NumberComponentsTotal)}
		}
		if status.NumberComponentErrors > 0 || (status.Status == "Failed" && status.NumberTestErrors == 0) {
			stages[2].state = "failed"
		}
		if status.NumberTestsTotal > 0 {
			stages[3].state = "done"
			stages[3].details = []string{fmt.Sprintf("Successful: %d/%d (%d%%)", status.NumberTestsCompleted, status.NumberTestsTotal, status.NumberTestsCompleted*100/status.NumberTestsTotal)}
			if status.NumberTestErrors > 0 {
				stages[3].state = "failed"
				stages[3].details = append(stages[3].details, fmt.Sprintf("Failed: %d", status.NumberTestErrors))
			}
		}
		if status.Status != "Succeeded" {
			stages[5].state = ""
		}
	}
	statusText := status.Status
	if statusText == "InProgress" {
		statusText = "In Progress"
	}
	info := [][2]string{{"Status", statusText}, {"Deploy ID", status.Id}, {"Target Org", org.Username}, {"Elapsed Time", formatElapsed(elapsed)}}
	writeStages(w, title, subtitle, stages, info)
}

// waitForDeploy polls a deployment until it is done or the timeout passes.
func waitForDeploy(org *targetOrg, id string, timeout time.Duration) (*deployState, error) {
	deadline := time.Now().Add(timeout)
	for {
		state, err := checkDeploy(org, id)
		if err != nil {
			return nil, err
		}
		if state.status.Done {
			return state, nil
		}
		if time.Now().After(deadline) {
			return state, &Error{
				Name:     "ClientTimeoutError",
				Message:  "The command has timed out, although the deployment is still running. Use \"sf project deploy resume\" to resume watching the deployment.",
				ExitCode: 69,
			}
		}
		time.Sleep(deployPollInterval)
	}
}

// finishDeploy prints the outcome of a finished deployment and returns the
// exit status as an error when the deployment did not succeed.
func finishDeploy(cmd *cobra.Command, org *targetOrg, state *deployState, source *sourceSet, zipSize, zipFileCount int, after func(w io.Writer)) error {
	status := 0
	if state.status.Status != "Succeeded" {
		status = 1
	}
	concise, _ := cmd.Flags().GetBool("concise")
	if err := emitStatus(cmd, status, deployJSON(org, state, source, zipSize, zipFileCount), func(w io.Writer) {
		writeDeployTables(w, state, source, concise)
		if status == 0 && after != nil {
			after(w)
		}
	}); err != nil {
		return err
	}
	if status != 0 {
		return exitStatus(status)
	}
	return nil
}

func addDeployJobFlags(cmd *cobra.Command) {
	addJSONFlag(cmd)
	cmd.Flags().StringP("job-id", "i", "", "Job ID of the deploy operation.")
	cmd.Flags().BoolP("use-most-recent", "r", false, "Use the job ID of the most recent deploy operation.")
	cmd.Flags().IntP("wait", "w", 33, "Number of minutes to wait for the command to complete and display results.")
	addTargetOrgFlag(cmd)
}

// deployCachePath returns ~/.sf/deploy-cache.json, where sf records the
// deployments it starts.
func deployCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sf", "deploy-cache.json"), nil
}

func readDeployCache() (map[string]map[string]interface{}, error) {
	entries := map[string]map[string]interface{}{}
	path, err := deployCachePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return entries, nil
}

// recordDeploy adds or updates the cache entry of a deployment.  The other
// entries and the fields sf wrote in an existing entry are kept.
func recordDeploy(id string, fields map[string]interface{}) error {
	entries, err := readDeployCache()
	if err != nil {
		return err
	}
	entry := entries[id]
	if entry == nil {
		entry = map[string]interface{}{}
	}
	for key, value := range fields {
		entry[key] = value
	}
	entry["timestamp"] = time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	entries[id] = entry
	path, err := deployCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// deployJob returns the deployment a command acts on, from --job-id or,
// with --use-most-recent, the newest entry of the deploy cache.  It also
// returns the org the deployment was started in: --target-org when given,
// then the org in the cache entry, then the default org.
func deployJob(cmd *cobra.Command) (string, *targetOrg, error) {
	id, _ := cmd.Flags().GetString("job-id")
	mostRecent, _ := cmd.Flags().GetBool("use-most-recent")
	if (id == "") == !mostRecent {
		return "", nil, exactlyOneError("use-most-recent", "job-id")
	}
	entries, err := readDeployCache()
	if err != nil {
		return "", nil, err
	}
	if mostRecent {
		newest := ""
		for key, entry := range entries {
			stamp, _ := entry["timestamp"].(string)
			if id == "" || stamp > newest {
				id, newest = key, stamp
			}
		}
		if id == "" {
			return "", nil, newError("NoRecentJobIdError", "There are no recent job IDs available.")
		}
	}
	if targetOrgFlag(cmd) == "" {
		if cached, _ := entries[id]["target-org"].(string); cached != "" {
			org, err := loadOrg(cached)
			if err != nil {
				return "", nil, newError("NamedOrgNotFoundError", "No authorization information found for %s.", cached)
			}
			if version := org.Fields.String("instanceApiVersion"); version != "" {
				lib.SetApiVersion(version)
				org.Force.Metadata = lib.NewForceMetadata(org.Force)
			}
			return id, org, nil
		}
	}
	org, err := resolveTargetOrg(cmd)
	return id, org, err
}

func newProjectDeployStartCmd(validate bool) *cobra.Command {
	use, short, className, title := "start", "Deploy metadata to an org from your local project.", "DeployMetadata", "Deploying Metadata"
	if validate {
		use, short, className, title = "validate", "Validate a metadata deployment without actually executing it.", "DeployMetadataValidate", "Validating Deployment"
	}
	cmd := &cobra.Command{
		Use:         use,
		Short:       short,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: className},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "coverage-formatters", "junit", "results-dir"); err != nil {
				return err
			}
			// Destructive change manifests are deployed under the file
			// names the Metadata API reads.
			destructive := lib.ForceMetadataFiles{}
			for flag, name := range map[string]string{"pre-destructive-changes": "destructiveChangesPre.xml", "post-destructive-changes": "destructiveChangesPost.xml"} {
				path, _ := cmd.Flags().GetString(flag)
				if path == "" {
					continue
				}
				if manifest, _ := cmd.Flags().GetString("manifest"); manifest == "" {
					return &Error{Message: fmt.Sprintf("The following error occurred:\n  All of the following must be provided when using --%s: --manifest\nSee more help with --help", flag), ExitCode: 2}
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				destructive[name] = data
			}
			testLevel, _ := cmd.Flags().GetString("test-level")
			tests := splitList(cmd, "tests")
			switch testLevel {
			case "", "NoTestRun", "RunSpecifiedTests", "RunLocalTests", "RunAllTestsInOrg", "RunRelevantTests":
			default:
				return &Error{Message: fmt.Sprintf("Expected --test-level=%s to be one of: NoTestRun, RunSpecifiedTests, RunLocalTests, RunAllTestsInOrg, RunRelevantTests\nSee more help with --help", testLevel), ExitCode: 1}
			}
			if testLevel == "RunSpecifiedTests" && len(tests) == 0 {
				return newError("NoTestsSpecifiedError", "You must specify tests using the --tests flag if the --test-level flag is set to RunSpecifiedTests.")
			}
			if testLevel == "" {
				switch {
				case len(tests) > 0:
					testLevel = "RunSpecifiedTests"
				case validate:
					testLevel = "RunLocalTests"
				}
			}
			metadataDir, _ := cmd.Flags().GetString("metadata-dir")
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			started := time.Now()
			var (
				source  *sourceSet
				files   lib.ForceMetadataFiles
				version = "v" + lib.ApiVersionNumber()
			)
			if metadataDir != "" {
				files, err = readMetadataDir(metadataDir)
				if err != nil {
					return err
				}
				version = "<version specified in manifest>"
			} else {
				project, err := loadProject()
				if err != nil {
					return err
				}
				source, err = collectSource(cmd, project, "source-dir")
				if err != nil {
					return err
				}
				if len(source.files) == 0 {
					return &Error{
						Name:     "NothingToDeploy",
						Message:  "No local changes to deploy.",
						ExitCode: 1,
						Actions:  []string{"To see conflicts and ignored files, run \"sf project deploy preview\" with any of the manifest, directory, or metadata flags."},
					}
				}
				manifestVersion := projectAPIVersion(project)
				version = "v" + manifestVersion
				files = source.deployFiles(manifestVersion)
			}
			for name, data := range destructive {
				files[name] = data
			}
			archive, err := zipFiles(files, "")
			if err != nil {
				return err
			}
			options := lib.ForceDeployOptions{
				CheckOnly:       validate,
				RollbackOnError: true,
				SinglePackage:   true,
				TestLevel:       testLevel,
				RunTests:        tests,
			}
			if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
				options.CheckOnly = true
			}
			if ignoreErrors, _ := cmd.Flags().GetBool("ignore-errors"); ignoreErrors {
				options.RollbackOnError = false
			}
			options.IgnoreWarnings, _ = cmd.Flags().GetBool("ignore-warnings")
			options.PurgeOnDelete, _ = cmd.Flags().GetBool("purge-on-delete")
			subtitle := fmt.Sprintf("Deploying %s metadata to %s using the v%s SOAP API.", version, org.Username, lib.ApiVersionNumber())
			out := cmd.OutOrStdout()
			if validate {
				if !jsonRequested(cmd) {
					fmt.Fprintf(out, "Validating Deployment of %s metadata to %s using the v%s SOAP API.\n", version, org.Username, lib.ApiVersionNumber())
				}
				subtitle = ""
			}
			prepare := time.Since(started)
			id, err := org.Force.Metadata.StartDeploy(files, options)
			if err != nil {
				return apiError(err)
			}
			if err := recordDeploy(id, map[string]interface{}{"target-org": org.Username, "isMdapi": metadataDir != "", "api": "SOAP"}); err != nil {
				return err
			}
			async, _ := cmd.Flags().GetBool("async")
			if async {
				state := &deployState{status: lib.ForceCheckDeploymentStatusResult{Id: id, Status: "Queued"}}
				result := newOrderedMap()
				result.Set("id", id)
				result.Set("done", false)
				result.Set("status", "Queued")
				result.Set("files", []interface{}{})
				result.Set("zipSize", len(archive))
				result.Set("zipFileCount", len(files))
				result.Set("deployUrl", deployURL(org, id))
				return emit(cmd, result, func(w io.Writer) {
					deployProgress(w, org, title, subtitle, state, true, prepare, 0)
					fmt.Fprintln(w, "Deploy has been queued.")
					fmt.Fprintln(w)
					fmt.Fprintf(w, "Run \"sf project deploy resume --job-id %s\" to resume watching the deploy.\n", id)
					fmt.Fprintf(w, "Run \"sf project deploy report --job-id %s\" to get the latest status.\n", id)
					fmt.Fprintf(w, "Run \"sf project deploy cancel --job-id %s\" to cancel the deploy.\n", id)
				})
			}
			minutes, _ := cmd.Flags().GetInt("wait")
			state, err := waitForDeploy(org, id, time.Duration(minutes)*time.Minute)
			if state != nil && !jsonRequested(cmd) {
				deployProgress(out, org, title, subtitle, state, false, prepare, time.Since(started))
			}
			if err != nil {
				return err
			}
			if err := recordDeploy(id, map[string]interface{}{"status": state.status.Status}); err != nil {
				return err
			}
			return finishDeploy(cmd, org, state, source, len(archive), len(files), func(w io.Writer) {
				if validate {
					fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Successfully validated the deployment (%s).", id)))
					fmt.Fprintf(w, "Run \"sf project deploy quick --job-id %s\" to execute this deploy\n", id)
				}
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("api-version", "a", "", "Target API version for the deploy.")
	cmd.Flags().Bool("async", false, "Run the command asynchronously.")
	cmd.Flags().Bool("concise", false, "Show concise output of the deploy result.")
	cmd.Flags().BoolP("ignore-warnings", "g", false, "Ignore warnings and allow a deployment to complete successfully.")
	cmd.Flags().StringP("manifest", "x", "", "Full file path for manifest (package.xml) of components to deploy.")
	cmd.Flags().StringArrayP("metadata", "m", nil, "Metadata component names to deploy.")
	cmd.Flags().String("metadata-dir", "", "Root of directory or zip file of metadata formatted files to deploy.")
	cmd.Flags().Bool("single-package", false, "Indicates that the metadata zip file points to a directory structure for a single package.")
	cmd.Flags().StringArrayP("source-dir", "d", nil, "Path to the local source files to deploy.")
	cmd.Flags().StringArrayP("tests", "t", nil, "Apex tests to run when --test-level is RunSpecifiedTests.")
	cmd.Flags().StringP("test-level", "l", "", "Deployment Apex testing level.")
	cmd.Flags().Bool("verbose", false, "Show verbose output of the deploy result.")
	cmd.Flags().IntP("wait", "w", 33, "Number of minutes to wait for command to complete and display results.")
	cmd.Flags().Bool("purge-on-delete", false, "Specify that deleted components in the destructive changes manifest file are immediately eligible for deletion.")
	cmd.Flags().String("pre-destructive-changes", "", "File path for a manifest of components to delete before the deploy.")
	cmd.Flags().String("post-destructive-changes", "", "File path for a manifest of components to delete after the deploy.")
	cmd.Flags().StringArray("coverage-formatters", nil, "Format of the code coverage results.")
	cmd.Flags().Bool("junit", false, "Output JUnit test results.")
	cmd.Flags().String("results-dir", "", "Output directory for code coverage and JUnit results.")
	if !validate {
		cmd.Flags().Bool("dry-run", false, "Validate deploy and run Apex tests but don’t save to the org.")
		cmd.Flags().BoolP("ignore-conflicts", "c", false, "Ignore conflicts and deploy local files, even if they overwrite changes in the org.")
		cmd.Flags().BoolP("ignore-errors", "r", false, "Ignore any errors and don’t roll back deployment.")
	}
	addTargetOrgFlag(cmd)
	return cmd
}

func newProjectDeployResumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "resume",
		Short:       "Resume watching a deploy operation and update source tracking when the deploy completes.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "DeployMetadataResume"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, org, err := deployJob(cmd)
			if err != nil {
				return err
			}
			started := time.Now()
			state, err := checkDeploy(org, id)
			if err != nil {
				return err
			}
			if state.status.Done {
				warn(cmd, fmt.Sprintf("Job ID %s is not resumable because it already completed with status: %s. Displaying results...", id, state.status.Status))
			} else {
				minutes, _ := cmd.Flags().GetInt("wait")
				state, err = waitForDeploy(org, id, time.Duration(minutes)*time.Minute)
				if state != nil && !jsonRequested(cmd) {
					deployProgress(cmd.OutOrStdout(), org, "Resuming Deploy", "", state, false, 0, time.Since(started))
				}
				if err != nil {
					return err
				}
			}
			return finishDeploy(cmd, org, state, nil, 0, 0, nil)
		},
	}
	addDeployJobFlags(cmd)
	cmd.Flags().Bool("concise", false, "Show concise output of the deploy result.")
	cmd.Flags().Bool("verbose", false, "Show verbose output of the deploy result.")
	return cmd
}

func newProjectDeployReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "report",
		Short:       "Check or poll for the status of a deploy operation.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "DeployMetadataReport"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, org, err := deployJob(cmd)
			if err != nil {
				return err
			}
			var state *deployState
			if cmd.Flags().Changed("wait") {
				minutes, _ := cmd.Flags().GetInt("wait")
				state, err = waitForDeploy(org, id, time.Duration(minutes)*time.Minute)
			} else {
				state, err = checkDeploy(org, id)
			}
			if err != nil {
				return err
			}
			return emit(cmd, deployJSON(org, state, nil, 0, 0), func(w io.Writer) {
				fmt.Fprintf(w, "%s... %s\n", state.status.Id, state.status.Status)
				var rows [][]string
				for _, key := range state.raw.keys {
					switch value := state.raw.values[key].(type) {
					case string, bool, json.Number:
						rows = append(rows, []string{key, displayValue(value)})
					}
				}
				fmt.Fprintln(w)
				printTable(w, paint(w, style{"\x1b[1m\x1b[34m", "\x1b[39m\x1b[22m"}, "Deploy Info"), []string{"Key", "Value"}, rows)
				options := [][]string{{"job-id", id}, {"target-org", org.Username}}
				if cmd.Flags().Changed("wait") {
					minutes, _ := cmd.Flags().GetInt("wait")
					options = append(options, []string{"wait", fmt.Sprintf("%d minutes", minutes)})
				}
				fmt.Fprintln(w)
				printTable(w, paint(w, style{"\x1b[1m\x1b[34m", "\x1b[39m\x1b[22m"}, "Deploy Options"), []string{"Key", "Value"}, options)
				writeDeployTables(w, state, nil, false)
			})
		},
	}
	addDeployJobFlags(cmd)
	return cmd
}

func newProjectDeployQuickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "quick",
		Short:       "Quickly deploy a validated deployment to an org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "DeployMetadataQuick"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, org, err := deployJob(cmd)
			if err != nil {
				return err
			}
			started := time.Now()
			quick, err := org.Force.Metadata.DeployRecentValidation(id)
			if err != nil {
				return newError("CannotQuickDeployError", "Job ID can't be used for quick deployment. Possible reasons include the deployment hasn't been validated, has already been deployed, or the validation expired because you ran it more than 10 days ago.")
			}
			deployID := quick.Id
			if deployID == "" {
				deployID = id
			}
			if async, _ := cmd.Flags().GetBool("async"); async {
				result := newOrderedMap()
				result.Set("id", deployID)
				result.Set("done", false)
				result.Set("status", "Queued")
				result.Set("files", []interface{}{})
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintln(w, "Deploy has been queued.")
					fmt.Fprintln(w)
					fmt.Fprintf(w, "Run \"sf project deploy resume --job-id %s\" to resume watching the deploy.\n", deployID)
					fmt.Fprintf(w, "Run \"sf project deploy report --job-id %s\" to get the latest status.\n", deployID)
					fmt.Fprintf(w, "Run \"sf project deploy cancel --job-id %s\" to cancel the deploy.\n", deployID)
				})
			}
			minutes, _ := cmd.Flags().GetInt("wait")
			state, err := waitForDeploy(org, deployID, time.Duration(minutes)*time.Minute)
			if state != nil && !jsonRequested(cmd) {
				deployProgress(cmd.OutOrStdout(), org, "Deploying Metadata", "", state, false, 0, time.Since(started))
			}
			if err != nil {
				return err
			}
			if state.status.Status != "Succeeded" {
				return newError("QuickDeployFailure", "Deployment %s exited with status code: %s.", deployID, state.status.Status)
			}
			return finishDeploy(cmd, org, state, nil, 0, 0, func(w io.Writer) {
				fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Successfully deployed (%s).", deployID)))
			})
		},
	}
	addDeployJobFlags(cmd)
	cmd.Flags().Bool("async", false, "Run the command asynchronously.")
	cmd.Flags().Bool("concise", false, "Show concise output of the deploy result.")
	cmd.Flags().Bool("verbose", false, "Show verbose output of the deploy result.")
	cmd.Flags().StringP("api-version", "a", "", "Target API version for the deploy.")
	return cmd
}

func newProjectDeployCancelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "cancel",
		Short:       "Cancel a deploy operation.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "DeployMetadataCancel"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, org, err := deployJob(cmd)
			if err != nil {
				return err
			}
			current, err := checkDeploy(org, id)
			if err != nil {
				return err
			}
			if current.status.Done {
				return newError("CannotCancelDeployError", "Can't cancel deploy with Job Id %s because it's already completed (status is %s)", id, current.status.Status)
			}
			if _, err := org.Force.Metadata.CancelDeploy(id); err != nil {
				if err == lib.AlreadyCompletedError {
					return newError("CannotCancelDeployError", "Can't cancel deploy because it's already completed.")
				}
				return apiError(err)
			}
			if async, _ := cmd.Flags().GetBool("async"); async {
				result := newOrderedMap()
				result.Set("id", id)
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintln(w, "Deploy has been queued for cancellation.")
					fmt.Fprintln(w)
					fmt.Fprintf(w, "Run \"sf project deploy resume --job-id %s\" to resume watching the deploy.\n", id)
					fmt.Fprintf(w, "Run \"sf project deploy report --job-id %s\" to get the latest status.\n", id)
				})
			}
			minutes, _ := cmd.Flags().GetInt("wait")
			state, err := waitForDeploy(org, id, time.Duration(minutes)*time.Minute)
			if err != nil {
				return err
			}
			status := 0
			if state.status.Status != "Canceled" {
				status = 1
			}
			if err := emitStatus(cmd, status, deployJSON(org, state, nil, 0, 0), func(w io.Writer) {
				if status == 0 {
					fmt.Fprintf(w, "Successfully canceled %s\n", id)
				} else {
					fmt.Fprintf(w, "Could not cancel %s\n", id)
				}
			}); err != nil {
				return err
			}
			if status != 0 {
				return exitStatus(status)
			}
			return nil
		},
	}
	addDeployJobFlags(cmd)
	cmd.Flags().Bool("async", false, "Run the command asynchronously.")
	return cmd
}

func newProjectRetrieveStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "start",
		Short:       "Retrieve metadata from an org to your local project.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "RetrieveMetadata"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "single-package"); err != nil {
				return err
			}
			metadata := splitList(cmd, "metadata")
			sourceDirs := splitList(cmd, "source-dir")
			packageNames := splitList(cmd, "package-name")
			manifest, _ := cmd.Flags().GetString("manifest")
			targetMetadataDir, _ := cmd.Flags().GetString("target-metadata-dir")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			if len(packageNames) > 1 {
				return newError("UnsupportedFlagError", "Only one --package-name value is supported by the sf-compatible mode of force.")
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			var project *sfProject
			if targetMetadataDir == "" || len(sourceDirs) > 0 {
				project, err = loadProject()
				if err != nil {
					return err
				}
			}
			var query lib.ForceMetadataQuery
			for _, name := range metadata {
				metadataType, member, hasMember := strings.Cut(name, ":")
				if !hasMember {
					member = "*"
				}
				query = append(query, lib.ForceMetadataQueryElement{Name: []string{metadataType}, Members: []string{member}})
			}
			if manifest != "" {
				parsed, err := readManifest(manifest)
				if err != nil {
					return err
				}
				query = append(query, parsed.query()...)
			}
			if len(sourceDirs) > 0 {
				// Retrieve the components that exist in the directories.
				set := newSourceSet()
				for _, dir := range sourceDirs {
					if err := set.addPath(dir); err != nil {
						return err
					}
				}
				names := make([]string, 0, len(set.types))
				for name := range set.types {
					names = append(names, name)
				}
				sort.Strings(names)
				for _, name := range names {
					query = append(query, lib.ForceMetadataQueryElement{Name: []string{name}, Members: set.types[name]})
				}
			}
			if len(query) == 0 && len(packageNames) == 0 {
				return &Error{
					Name:     "NoSourceTrackingError",
					Message:  "Unable to track changes in your source files.\nThis command expects the org to support source tracking. If it doesn't, you must specify the metadata you want to retrieve.",
					ExitCode: 1,
					Actions: []string{
						"Use the `--source-dir`, `--manifest` or `--package-name` flags to retrieve metadata in source format.",
						"Use the `--target-metadata-dir` flag to retrieve metadata in metadata format to a directory.",
					},
				}
			}
			started := time.Now()
			version := lib.ApiVersionNumber()
			subtitle := fmt.Sprintf("Retrieving v%s metadata from %s using the v%s SOAP API", version, org.Username, version)
			stages := []stage{
				{label: "Preparing retrieve request", state: "done", elapsed: time.Since(started)},
				{label: "Sending request to org"},
				{label: "Waiting for the org to respond"},
				{label: "Done"},
			}
			out := cmd.OutOrStdout()
			frame := func(status string) {
				if !jsonRequested(cmd) {
					writeStages(out, "Retrieving Metadata", subtitle, stages, [][2]string{{"Status", status}, {"Elapsed Time", formatElapsed(time.Since(started))}})
				}
			}
			requestStart := time.Now()
			var (
				files    lib.ForceMetadataFiles
				problems []string
			)
			if len(packageNames) == 1 {
				files, problems, err = org.Force.Metadata.RetrievePackage(packageNames[0])
			} else {
				files, problems, err = org.Force.Metadata.Retrieve(query)
			}
			if err != nil {
				stages[1].state = "failed"
				stages[1].elapsed = time.Since(requestStart)
				frame("✘")
				converted := apiError(err)
				if e, ok := converted.(*Error); ok {
					e.Label = "1"
				}
				return converted
			}
			stages[1] = stage{label: "Sending request to org", state: "done", elapsed: time.Since(requestStart)}
			stages[2] = stage{label: "Waiting for the org to respond", state: "done"}
			stages[3] = stage{label: "Done", state: "done"}
			for _, problem := range problems {
				warnJSON(cmd, problem)
			}
			result := newOrderedMap()
			result.Set("done", true)
			result.Set("status", "Succeeded")
			result.Set("success", true)
			if targetMetadataDir != "" {
				zipName, _ := cmd.Flags().GetString("zip-file-name")
				if zipName == "" {
					zipName = "unpackaged.zip"
				}
				if !strings.HasSuffix(zipName, ".zip") {
					zipName += ".zip"
				}
				archive, err := zipFiles(files, "unpackaged/")
				if err != nil {
					return err
				}
				if err := os.MkdirAll(targetMetadataDir, 0o755); err != nil {
					return err
				}
				zipPath := filepath.Join(targetMetadataDir, zipName)
				if err := os.WriteFile(zipPath, archive, 0o644); err != nil {
					return err
				}
				unzip, _ := cmd.Flags().GetBool("unzip")
				extracted := filepath.Join(targetMetadataDir, strings.TrimSuffix(zipName, ".zip"))
				if unzip {
					write := lib.EntryDiskWriter(filepath.Join(extracted, "unpackaged"))
					for name, data := range files {
						if err := write(name, strings.NewReader(string(data))); err != nil {
							return err
						}
					}
				}
				result.Set("zipFilePath", zipPath)
				result.Set("files", []interface{}{})
				frame("Succeeded")
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintf(w, "Wrote retrieve zip file to %s.\n", zipPath)
					if unzip {
						fmt.Fprintf(w, "Extracted %s to %s.\n", strings.TrimSuffix(zipName, ".zip"), extracted)
					}
					for _, problem := range problems {
						warn(cmd, problem)
					}
				})
			}
			if outputDir == "" {
				outputDir = project.defaultPackageDir()
			}
			target, err := filepath.Abs(filepath.Join(outputDir, "main", "default"))
			if err != nil {
				return err
			}
			if len(query) == 0 {
				query = wildcardQuery
			}
			written, err := writeSourceFormat(target, files, query)
			if err != nil {
				return err
			}
			fileItems := make([]*orderedMap, 0, len(written))
			rows := make([][]string, 0, len(written))
			for _, file := range written {
				state := "Created"
				if file.State != "Add" {
					state = file.State
				}
				absolute, _ := filepath.Abs(file.FilePath)
				item := newOrderedMap()
				item.Set("fullName", file.FullName)
				item.Set("type", file.Type)
				item.Set("state", state)
				item.Set("filePath", absolute)
				fileItems = append(fileItems, item)
				rows = append(rows, []string{state, file.FullName, file.Type, file.FilePath})
			}
			result.Set("files", fileItems)
			frame("Succeeded")
			return emit(cmd, result, func(w io.Writer) {
				if len(rows) == 0 {
					warn(cmd, "Nothing retrieved")
				} else {
					fmt.Fprintln(w)
					printTable(w, paint(w, style{"\x1b[1m\x1b[34m", "\x1b[39m\x1b[22m"}, "Retrieved Source"), []string{"State", "Name", "Type", "Path"}, rows)
				}
				for _, problem := range problems {
					warn(cmd, problem)
				}
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("api-version", "a", "", "Target API version for the retrieve.")
	cmd.Flags().BoolP("ignore-conflicts", "c", false, "Ignore conflicts and retrieve and save files to your local filesystem, even if they overwrite your local changes.")
	cmd.Flags().StringP("manifest", "x", "", "File path for the manifest (package.xml) that specifies the components to retrieve.")
	cmd.Flags().StringArrayP("metadata", "m", nil, "Metadata component names to retrieve.")
	cmd.Flags().StringArrayP("package-name", "n", nil, "Installed packages to retrieve.")
	cmd.Flags().StringP("output-dir", "r", "", "Directory root for the retrieved source files.")
	cmd.Flags().Bool("single-package", false, "Indicates that the zip file points to a directory structure for a single package.")
	cmd.Flags().StringArrayP("source-dir", "d", nil, "File paths for source to retrieve from the org.")
	cmd.Flags().StringP("target-metadata-dir", "t", "", "Directory that will contain the retrieved metadata format files or ZIP.")
	cmd.Flags().IntP("wait", "w", 33, "Number of minutes to wait for the command to complete and display results to the terminal window.")
	cmd.Flags().BoolP("unzip", "z", false, "Extract all files from the retrieved zip file.")
	cmd.Flags().String("zip-file-name", "", "File name to use for the retrieved zip file.")
	addTargetOrgFlag(cmd)
	return cmd
}
