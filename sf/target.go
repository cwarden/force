package sf

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib"
)

// targetOrg is an org authorization read from the sf state directory, with
// a session for making API requests.
type targetOrg struct {
	Username string
	Alias    string
	Fields   lib.SFDXAuthFields
	Order    []string
	Force    *lib.Force
}

// aliasFor returns the first alias in alias.json that refers to username.
func aliasFor(username string) string {
	aliases, _ := lib.ListSFDXAliasesOrdered()
	for _, alias := range aliases {
		if alias.Value == username {
			return alias.Alias
		}
	}
	return ""
}

// aliasesFor returns every alias that refers to username.
func aliasesFor(username string) []string {
	aliases, _ := lib.ListSFDXAliasesOrdered()
	var names []string
	for _, alias := range aliases {
		if alias.Value == username {
			names = append(names, alias.Alias)
		}
	}
	return names
}

// loadOrg reads the authorization for a username or alias and builds a
// session for it.  A refreshed access token is saved to the authorization
// file.
func loadOrg(nameOrAlias string) (*targetOrg, error) {
	username := lib.ResolveSFDXUsername(nameOrAlias)
	fields, order, err := lib.ReadSFDXAuthFieldsOrdered(username)
	if err != nil {
		return nil, err
	}
	username = fields.String("username")
	instanceURL := strings.TrimRight(fields.String("instanceUrl"), "/")
	loginURL := strings.TrimRight(fields.String("loginUrl"), "/")
	if loginURL == "" {
		loginURL = instanceURL
	}
	session := &lib.ForceSession{
		AccessToken:  fields.String("accessToken"),
		InstanceUrl:  instanceURL,
		EndpointUrl:  loginURL,
		ClientId:     fields.String("clientId"),
		RefreshToken: fields.String("refreshToken"),
		Id:           fields.String("orgId"),
		UserInfo: &lib.UserInfo{
			UserName:     username,
			OrgId:        fields.String("orgId"),
			UserId:       fields.String("userId"),
			OrgNamespace: fields.String("namespacePrefix"),
		},
		SessionOptions: &lib.SessionOptions{
			ApiVersion: fields.String("instanceApiVersion"),
			Alias:      username,
		},
	}
	session.SessionOptions.RefreshFunc = func(f *lib.Force) error {
		if f.Credentials.RefreshToken == "" {
			return fmt.Errorf("Unable to refresh session: expired access/refresh token")
		}
		if err := f.RefreshAccessToken(); err != nil {
			return err
		}
		return lib.UpdateSFDXAuthFields(username, lib.SFDXAuthFields{"accessToken": f.Credentials.AccessToken})
	}
	return &targetOrg{
		Username: username,
		Alias:    aliasFor(username),
		Fields:   fields,
		Order:    order,
		Force:    lib.NewForce(session),
	}, nil
}

// resolveTargetOrg returns the org a command acts on: the value of
// --target-org, or the target-org config value.  It also selects the API
// version from --api-version, the org-api-version config value, or the
// version recorded in the authorization file.
func resolveTargetOrg(cmd *cobra.Command) (*targetOrg, error) {
	return resolveOrg(cmd, targetOrgFlag(cmd), "target-org", lib.SFConfigTargetOrg)
}

// resolveTargetDevHub is resolveTargetOrg for --target-dev-hub.
func resolveTargetDevHub(cmd *cobra.Command) (*targetOrg, error) {
	name, _ := cmd.Flags().GetString("target-dev-hub")
	return resolveOrg(cmd, name, "target-dev-hub", lib.SFConfigTargetDevHub)
}

func resolveOrg(cmd *cobra.Command, name, flagName, configKey string) (*targetOrg, error) {
	fromFlag := name != ""
	if name == "" {
		value, found, err := lib.GetSFConfig(configKey)
		if err != nil {
			return nil, err
		}
		if !found || value.Value == "" {
			if configKey == lib.SFConfigTargetDevHub {
				return nil, newError("NoDefaultDevHubError", "No default dev hub found. Use -v or --target-dev-hub to specify an environment.")
			}
			return nil, newError("NoDefaultEnvError", "No default environment found. Use -o or --target-org to specify an environment.")
		}
		name = value.Value
	}
	org, err := loadOrg(name)
	if err != nil {
		if fromFlag {
			return nil, &Error{
				Name:     "NamedOrgNotFoundError",
				Message:  fmt.Sprintf("Parsing --%s \n\tNo authorization information found for %s.\nSee more help with --help", flagName, name),
				ExitCode: 2,
			}
		}
		return nil, &Error{
			Name:     "NamedOrgNotFoundError",
			Message:  fmt.Sprintf("No authorization information found for %s.", name),
			ExitCode: 1,
		}
	}
	version := ""
	if flag := cmd.Flags().Lookup("api-version"); flag != nil {
		version = flag.Value.String()
	}
	if version == "" {
		if value, found, _ := lib.GetSFConfig(lib.SFConfigApiVersion); found {
			version = value.Value
		}
	}
	if version == "" {
		version = org.Fields.String("instanceApiVersion")
	}
	if version != "" {
		if err := lib.SetApiVersion(version); err != nil {
			return nil, newError("InvalidApiVersionError", "%s", err.Error())
		}
		org.Force.Credentials.SessionOptions.ApiVersion = version
	}
	return org, nil
}

// connectedStatus makes a request to the org and returns "Connected" or
// the reason the request failed.
func connectedStatus(org *targetOrg) string {
	_, err := org.Force.GetAbsolute("/services/data")
	if err == nil {
		return "Connected"
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "connection refused"):
		return "ECONNREFUSED"
	case strings.Contains(message, "no such host"):
		return "ENOTFOUND"
	}
	return message
}

const redactedSecret = "[REDACTED] Use 'sf org auth show-access-token' to view"

func secretsHiddenWarning(command string) string {
	return fmt.Sprintf("Secrets are now hidden from 'sf %s' command output. Use the 'sf org auth' commands instead. As a temporary workaround, you can set SF_TEMP_SHOW_SECRETS=true to render these secrets. This workaround will be removed in an upcoming release.", command)
}

// defaultUsernames returns the usernames that the target-org and
// target-dev-hub config values refer to.
func defaultUsernames() (org string, devHub string) {
	if value, found, _ := lib.GetSFConfig(lib.SFConfigTargetOrg); found {
		org = lib.ResolveSFDXUsername(value.Value)
	}
	if value, found, _ := lib.GetSFConfig(lib.SFConfigTargetDevHub); found {
		devHub = lib.ResolveSFDXUsername(value.Value)
	}
	return
}
