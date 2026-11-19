package sf

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

func newOrgCmd() *cobra.Command {
	return newGroup("org", "Commands to create and manage orgs and scratch org users.",
		withSubcommands(newOrgListCmd(),
			newOrgListAuthCmd(),
			newOrgListLimitsCmd(),
			newOrgListMetadataCmd(),
			newOrgListMetadataTypesCmd(),
			newGroup("sobject", "List sObject information.", newOrgListSobjectRecordCountsCmd()),
		),
		newOrgOpenCmd(),
		withSubcommands(newOrgDisplayCmd(),
			newOrgDisplayUserCmd(),
		),
		newGroup("auth", "Show stored credentials of an org.",
			newOrgAuthShowCmd("show-access-token", "Display the access token of an org.", "OrgAuthShowAccessToken"),
			newOrgAuthShowCmd("show-sfdx-auth-url", "Display the SFDX auth URL of an org.", "OrgAuthShowSfdxAuthUrl"),
			newOrgAuthShowCmd("show-user-password", "Display the locally stored password of an org user.", "OrgAuthShowUserPassword"),
		),
		newGroup("assign", "Assign permission sets and licenses.", newOrgAssignPermsetCmd(), newOrgAssignPermsetLicenseCmd()),
		newGroup("create", "Create scratch orgs.", newOrgCreateScratchCmd()),
		newGroup("delete", "Delete scratch orgs.", newOrgDeleteScratchCmd()),
		newGroup("disable", "Disable source tracking.", newOrgTrackingCmd(false)),
		newGroup("enable", "Enable source tracking.", newOrgTrackingCmd(true)),
		newGroup("generate", "Generate a password.", newOrgGeneratePasswordCmd()),
		newOrgLoginCmd(),
		newOrgLogoutCmd(),
	)
}

// orgEntry is one authorized org as listed by org list and org list auth.
type orgEntry struct {
	org        *targetOrg
	username   string
	alias      string
	isDevHub   bool
	isSandbox  bool
	isScratch  bool
	isDefault  bool
	isDefaultH bool
	status     string
}

func (e *orgEntry) orgType() string {
	switch {
	case e.isDevHub:
		return "DevHub"
	case e.isSandbox:
		return "Sandbox"
	case e.isScratch:
		return "Scratch"
	}
	return ""
}

// loadOrgEntries reads every authorization file.  Files that cannot be read
// are skipped.
func loadOrgEntries() []*orgEntry {
	defaultOrg, defaultDevHub := defaultUsernames()
	var entries []*orgEntry
	for _, username := range lib.ListSFDXUsernames() {
		org, err := loadOrg(username)
		if err != nil || org.Fields.String("accessToken") == "" || org.Fields.String("instanceUrl") == "" {
			// alias.json, key.json and config files are in the same
			// directory as the authorization files.
			continue
		}
		entries = append(entries, &orgEntry{
			org:        org,
			username:   org.Username,
			alias:      org.Alias,
			isDevHub:   org.Fields.Bool("isDevHub"),
			isSandbox:  org.Fields.Bool("isSandbox"),
			isScratch:  org.Fields.Bool("isScratch") || org.Fields.String("devHubUsername") != "",
			isDefault:  org.Username == defaultOrg,
			isDefaultH: org.Username == defaultDevHub,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if (a.alias == "") != (b.alias == "") {
			return a.alias != ""
		}
		if a.alias != b.alias {
			return a.alias < b.alias
		}
		return a.username < b.username
	})
	return entries
}

func checkConnections(entries []*orgEntry) {
	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		go func(entry *orgEntry) {
			defer wg.Done()
			entry.status = connectedStatus(entry.org)
		}(entry)
	}
	wg.Wait()
}

// secretFields are the authorization file fields left out of org list
// output.
var secretFields = map[string]bool{"refreshToken": true, "clientSecret": true, "privateKey": true, "password": true}

func (e *orgEntry) listJSON(withStatus bool) *orderedMap {
	result := newOrderedMap()
	for _, key := range e.org.Order {
		if secretFields[key] {
			continue
		}
		value := e.org.Fields[key]
		if key == "accessToken" {
			value = redactedSecret
		}
		result.Set(key, value)
	}
	if e.alias != "" {
		result.Set("alias", e.alias)
	}
	result.Set("isDefaultDevHubUsername", e.isDefaultH)
	result.Set("isDefaultUsername", e.isDefault)
	if modified, err := lib.SFDXAuthModTime(e.username); err == nil {
		result.Set("lastUsed", modified.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	if withStatus {
		result.Set("connectedStatus", e.status)
	}
	return result
}

type orgListResult struct {
	Other          []*orderedMap `json:"other"`
	Sandboxes      []*orderedMap `json:"sandboxes"`
	NonScratchOrgs []*orderedMap `json:"nonScratchOrgs"`
	DevHubs        []*orderedMap `json:"devHubs"`
	ScratchOrgs    []*orderedMap `json:"scratchOrgs"`
}

func newOrgListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all orgs you've created or authenticated to.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			skipStatus, _ := cmd.Flags().GetBool("skip-connection-status")
			all, _ := cmd.Flags().GetBool("all")
			entries := loadOrgEntries()
			if !skipStatus {
				checkConnections(entries)
			}
			result := orgListResult{
				Other:          []*orderedMap{},
				Sandboxes:      []*orderedMap{},
				NonScratchOrgs: []*orderedMap{},
				DevHubs:        []*orderedMap{},
				ScratchOrgs:    []*orderedMap{},
			}
			// Rows are grouped as sf prints them: Dev Hubs, other
			// orgs, sandboxes, scratch orgs.
			groups := [4][]*orgEntry{}
			for _, entry := range entries {
				item := entry.listJSON(!skipStatus)
				switch {
				case entry.isScratch:
					result.ScratchOrgs = append(result.ScratchOrgs, item)
					groups[3] = append(groups[3], entry)
					continue
				case entry.isDevHub:
					result.DevHubs = append(result.DevHubs, item)
					groups[0] = append(groups[0], entry)
				case entry.isSandbox:
					result.Sandboxes = append(result.Sandboxes, item)
					groups[2] = append(groups[2], entry)
				default:
					result.Other = append(result.Other, item)
					groups[1] = append(groups[1], entry)
				}
				result.NonScratchOrgs = append(result.NonScratchOrgs, item)
			}
			warnJSON(cmd, secretsHiddenWarning("org list"))
			return emit(cmd, result, func(w io.Writer) {
				var ordered []*orgEntry
				for _, group := range groups {
					ordered = append(ordered, group...)
				}
				printOrgList(w, ordered, !skipStatus, all)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().Bool("verbose", false, "List more information about each org.")
	cmd.Flags().Bool("all", false, "Include expired, deleted, and unknown-status scratch orgs.")
	cmd.Flags().Bool("clean", false, "Remove all local org authorizations for non-active scratch orgs.")
	cmd.Flags().BoolP("no-prompt", "p", false, "Don't prompt for confirmation.")
	cmd.Flags().Bool("skip-connection-status", false, "Skip retrieving the connection status of non-scratch orgs.")
	return cmd
}

func printOrgList(w io.Writer, entries []*orgEntry, withStatus bool, all bool) {
	if len(entries) == 0 {
		fmt.Fprintln(w, "No Orgs found.")
	} else {
		hasType := false
		for _, entry := range entries {
			if entry.orgType() != "" {
				hasType = true
			}
		}
		headers := []string{""}
		if hasType {
			headers = append(headers, "Type")
		}
		headers = append(headers, "Alias", "Username", "Org Id")
		if withStatus {
			headers = append(headers, "Status")
		}
		rows := make([][]string, 0, len(entries))
		for _, entry := range entries {
			marker := ""
			switch {
			case entry.isDefaultH && entry.isDefault:
				marker = "🌳,🍁"
			case entry.isDefaultH:
				marker = "🌳"
			case entry.isDefault:
				marker = "🍁"
			}
			row := []string{marker}
			if hasType {
				row = append(row, entry.orgType())
			}
			row = append(row, entry.alias, entry.username, entry.org.Fields.String("orgId"))
			if withStatus {
				row = append(row, entry.status)
			}
			rows = append(rows, row)
		}
		printStyledTable(w, "", headers, rows, func(row int) *style {
			switch entries[row].orgType() {
			case "DevHub":
				return &styleBrightCyan
			case "Sandbox":
				return &styleBrightYellow
			case "Scratch":
				return &styleGreen
			}
			return &styleBrightMagenta
		})
	}
	fmt.Fprintln(w, paint(w, styleDim, ""))
	legend := "Legend:  🌳=Default DevHub, 🍁=Default Org"
	if !all {
		legend += "      Use --all to see expired and deleted scratch orgs"
	}
	fmt.Fprintln(w, paint(w, styleDim, legend))
}

func oauthMethod(fields lib.SFDXAuthFields) string {
	switch {
	case fields.String("privateKey") != "":
		return "jwt"
	case fields.String("refreshToken") != "":
		return "web"
	}
	return "token"
}

func newOrgListAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "List authorization information about the orgs you created or logged into.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries := loadOrgEntries()
			sort.SliceStable(entries, func(i, j int) bool { return entries[i].username < entries[j].username })
			result := make([]*orderedMap, 0, len(entries))
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				fields := entry.org.Fields
				configs := []string{}
				if entry.isDefault {
					configs = append(configs, lib.SFConfigTargetOrg)
				}
				if entry.isDefaultH {
					configs = append(configs, lib.SFConfigTargetDevHub)
				}
				item := newOrderedMap()
				item.Set("configs", configs)
				item.Set("username", entry.username)
				item.Set("instanceUrl", fields.String("instanceUrl"))
				item.Set("isScratchOrg", entry.isScratch)
				item.Set("isDevHub", entry.isDevHub)
				item.Set("isSandbox", entry.isSandbox)
				item.Set("orgId", fields.String("orgId"))
				item.Set("accessToken", redactedSecret)
				item.Set("oauthMethod", oauthMethod(fields))
				if expiration := fields.String("expirationDate"); expiration != "" {
					expires, err := time.Parse("2006-01-02", expiration)
					item.Set("isExpired", err == nil && time.Now().After(expires.Add(24*time.Hour)))
				} else {
					item.Set("isExpired", "unknown")
				}
				if entry.alias != "" {
					item.Set("alias", strings.Join(aliasesFor(entry.username), ","))
				}
				result = append(result, item)
				rows = append(rows, []string{
					strings.Join(aliasesFor(entry.username), ","),
					entry.username,
					fields.String("orgId"),
					fields.String("instanceUrl"),
					oauthMethod(fields),
				})
			}
			if len(result) > 0 {
				warnJSON(cmd, secretsHiddenWarning("org list auth"))
			}
			return emit(cmd, result, func(w io.Writer) {
				if len(rows) == 0 {
					fmt.Fprintln(w, "No results found.")
					return
				}
				printTable(w, "authenticated orgs", []string{"Alias", "Username", "Org Id", "Instance Url", "Auth Method"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	return cmd
}

func newOrgDisplayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "display",
		Short: "Display information about an org.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			fields := org.Fields
			status := connectedStatus(org)
			result := newOrderedMap()
			result.Set("id", fields.String("orgId"))
			result.Set("apiVersion", fields.String("instanceApiVersion"))
			result.Set("accessToken", redactedSecret)
			result.Set("instanceUrl", fields.String("instanceUrl"))
			result.Set("username", org.Username)
			rows := [][]string{
				{"Access Token", redactedSecret},
			}
			if org.Alias != "" {
				rows = append(rows, []string{"Alias", org.Alias})
			}
			rows = append(rows, []string{"Api Version", fields.String("instanceApiVersion")})
			if clientID := fields.String("clientId"); clientID != "" {
				result.Set("clientId", clientID)
				rows = append(rows, []string{"Client Id", clientID})
			}
			result.Set("connectedStatus", status)
			rows = append(rows, []string{"Connected Status", status})
			if devHub := fields.String("devHubUsername"); devHub != "" {
				result.Set("devHubUsername", devHub)
				rows = append(rows, []string{"Dev Hub Username", devHub})
			}
			if expiration := fields.String("expirationDate"); expiration != "" {
				result.Set("expirationDate", expiration)
				rows = append(rows, []string{"Expiration Date", expiration})
			}
			rows = append(rows,
				[]string{"Id", fields.String("orgId")},
				[]string{"Instance Url", fields.String("instanceUrl")},
				[]string{"Username", org.Username},
			)
			if org.Alias != "" {
				result.Set("alias", org.Alias)
			}
			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose && fields.String("refreshToken") != "" {
				redacted := "[REDACTED] Use 'sf org auth show-sfdx-auth-url' to view"
				result.Set("sfdxAuthUrl", redacted)
				rows = append(rows[:len(rows)-1], []string{"Sfdx Auth Url", redacted}, rows[len(rows)-1])
			}
			if status != "Connected" && jsonRequested(cmd) {
				warnJSON(cmd, "unable to refresh auth for org")
			}
			warn(cmd, secretsHiddenWarning("org display"))
			return emit(cmd, result, func(w io.Writer) {
				fmt.Fprintln(w)
				printTable(w, "Org Description", []string{"KEY", "VALUE"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().Bool("verbose", false, "Display the sfdxAuthUrl property.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newOrgDisplayUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Display information about a Salesforce user.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			query := fmt.Sprintf("SELECT Id, Profile.Name FROM User WHERE Username = '%s'", strings.ReplaceAll(org.Username, "'", "\\'"))
			records, err := org.Force.Query(query)
			if err != nil {
				return err
			}
			if len(records.Records) == 0 {
				return newError("UserNotFoundError", "No user found with username %s.", org.Username)
			}
			record := records.Records[0]
			id, _ := record["Id"].(string)
			profileName := ""
			if profile, ok := record["Profile"].(map[string]interface{}); ok {
				profileName, _ = profile["Name"].(string)
			}
			fields := org.Fields
			result := newOrderedMap()
			result.Set("accessToken", redactedSecret)
			result.Set("id", id)
			result.Set("instanceUrl", fields.String("instanceUrl"))
			result.Set("loginUrl", fields.String("loginUrl"))
			result.Set("orgId", fields.String("orgId"))
			result.Set("profileName", profileName)
			result.Set("username", org.Username)
			rows := [][]string{
				{"Username", org.Username},
				{"Profile Name", profileName},
				{"Id", id},
				{"Org Id", fields.String("orgId")},
				{"Access Token", redactedSecret},
				{"Instance Url", fields.String("instanceUrl")},
				{"Login Url", fields.String("loginUrl")},
			}
			if org.Alias != "" {
				result.Set("alias", org.Alias)
				rows = append(rows, []string{"Alias", org.Alias})
			}
			warn(cmd, secretsHiddenWarning("org display user"))
			return emit(cmd, result, func(w io.Writer) {
				fmt.Fprintln(w)
				printTable(w, "User Description", []string{"Key", "Label"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

// confirm asks a yes or no question on the terminal.  The answer is yes
// without asking when --no-prompt or --json is given.
func confirm(cmd *cobra.Command, question string) bool {
	if noPrompt, _ := cmd.Flags().GetBool("no-prompt"); noPrompt || jsonRequested(cmd) {
		return true
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "? %s (y/N) ", question)
	answer, _ := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

// logout removes the authorization file for username, the aliases that
// refer to it, and the target-org and target-dev-hub config values that
// refer to it or to one of its aliases.
func logout(username string) error {
	names := map[string]bool{username: true}
	for _, alias := range aliasesFor(username) {
		names[alias] = true
		if _, _, err := lib.UnsetSFDXAlias(alias); err != nil {
			return err
		}
	}
	for _, global := range []bool{true, false} {
		values, _, err := lib.SFConfigFileValues(global)
		if err != nil {
			continue
		}
		for _, key := range []string{lib.SFConfigTargetOrg, lib.SFConfigTargetDevHub} {
			if value, ok := values[key]; ok && names[value] {
				if _, err := lib.UnsetSFConfig(key, global); err != nil {
					return err
				}
			}
		}
	}
	return lib.DeleteSFDXAuth(username)
}

func newOrgLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Log out of a Salesforce org.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			var usernames []string
			if all {
				for _, entry := range loadOrgEntries() {
					usernames = append(usernames, entry.username)
				}
				sort.Strings(usernames)
			} else {
				name := targetOrgFlag(cmd)
				if name == "" {
					if value, found, _ := lib.GetSFConfig(lib.SFConfigTargetOrg); found {
						name = value.Value
					}
				}
				if name == "" {
					return newError("NoOrgSpecifiedWithNoPromptError", "You must specify a target-org (or default target-org config is set) or use --all flag when using the --no-prompt flag.")
				}
				org, err := loadOrg(name)
				if err != nil {
					return newError("NoAuthFoundForTargetOrgError", "No authenticated org found with the %s username or alias.", name)
				}
				usernames = []string{org.Username}
			}
			if len(usernames) == 0 {
				return emit(cmd, []string{}, func(w io.Writer) {
					fmt.Fprintln(w, "No orgs found to log out of.")
				})
			}
			if !confirm(cmd, fmt.Sprintf("Are you sure you want to log out of %s?", strings.Join(usernames, ", "))) {
				return emit(cmd, []string{}, func(w io.Writer) {})
			}
			for _, username := range usernames {
				if err := logout(username); err != nil {
					return err
				}
			}
			return emit(cmd, usernames, func(w io.Writer) {
				fmt.Fprintf(w, "Successfully logged out of orgs: %s\n", strings.Join(usernames, ", "))
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().BoolP("all", "a", false, "Include all authenticated orgs.")
	cmd.Flags().BoolP("no-prompt", "p", false, "Don't prompt for confirmation.")
	cmd.Flags().StringP("client-app", "c", "", "Client app to log out of.")
	addTargetOrgFlag(cmd)
	return cmd
}

// sfdxAuthURL returns the SFDX auth URL of an authorization:
// force://<client id>:<client secret>:<refresh token>@<instance host>.
func sfdxAuthURL(fields lib.SFDXAuthFields) string {
	instance := strings.TrimRight(fields.String("instanceUrl"), "/")
	instance = strings.TrimPrefix(strings.TrimPrefix(instance, "https://"), "http://")
	clientID := fields.String("clientId")
	if clientID == "" {
		clientID = "PlatformCLI"
	}
	return fmt.Sprintf("force://%s:%s:%s@%s", clientID, fields.String("clientSecret"), fields.String("refreshToken"), instance)
}

// newOrgAuthShowCmd builds the three "org auth show-*" commands, which
// print one stored credential after a confirmation.
func newOrgAuthShowCmd(use, short, className string) *cobra.Command {
	type credential struct {
		label, key, prompt, warning, missingName, missing string
		value                                             func(lib.SFDXAuthFields) string
	}
	credentials := map[string]credential{
		"show-access-token": {
			label: "Access Token", key: "accessToken",
			prompt:      `You're about to reveal the access token for "%s". This token grants full access to the org with your current permissions. Sharing or logging this token is equivalent to sharing your credentials. Do you want to continue?`,
			warning:     "This command exposes a sensitive Access Token that allows for subsequent activity using your current authenticated session. Sharing this information is equivalent to logging someone in under the current credential, resulting in unintended access and escalation of privilege. For additional information about org authorization, review https://developer.salesforce.com/docs/atlas.en-us.sfdx_dev.meta/sfdx_dev/sfdx_dev_auth.htm.",
			missingName: "NoAccessTokenError", missing: `No access token found for "%s". The org may need to be re-authenticated.`,
			value: func(fields lib.SFDXAuthFields) string { return fields.String("accessToken") },
		},
		"show-sfdx-auth-url": {
			label: "SFDX Auth URL", key: "sfdxAuthUrl",
			prompt:      `You're about to reveal the SFDX Auth URL for "%s". This URL contains a refresh token that can be used to authenticate to the org without user interaction. Do you want to continue?`,
			warning:     "This command exposes an SFDX Auth URL. Unlike an access token, this credential contains a refresh token that allows extended access to an org. Avoid sharing or logging this URL. For additional information about org authorization, review https://developer.salesforce.com/docs/atlas.en-us.sfdx_dev.meta/sfdx_dev/sfdx_dev_auth_url.htm.",
			missingName: "NoRefreshTokenError", missing: `An SFDX Auth URL is not available for "%s". This URL is only available for orgs authenticated via a web-based login flow. Re-authenticate to the org using "sf org login web" to make it available.`,
			value: func(fields lib.SFDXAuthFields) string {
				if fields.String("refreshToken") == "" {
					return ""
				}
				return sfdxAuthURL(fields)
			},
		},
		"show-user-password": {
			label: "Password", key: "password",
			prompt:      `You're about to reveal the password for "%s". Do you want to continue?`,
			warning:     "This command exposes a user password. While a password alone isn't sufficient to gain access to an org (additional factors like a security token or an enabled OAuth username-password flow are required), treat it as a sensitive credential and avoid sharing or logging it.",
			missingName: "NoPasswordError", missing: `No password found for "%s". A password is available only if one was previously generated locally in your DX project by running either "sf org generate password" or "sf org create user". If you generated a password in your org using Setup, it's not available to this command.`,
			value: func(fields lib.SFDXAuthFields) string { return fields.String("password") },
		},
	}
	shown := credentials[use]
	cmd := &cobra.Command{
		Use:         use,
		Short:       short,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: className},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			if noPrompt, _ := cmd.Flags().GetBool("no-prompt"); noPrompt || jsonRequested(cmd) {
				warn(cmd, shown.warning)
			} else if !confirm(cmd, fmt.Sprintf(shown.prompt, org.Username)) {
				return &Error{Message: "Show " + strings.ToLower(shown.label) + " confirmation denied or timed out.", ExitCode: 1}
			}
			value := shown.value(org.Fields)
			if value == "" {
				return newError(shown.missingName, shown.missing, org.Username)
			}
			result := newOrderedMap()
			result.Set(shown.key, value)
			return emit(cmd, result, func(w io.Writer) {
				printTable(w, "", []string{"Key", "Value"}, [][]string{{shown.label, value}})
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().BoolP("no-prompt", "p", false, "Don't prompt for confirmation.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}
