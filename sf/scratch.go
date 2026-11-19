package sf

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// scratchDefinitionKeys maps the keys of a scratch org definition file to
// ScratchOrgInfo fields.  "features" and "settings" are handled separately.
var scratchDefinitionKeys = map[string]string{
	"orgName": "OrgName", "edition": "Edition", "country": "Country", "language": "Language",
	"adminEmail": "AdminEmail", "description": "Description", "hasSampleData": "HasSampleData",
	"username": "Username", "release": "Release", "namespace": "Namespace", "duration": "DurationDays",
	"durationDays": "DurationDays", "sourceOrg": "SourceOrg", "snapshot": "Snapshot",
}

// xmlFromJSON writes a JSON value as XML elements: an object becomes child
// elements, an array repeats the element, and other values become text.
func xmlFromJSON(b *bytes.Buffer, name string, value interface{}, indent string) {
	switch v := value.(type) {
	case *orderedMap:
		fmt.Fprintf(b, "%s<%s>\n", indent, name)
		for _, key := range v.keys {
			xmlFromJSON(b, key, v.values[key], indent+"    ")
		}
		fmt.Fprintf(b, "%s</%s>\n", indent, name)
	case []interface{}:
		for _, item := range v {
			xmlFromJSON(b, name, item, indent)
		}
	default:
		fmt.Fprintf(b, "%s<%s>", indent, name)
		xml.EscapeText(b, []byte(displayValue(v)))
		fmt.Fprintf(b, "</%s>\n", name)
	}
}

// scratchSettingsFiles converts the "settings" object of a definition file
// to Settings metadata.  The key lightningExperienceSettings becomes the
// file settings/LightningExperience.settings.
func scratchSettingsFiles(settings *orderedMap) lib.ForceMetadataFiles {
	files := lib.ForceMetadataFiles{}
	var members []string
	for _, key := range settings.keys {
		body, ok := settings.values[key].(*orderedMap)
		if !ok {
			continue
		}
		root := strings.ToUpper(key[:1]) + key[1:]
		member := strings.TrimSuffix(root, "Settings")
		var b bytes.Buffer
		b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
		fmt.Fprintf(&b, "<%s xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n", root)
		for _, field := range body.keys {
			xmlFromJSON(&b, field, body.values[field], "    ")
		}
		fmt.Fprintf(&b, "</%s>\n", root)
		files["settings/"+member+".settings"] = b.Bytes()
		members = append(members, member)
	}
	if len(members) > 0 {
		sort.Strings(members)
		files["package.xml"] = packageXML(map[string][]string{"Settings": members}, lib.ApiVersionNumber())
	}
	return files
}

func newOrgCreateScratchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "scratch",
		Short:       "Create a scratch org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "OrgCreateScratch"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "async", "client-id", "no-ancestors", "track-source", "source-org", "snapshot"); err != nil {
				return err
			}
			definitionFile, _ := cmd.Flags().GetString("definition-file")
			edition, _ := cmd.Flags().GetString("edition")
			if definitionFile == "" && edition == "" {
				return &Error{Message: "The following error occurred:\n  Exactly one of the following must be provided: --definition-file, --edition, --snapshot, --source-org\nSee more help with --help", ExitCode: 2}
			}
			request := newOrderedMap()
			var settings *orderedMap
			if definitionFile != "" {
				data, err := os.ReadFile(definitionFile)
				if err != nil {
					return err
				}
				parsed, err := parseOrderedJSON(data)
				if err != nil {
					return fmt.Errorf("%s: %w", definitionFile, err)
				}
				definition, ok := parsed.(*orderedMap)
				if !ok {
					return fmt.Errorf("%s is not a JSON object", definitionFile)
				}
				for _, key := range definition.keys {
					value := definition.values[key]
					switch key {
					case "features":
						var features []string
						switch v := value.(type) {
						case []interface{}:
							for _, feature := range v {
								features = append(features, displayValue(feature))
							}
						case string:
							features = strings.FieldsFunc(v, func(r rune) bool { return r == ';' || r == ',' })
						}
						request.Set("Features", strings.Join(features, ";"))
					case "settings":
						settings, _ = value.(*orderedMap)
					case "objectSettings":
						return newError("UnsupportedDefinitionKey", "The scratch org definition key \"%s\" is not supported by the sf-compatible mode of force.", key)
					default:
						field, known := scratchDefinitionKeys[key]
						if !known {
							return newError("UnsupportedDefinitionKey", "The scratch org definition key \"%s\" is not supported by the sf-compatible mode of force.", key)
						}
						request.Set(field, value)
					}
				}
			}
			flagValues := map[string]string{"edition": "Edition", "username": "Username", "description": "Description", "name": "OrgName", "release": "Release", "admin-email": "AdminEmail"}
			for _, flag := range []string{"edition", "username", "description", "name", "release", "admin-email"} {
				if value, _ := cmd.Flags().GetString(flag); value != "" {
					request.Set(flagValues[flag], value)
				}
			}
			if release, ok := request.Get("Release"); ok {
				release := strings.ToLower(displayValue(release))
				if release != "preview" && release != "previous" {
					return &Error{Message: fmt.Sprintf("Expected --release=%s to be one of: preview, previous\nSee more help with --help", release), ExitCode: 1}
				}
				request.Set("Release", strings.ToUpper(release[:1])+release[1:])
			}
			if _, ok := request.Get("OrgName"); !ok {
				request.Set("OrgName", "Company")
			}
			if editionValue, ok := request.Get("Edition"); ok {
				// The picklist values are capitalized: Developer, Enterprise.
				name := displayValue(editionValue)
				if name != "" {
					words := strings.Split(name, "-")
					for i, word := range words {
						if word != "" {
							words[i] = strings.ToUpper(word[:1]) + word[1:]
						}
					}
					request.Set("Edition", strings.Join(words, " "))
				}
			}
			days, _ := cmd.Flags().GetInt("duration-days")
			if _, fromFile := request.Get("DurationDays"); !fromFile || cmd.Flags().Changed("duration-days") {
				request.Set("DurationDays", days)
			}
			noNamespace, _ := cmd.Flags().GetBool("no-namespace")
			if noNamespace {
				request.Delete("Namespace")
			} else if _, ok := request.Get("Namespace"); !ok {
				if wd, err := os.Getwd(); err == nil {
					if root, found := lib.SFProjectRoot(wd); found {
						if data, err := os.ReadFile(root + "/sfdx-project.json"); err == nil {
							var project struct {
								Namespace string `json:"namespace"`
							}
							if json.Unmarshal(data, &project) == nil && project.Namespace != "" {
								request.Set("Namespace", project.Namespace)
							}
						}
					}
				}
			}
			request.Set("ConnectedAppConsumerKey", "PlatformCLI")
			request.Set("ConnectedAppCallbackUrl", "http://localhost:1717/OauthRedirect")

			hub, err := resolveTargetDevHub(cmd)
			if err != nil {
				return err
			}
			started := time.Now()
			content, err := encodeJSON(request)
			if err != nil {
				return err
			}
			body, err := hub.Force.PostREST("/sobjects/ScratchOrgInfo", content)
			if err != nil {
				return apiError(err)
			}
			var created struct {
				Id string `json:"id"`
			}
			if err := json.Unmarshal([]byte(body), &created); err != nil || created.Id == "" {
				return fmt.Errorf("the Dev Hub did not return a ScratchOrgInfo Id")
			}
			session, err := hub.Force.ForceLoginNewScratch(created.Id)
			if err != nil {
				return &Error{
					Name:     "ScratchOrgCreateError",
					Message:  err.Error(),
					ExitCode: 1,
					Actions:  []string{fmt.Sprintf("Resume scratch org creation by running \"sf org resume scratch --job-id %s\"", created.Id)},
				}
			}
			infoBody, err := hub.Force.GetREST("/sobjects/ScratchOrgInfo/" + created.Id)
			if err != nil {
				return apiError(err)
			}
			parsedInfo, err := parseOrderedJSON([]byte(infoBody))
			if err != nil {
				return err
			}
			info, _ := parsedInfo.(*orderedMap)
			if info == nil {
				info = newOrderedMap()
			}
			info.Delete("AuthCode")

			fields, order, err := saveAuthorization(cmd, authorization{
				session:  session,
				loginURL: session.InstanceUrl,
				order:    []string{"accessToken", "instanceUrl", "orgId", "username", "loginUrl", "refreshToken", "clientId"},
			})
			if err != nil {
				return err
			}
			username := fields.String("username")
			extra := lib.SFDXAuthFields{
				"devHubUsername": hub.Username,
				"isScratch":      true,
				"isDevHub":       false,
				"created":        fmt.Sprintf("%d", time.Now().UnixMilli()),
				"tracksSource":   true,
			}
			if expiration := text(info.values["ExpirationDate"]); expiration != "" {
				extra["expirationDate"] = expiration
			}
			if instance := text(info.values["SignupInstance"]); instance != "" {
				extra["createdOrgInstance"] = instance
			}
			if err := lib.UpdateSFDXAuthFields(username, extra); err != nil {
				return err
			}
			fields, order, err = lib.ReadSFDXAuthFieldsOrdered(username)
			if err != nil {
				return err
			}

			settingsState := "skipped"
			if settings != nil {
				files := scratchSettingsFiles(settings)
				if len(files) > 0 {
					scratch, err := loadOrg(username)
					if err != nil {
						return err
					}
					if version := fields.String("instanceApiVersion"); version != "" {
						lib.SetApiVersion(version)
						scratch.Force.Metadata = lib.NewForceMetadata(scratch.Force)
					}
					result, err := scratch.Force.Metadata.Deploy(files, lib.ForceDeployOptions{RollbackOnError: true, SinglePackage: true})
					if err != nil {
						return apiError(err)
					}
					if !result.Success {
						var problems []string
						for _, failure := range result.Details.ComponentFailures {
							problems = append(problems, fmt.Sprintf("%s: %s", failure.FullName, failure.Problem))
						}
						return newError("SettingsDeployError", "A scratch org was created with username %s, but the settings failed to deploy due to: \n%s", username, strings.Join(problems, "\n"))
					}
					settingsState = "done"
				}
			}

			authFields := newOrderedMap()
			for _, key := range order {
				if secretFields[key] {
					continue
				}
				value := fields[key]
				if key == "accessToken" {
					value = redactedSecret
				}
				authFields.Set(key, value)
			}
			result := newOrderedMap()
			result.Set("username", username)
			result.Set("scratchOrgInfo", info)
			result.Set("authFields", authFields)
			result.Set("warnings", []interface{}{})
			result.Set("orgId", fields.String("orgId"))
			return emit(cmd, result, func(w io.Writer) {
				elapsed := time.Since(started)
				stages := []stage{
					{label: "Prepare Request", state: "done"},
					{label: "Send Request", state: "done"},
					{label: "Wait For Org", state: "done", elapsed: elapsed},
					{label: "Available", state: "done"},
					{label: "Authenticate", state: "done"},
					{label: "Deploy Settings", state: settingsState},
					{label: "Done", state: "done"},
				}
				infoLines := [][2]string{
					{"Request Id", fmt.Sprintf("%s (%s/%s)", created.Id, strings.TrimRight(hub.Fields.String("instanceUrl"), "/"), created.Id)},
					{"OrgId", text(info.values["ScratchOrg"])},
					{"Username", username},
				}
				if alias, _ := cmd.Flags().GetString("alias"); alias != "" {
					infoLines = append(infoLines, [2]string{"Alias", alias})
				}
				infoLines = append(infoLines, [2]string{"Elapsed Time", formatElapsed(elapsed)})
				writeStages(w, "Creating Scratch Org", "", stages, infoLines)
				fmt.Fprintln(w, paint(w, styleGreen, "Your scratch org is ready."))
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("alias", "a", "", "Alias for the scratch org.")
	cmd.Flags().Bool("async", false, "Request the org, but don't wait for it to complete.")
	cmd.Flags().BoolP("set-default", "d", false, "Set the scratch org as your default org")
	cmd.Flags().StringP("definition-file", "f", "", "Path to a scratch org definition file.")
	cmd.Flags().BoolP("no-ancestors", "c", false, "Don't include second-generation managed package (2GP) ancestors in the scratch org.")
	cmd.Flags().StringP("edition", "e", "", "Salesforce edition of the scratch org. Overrides the value of the \"edition\" option in the definition file, if set.")
	cmd.Flags().BoolP("no-namespace", "m", false, "Create the scratch org with no namespace, even if the Dev Hub has a namespace.")
	cmd.Flags().IntP("duration-days", "y", 7, "Number of days before the org expires.")
	cmd.Flags().IntP("wait", "w", 5, "Number of minutes to wait for the scratch org to be ready.")
	cmd.Flags().StringP("client-id", "i", "", "Consumer key of the Dev Hub connected app.")
	cmd.Flags().BoolP("track-source", "t", true, "Use source tracking for this scratch org.")
	cmd.Flags().String("username", "", "Username of the scratch org admin user.")
	cmd.Flags().String("description", "", "Description of the scratch org in the Dev Hub.")
	cmd.Flags().String("name", "", "Name of the org, such as \"Acme Company\".")
	cmd.Flags().String("release", "", "Release of the scratch org as compared to the Dev Hub release.")
	cmd.Flags().String("admin-email", "", "Email address that will be applied to the org's admin user.")
	cmd.Flags().String("source-org", "", "15-character ID of the org whose shape the new scratch org will be based on.")
	cmd.Flags().StringP("snapshot", "s", "", "Name of the snapshot to use when creating this scratch org.")
	addAPIVersionFlag(cmd)
	addTargetDevHubFlag(cmd)
	return cmd
}
