package sf

import (
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// packageVersionSleep waits between two checks of a package version create
// request.  Tests replace it.
var packageVersionSleep = time.Sleep

// statusLine prints "message... status" on stderr.  On a terminal each
// update replaces the line; otherwise only the final status is printed, as
// sf does when its output is not a terminal.
type statusLine struct {
	out      io.Writer
	message  string
	terminal bool
	silent   bool
}

func startStatusLine(cmd *cobra.Command, message string) *statusLine {
	line := &statusLine{out: cmd.ErrOrStderr(), message: message, silent: jsonRequested(cmd)}
	if line.silent {
		return line
	}
	line.terminal = colorEnabled(line.out)
	fmt.Fprintf(line.out, "%s... ", message)
	return line
}

func (l *statusLine) update(status string) {
	if l.silent || !l.terminal {
		return
	}
	fmt.Fprintf(l.out, "\r\x1b[2K%s... %s", l.message, status)
}

func (l *statusLine) stop(status string) {
	if l.silent {
		return
	}
	if l.terminal {
		fmt.Fprintf(l.out, "\r\x1b[2K%s... %s\n", l.message, status)
		return
	}
	fmt.Fprintln(l.out, status)
}

// waitForPackageVersion submits a package version and reports the create
// request the way sf does: at once without a wait, otherwise after the
// request finishes or the wait ends.
func waitForPackageVersion(cmd *cobra.Command, org *targetOrg, projectRoot string, request lib.PackageVersionRequest, wait time.Duration, frequency time.Duration) error {
	line := startStatusLine(cmd, "Version create.")
	requestID, err := org.Force.SubmitPackageVersion(request, nil)
	if err != nil {
		line.stop("")
		return apiError(err)
	}
	load := func() (*orderedMap, error) {
		records, err := toolingRecords(org, fmt.Sprintf("SELECT %s FROM Package2VersionCreateRequest WHERE Id = '%s'", createRequestFields, requestID))
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, newError("PackageVersionCreateRequestNotFound", "No package version create request found for ID %s.", requestID)
		}
		return createRequestResult(org, records[0], true), nil
	}
	result, err := load()
	if err != nil {
		line.stop("")
		return err
	}
	finished := func() bool {
		switch text(result.values["Status"]) {
		case "Success", "Error", "PerformingValidations":
			return true
		}
		return false
	}
	for remaining := wait; wait > 0 && !finished(); remaining -= frequency {
		if remaining <= 0 {
			line.stop("")
			return &Error{
				Name:     "PollingClientTimeout",
				Message:  fmt.Sprintf("The client has timed out. Run '%s package version create report -i %s' to check the status.", binName, requestID),
				ExitCode: 1,
				Data:     orderedFields("VersionCreateRequestId", requestID),
			}
		}
		if status := text(result.values["Status"]); status != "Queued" {
			line.update(fmt.Sprintf("%d minutes remaining until timeout. Create version status: %s", int(math.Round(remaining.Minutes())), status))
		}
		packageVersionSleep(frequency)
		if result, err = load(); err != nil {
			line.stop("")
			return err
		}
	}
	status := text(result.values["Status"])
	line.stop("Create version status: " + status)
	id := text(result.values["Id"])
	subscriberID := text(result.values["SubscriberPackageVersionId"])
	success := func(w io.Writer) {
		fmt.Fprintf(w, "Successfully created the package version [%s]. Subscriber Package Version Id: %s\n", id, subscriberID)
		fmt.Fprintf(w, "Package Installation URL: https://login.salesforce.com/packaging/installPackage.apexp?p0=%s\n", subscriberID)
		fmt.Fprintln(w, "As an alternative, you can use the \""+binName+" package install\" command.")
	}
	switch status {
	case "Error":
		var numbered strings.Builder
		messages, _ := result.values["Error"].([]interface{})
		for i, message := range messages {
			fmt.Fprintf(&numbered, "\n(%d) %s", i+1, text(message))
		}
		return newError("MultipleErrorsError", "Multiple errors occurred: %s", numbered.String())
	case "Success", "PerformingValidations":
		if wait > 0 {
			if err := addPackageVersionAlias(org, projectRoot, result); err != nil {
				return err
			}
		}
		return emit(cmd, result, func(w io.Writer) {
			if status == "PerformingValidations" {
				fmt.Fprintln(w, "The validations for this package version are in progress, but you can now begin testing this package version.")
				fmt.Fprintln(w, "To determine whether all package validations complete successfully, run \"sf package version create report --package-create-request-id 08cxx\" and review the Status.")
			}
			success(w)
		})
	}
	return emit(cmd, result, func(w io.Writer) {
		fmt.Fprintf(w, "Package version creation request status is '%s'. Run \"%s package version create report -i %s\" to query for status.\n", titleCase(status), binName, id)
	})
}

// addPackageVersionAlias records the new version in the packageAliases of
// sfdx-project.json as <package alias>@<major>.<minor>.<patch>-<build>,
// followed by -<branch> for a version on a branch.
func addPackageVersionAlias(org *targetOrg, projectRoot string, result *orderedMap) error {
	subscriberID := text(result.values["SubscriberPackageVersionId"])
	if subscriberID == "" || os.Getenv("SF_PROJECT_AUTOUPDATE_DISABLE_FOR_PACKAGE_VERSION_CREATE") == "true" {
		return nil
	}
	records, err := toolingRecords(org, fmt.Sprintf("SELECT Branch, MajorVersion, MinorVersion, PatchVersion, BuildNumber FROM Package2Version WHERE SubscriberPackageVersionId='%s'", subscriberID))
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	packageRecordID := text(result.values["Package2Id"])
	// sf uses the first alias of the package in the order of the file.
	packageAlias := ""
	if data, err := os.ReadFile(filepath.Join(projectRoot, "sfdx-project.json")); err == nil {
		if parsed, err := parseOrderedJSON(data); err == nil {
			if project, ok := parsed.(*orderedMap); ok {
				if aliases, ok := project.values["packageAliases"].(*orderedMap); ok {
					for _, alias := range aliases.keys {
						if text(aliases.values[alias]) == packageRecordID {
							packageAlias = alias
							break
						}
					}
				}
			}
		}
	}
	if packageAlias == "" {
		return newError("PackageAliasNotFoundError", "The provided package ID: [%s] could not be resolved to an alias.", packageRecordID)
	}
	version := records[0]
	number := func(field string) string {
		if version.values[field] == nil {
			return "0"
		}
		return displayValue(version.values[field])
	}
	name := fmt.Sprintf("%s@%s.%s.%s", packageAlias, number("MajorVersion"), number("MinorVersion"), number("PatchVersion"))
	if build := number("BuildNumber"); build != "0" {
		name += "-" + build
	}
	if branch := text(version.values["Branch"]); branch != "" {
		name += "-" + branch
	}
	return addPackageAlias(projectRoot, name, subscriberID)
}
