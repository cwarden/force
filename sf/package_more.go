package sf

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// packageSaveResult is the result sf reports for a change to a package or
// package version record.
func packageSaveResult(id string) *orderedMap {
	result := newOrderedMap()
	result.Set("id", id)
	result.Set("success", true)
	result.Set("errors", []interface{}{})
	return result
}

// patchToolingRecord updates fields of a Tooling API record.
func patchToolingRecord(org *targetOrg, sobject, id string, fields *orderedMap) error {
	content, err := encodeJSON(fields)
	if err != nil {
		return err
	}
	if _, err := org.Force.PatchREST("/tooling/sobjects/"+sobject+"/"+id, content); err != nil {
		return apiError(err)
	}
	return nil
}

func newPackageInstallReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "report",
		Short:       "Retrieve the status of a package installation request.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Report"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("request-id")
			if id == "" {
				return missingFlagError("request-id")
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			record, err := toolingRecord(org, "PackageInstallRequest", id)
			if err != nil {
				return err
			}
			return emitInstallRequest(cmd, org, record)
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("request-id", "i", "", "ID of the package install request you want to check; starts with 0Hf.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

// emitInstallRequest prints the state of a PackageInstallRequest, or
// returns the installation errors as an error.
func emitInstallRequest(cmd *cobra.Command, org *targetOrg, record *orderedMap) error {
	status := strings.ToUpper(text(record.values["Status"]))
	if status == "ERROR" {
		var messages []string
		if errors, ok := record.values["Errors"].(*orderedMap); ok {
			if items, ok := errors.values["errors"].([]interface{}); ok {
				for i, item := range items {
					if detail, ok := item.(*orderedMap); ok {
						messages = append(messages, fmt.Sprintf("(%d) %s", i+1, text(detail.values["message"])))
					}
				}
			}
		}
		return newError("PackageInstallError", "Encountered errors installing the package! Installation errors: \n%s", strings.Join(messages, "\n"))
	}
	return emit(cmd, record, func(w io.Writer) {
		if status == "SUCCESS" {
			fmt.Fprintf(w, "Successfully installed package [%s]\n", text(record.values["SubscriberPackageVersionKey"]))
			return
		}
		fmt.Fprintf(w, "PackageInstallRequest is currently InProgress. You can continue to query the status using\nsf package:install:report -i %s -o %s\n", text(record.values["Id"]), org.Username)
	})
}

func newPackageUninstallReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "report",
		Short:       "Retrieve the status of a package uninstall request.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "PackageUninstallReportCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("request-id")
			if id == "" {
				return missingFlagError("request-id")
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			record, err := toolingRecord(org, "SubscriberPackageVersionUninstallRequest", id)
			if err != nil {
				return err
			}
			return emitUninstallRequest(cmd, org, record)
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("request-id", "i", "", "ID of the package uninstall request you want to check; starts with 06y.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

// emitUninstallRequest prints the state of an uninstall request, or returns
// its errors as an error.
func emitUninstallRequest(cmd *cobra.Command, org *targetOrg, record *orderedMap) error {
	id := text(record.values["Id"])
	status := text(record.values["Status"])
	if strings.EqualFold(status, "Error") {
		messages, _ := toolingRecords(org, fmt.Sprintf("SELECT Message FROM PackageVersionUninstallRequestError WHERE ParentRequest.Id = '%s' ORDER BY Message", id))
		var lines []string
		for i, message := range messages {
			lines = append(lines, fmt.Sprintf("(%d) %s", i+1, text(message.values["Message"])))
		}
		return newError("UNINSTALL_ERROR", "Can't uninstall the package %s during uninstall request %s.\n%s", text(record.values["SubscriberPackageVersionId"]), id, strings.Join(lines, "\n"))
	}
	return emit(cmd, record, func(w io.Writer) {
		if strings.EqualFold(status, "Success") {
			fmt.Fprintf(w, "Successfully uninstalled package [%s]\n", text(record.values["SubscriberPackageVersionId"]))
			return
		}
		fmt.Fprintf(w, "PackageUninstallRequest is currently InProgress.\nYou can continue to query the status using sf package uninstall report -i %s -o %s\n", id, org.Username)
	})
}

// newPackageDeleteCmd builds "package delete" and "package version delete",
// which set IsDeprecated on the record.
func newPackageDeleteCmd(version bool) *cobra.Command {
	short, className, sobject := "Delete a package.", "PackageDeleteCommand", "Package2"
	prompt, denied := "Deleted packages can't be recovered.\n\nDo you want to continue? (y/n)", "The request to delete this package was canceled"
	undeletePrompt := "This will undelete the package, which may result in unintended consequences for customers. Proceed with caution.\n\nDo you want to continue? (y/n)"
	if version {
		short, className, sobject = "Delete a package version.", "PackageVersionDeleteCommand", "Package2Version"
		prompt, denied = "Deleted package versions can’t be recovered.\n\nDo you want to continue? (y/n)", "The request to delete this package version has been canceled."
		undeletePrompt = "This will undelete the package version, which may result in unintended consequences for customers. Proceed with caution.\n\nDo you want to continue? (y/n)"
	}
	cmd := &cobra.Command{
		Use:         "delete",
		Short:       short,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: className},
		RunE: func(cmd *cobra.Command, args []string) error {
			packageFlag, _ := cmd.Flags().GetString("package")
			if packageFlag == "" {
				return missingFlagError("package")
			}
			undelete, _ := cmd.Flags().GetBool("undelete")
			org, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			id := packageID(packageFlag)
			if version {
				record, err := findPackageVersion(org, packageFlag)
				if err != nil {
					return err
				}
				id = text(record.values["Id"])
			}
			question := prompt
			if undelete {
				question = undeletePrompt
			}
			if !confirm(cmd, question) {
				return newError("PromptDeleteDenyError", "%s", denied)
			}
			fields := newOrderedMap()
			fields.Set("IsDeprecated", !undelete)
			if err := patchToolingRecord(org, sobject, id, fields); err != nil {
				return err
			}
			return emit(cmd, packageSaveResult(id), func(w io.Writer) {
				switch {
				case undelete && version:
					fmt.Fprintf(w, "Successfully undeleted package version %s.\n", id)
				case undelete:
					fmt.Fprintln(w, "Successfully undeleted the package.")
				case version:
					fmt.Fprintf(w, "Successfully deleted the package version with ID: %s\n", id)
				default:
					fmt.Fprintf(w, "Successfully deleted the package with ID: %s\n", id)
				}
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().BoolP("no-prompt", "n", false, "Don’t prompt before deleting.")
	cmd.Flags().StringP("package", "p", "", "ID or alias of the package or package version to delete.")
	cmd.Flags().Bool("undelete", false, "Undelete a deleted package or package version.")
	cmd.Flags().MarkHidden("undelete")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}

// setIfChanged adds a field to a record update when its flag was given.
func setIfChanged(cmd *cobra.Command, fields *orderedMap, flag, field string) {
	if cmd.Flags().Changed(flag) {
		value, _ := cmd.Flags().GetString(flag)
		fields.Set(field, value)
	}
}

func newPackageUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "update",
		Short:       "Update package details.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "PackageUpdateCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "skip-ancestor-check"); err != nil {
				return err
			}
			packageFlag, _ := cmd.Flags().GetString("package")
			if packageFlag == "" {
				return missingFlagError("package")
			}
			org, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			id := packageID(packageFlag)
			fields := newOrderedMap()
			setIfChanged(cmd, fields, "name", "Name")
			setIfChanged(cmd, fields, "description", "Description")
			setIfChanged(cmd, fields, "error-notification-username", "PackageErrorUsername")
			setIfChanged(cmd, fields, "recommended-version-id", "RecommendedVersionId")
			if cmd.Flags().Changed("enable-app-analytics") {
				enabled, _ := cmd.Flags().GetBool("enable-app-analytics")
				fields.Set("AppAnalyticsEnabled", enabled)
			}
			if err := patchToolingRecord(org, "Package2", id, fields); err != nil {
				return err
			}
			return emit(cmd, packageSaveResult(id), func(w io.Writer) {
				fmt.Fprintf(w, "Successfully updated the package. %s\n", id)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("package", "p", "", "ID (starts with 0Ho) or alias of the package to update.")
	cmd.Flags().StringP("name", "n", "", "New name of the package.")
	cmd.Flags().StringP("description", "d", "", "New description of the package.")
	cmd.Flags().StringP("error-notification-username", "o", "", "Active Dev Hub user designated to receive email notifications for package errors.")
	cmd.Flags().Bool("enable-app-analytics", false, "Enable AppExchange App Analytics usage data collection on this managed package and its components.")
	cmd.Flags().StringP("recommended-version-id", "r", "", "Package version ID to recommend to subscribers.")
	cmd.Flags().Bool("skip-ancestor-check", false, "Bypass the ancestry check for setting a recommended version.")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}

func newPackageVersionUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "update",
		Short:       "Update a package version.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "PackageVersionUpdateCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			packageFlag, _ := cmd.Flags().GetString("package")
			if packageFlag == "" {
				return missingFlagError("package")
			}
			org, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			record, err := findPackageVersion(org, packageFlag)
			if err != nil {
				return err
			}
			id := text(record.values["Id"])
			fields := newOrderedMap()
			setIfChanged(cmd, fields, "version-name", "Name")
			setIfChanged(cmd, fields, "version-description", "Description")
			setIfChanged(cmd, fields, "branch", "Branch")
			setIfChanged(cmd, fields, "tag", "Tag")
			setIfChanged(cmd, fields, "installation-key", "InstallKey")
			if err := patchToolingRecord(org, "Package2Version", id, fields); err != nil {
				return err
			}
			return emit(cmd, packageSaveResult(id), func(w io.Writer) {
				fmt.Fprintf(w, "Successfully updated the package version. %s\n", id)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("package", "p", "", "ID (starts with 04t) or alias of the package version to update.")
	cmd.Flags().StringP("version-name", "a", "", "New package version name.")
	cmd.Flags().StringP("version-description", "e", "", "New package version description.")
	cmd.Flags().StringP("branch", "b", "", "New package version branch.")
	cmd.Flags().StringP("tag", "t", "", "New package version tag.")
	cmd.Flags().StringP("installation-key", "k", "", "New installation key for key-protected package (default: null)")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}

const createRequestFields = "Id, Status, Package2Id, Package2.Name, Package2VersionId, Package2Version.SubscriberPackageVersionId, Package2Version.HasMetadataRemoved, Tag, Branch, CreatedDate, CreatedById"

// createRequestResult converts a Package2VersionCreateRequest record to the
// result sf reports.
func createRequestResult(org *targetOrg, record *orderedMap) *orderedMap {
	var subscriberID, hasMetadataRemoved interface{}
	if version, ok := record.values["Package2Version"].(*orderedMap); ok {
		subscriberID = version.values["SubscriberPackageVersionId"]
		hasMetadataRemoved = version.values["HasMetadataRemoved"]
	}
	var packageName interface{}
	if parent, ok := record.values["Package2"].(*orderedMap); ok {
		packageName = parent.values["Name"]
	}
	id := text(record.values["Id"])
	errors := []interface{}{}
	if text(record.values["Status"]) == "Error" {
		messages, _ := toolingRecords(org, fmt.Sprintf("SELECT Message FROM Package2VersionCreateRequestError WHERE ParentRequest.Id = '%s'", id))
		for _, message := range messages {
			errors = append(errors, text(message.values["Message"]))
		}
	}
	created := isoTime(text(record.values["CreatedDate"]))
	if len(created) >= 16 {
		created = strings.Replace(created[:16], "T", " ", 1)
	}
	result := newOrderedMap()
	result.Set("Id", id)
	result.Set("Status", record.values["Status"])
	result.Set("Package2Id", record.values["Package2Id"])
	result.Set("Package2Name", packageName)
	result.Set("Package2VersionId", record.values["Package2VersionId"])
	result.Set("SubscriberPackageVersionId", subscriberID)
	result.Set("Tag", record.values["Tag"])
	result.Set("Branch", record.values["Branch"])
	result.Set("Error", errors)
	result.Set("CreatedDate", created)
	result.Set("HasMetadataRemoved", hasMetadataRemoved)
	result.Set("CreatedBy", record.values["CreatedById"])
	return result
}

// titleCase puts a space before each inner capital: InProgress becomes
// "In Progress".
func titleCase(value string) string {
	var b strings.Builder
	for i, char := range value {
		if i > 0 && char >= 'A' && char <= 'Z' {
			b.WriteByte(' ')
		}
		b.WriteRune(char)
	}
	return b.String()
}

func newPackageVersionCreateReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "report",
		Short:       "Retrieve details about a package version creation request.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "PackageVersionCreateReportCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("package-create-request-id")
			if id == "" {
				return missingFlagError("package-create-request-id")
			}
			org, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			records, err := toolingRecords(org, fmt.Sprintf("SELECT %s FROM Package2VersionCreateRequest WHERE Id = '%s'", createRequestFields, strings.ReplaceAll(id, "'", `\'`)))
			if err != nil {
				return err
			}
			if len(records) == 0 {
				return newError("PackageVersionCreateRequestNotFound", "No package version create request found for ID %s.", id)
			}
			result := createRequestResult(org, records[0])
			return emit(cmd, []*orderedMap{result}, func(w io.Writer) {
				installURL := ""
				if text(result.values["Status"]) == "Success" {
					installURL = "https://login.salesforce.com/packaging/installPackage.apexp?p0=" + resultCell(result, "SubscriberPackageVersionId")
				}
				rows := [][]string{
					{"ID", resultCell(result, "Id")},
					{"Status", titleCase(resultCell(result, "Status"))},
					{"Package Id", resultCell(result, "Package2Id")},
					{"Package Version Id", resultCell(result, "Package2VersionId")},
					{"Subscriber Package Version Id", resultCell(result, "SubscriberPackageVersionId")},
					{"Tag", resultCell(result, "Tag")},
					{"Branch", resultCell(result, "Branch")},
					{"Created Date", resultCell(result, "CreatedDate")},
					{"Installation URL", installURL},
					{"Created By", resultCell(result, "CreatedBy")},
				}
				printTable(w, paint(w, style{"\x1b[34m", "\x1b[39m"}, "Package Version Create Request"), []string{"Name", "Value"}, rows)
				if errors, _ := result.values["Error"].([]interface{}); len(errors) > 0 {
					fmt.Fprintln(w)
					styledHeader(w, paint(w, styleRed, "Errors"))
					lines := make([]string, 0, len(errors))
					for i, message := range errors {
						lines = append(lines, fmt.Sprintf("(%d) %s", i+1, displayValue(message)))
					}
					warn(cmd, strings.Join(lines, "\n"))
				}
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("package-create-request-id", "i", "", "ID (starts with 08c) of the package version creation request you want to display.")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}

func newPackageVersionCreateListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "List package version creation requests.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "PackageVersionCreateListCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "show-conversions-only", "verbose"); err != nil {
				return err
			}
			org, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			var conditions []string
			if cmd.Flags().Changed("created-last-days") {
				days, _ := cmd.Flags().GetInt("created-last-days")
				conditions = append(conditions, fmt.Sprintf("CreatedDate = LAST_N_DAYS:%d", days))
			}
			if status, _ := cmd.Flags().GetString("status"); status != "" {
				switch status {
				case "Queued", "InProgress", "Success", "Error":
				default:
					return &Error{Message: fmt.Sprintf("Expected --status=%s to be one of: Queued, InProgress, Success, Error\nSee more help with --help", status), ExitCode: 2}
				}
				conditions = append(conditions, fmt.Sprintf("Status = '%s'", status))
			}
			soql := "SELECT " + createRequestFields + " FROM Package2VersionCreateRequest"
			if len(conditions) > 0 {
				soql += " WHERE " + strings.Join(conditions, " AND ")
			}
			records, err := toolingRecords(org, soql+" ORDER BY CreatedDate")
			if err != nil {
				return err
			}
			results := make([]*orderedMap, 0, len(records))
			rows := make([][]string, 0, len(records))
			for _, record := range records {
				result := createRequestResult(org, record)
				results = append(results, result)
				rows = append(rows, []string{
					resultCell(result, "Id"), resultCell(result, "Status"), resultCell(result, "Package2Id"), resultCell(result, "Package2VersionId"),
					resultCell(result, "SubscriberPackageVersionId"), resultCell(result, "Tag"), resultCell(result, "Branch"),
					resultCell(result, "CreatedDate"), resultCell(result, "CreatedBy"),
				})
			}
			if len(rows) == 0 {
				warn(cmd, "No results found")
				return emit(cmd, results, nil)
			}
			return emit(cmd, results, func(w io.Writer) {
				printTable(w, paint(w, style{"\x1b[34m", "\x1b[39m"}, fmt.Sprintf("Package Version Create Requests  [%d]", len(rows))),
					[]string{"Id", "Status", "Package Id", "Package Version Id", "Subscriber Package Version Id", "Tag", "Branch", "Created Date", "Created By"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().IntP("created-last-days", "c", 0, "Number of days since the request was created, starting at 00:00:00 of first day to now.")
	cmd.Flags().StringP("status", "s", "", "Status of the version creation request, used to filter the list.")
	cmd.Flags().Bool("show-conversions-only", false, "Filter the list output to display only converted package version.")
	cmd.Flags().Bool("verbose", false, "Displays additional information at a slight performance cost, such as the version name and number for each package version create request.")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}
