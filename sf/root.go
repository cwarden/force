// Package sf implements the command tree used when the force binary is
// invoked under the name sf.  Commands accept the names and flags of the
// Salesforce sf CLI and print output in its format.
package sf

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	forceerror "github.com/ForceCLI/force/error"
)

// CompatVersion is the sf release whose commands, flags and output this
// package reproduces.
const CompatVersion = "2.138.6"

// InvokedAsSf reports whether argv0 names the binary sf.
func InvokedAsSf(argv0 string) bool {
	name := filepath.Base(strings.ReplaceAll(argv0, "\\", "/"))
	if len(name) > 4 && strings.EqualFold(name[len(name)-4:], ".exe") {
		name = name[:len(name)-4]
	}
	return name == "sf"
}

// ModeFlag is the force flag that selects sf-compatible mode.
const ModeFlag = "--sf"

// StripModeFlag reports whether args contain --sf and returns args without
// it.  An argument after "--" is not a flag.  "force --sf org list" and
// "force org list --sf" both run "sf org list".
func StripModeFlag(args []string) ([]string, bool) {
	for i, arg := range args {
		if arg == "--" {
			break
		}
		if arg == ModeFlag {
			rest := append([]string{}, args[:i]...)
			return append(rest, args[i+1:]...), true
		}
	}
	return args, false
}

// Execute runs the sf command tree with the process arguments and exits.
func Execute() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

// Run executes the sf command tree and returns the exit status.
func Run(args []string, out, errOut io.Writer) int {
	root := NewRootCmd()
	args = normalizeArgsFor(root, args)
	jsonOutput := hasJSONFlag(args)
	root.SetArgs(args)
	root.SetContext(context.WithValue(context.Background(), runStateKey{}, &runState{}))
	root.SetOut(out)
	root.SetErr(errOut)

	// lib reports some failures by calling ErrorAndExit.  Print those in
	// sf's format.  ErrorAndExit exits with status 1 after the handler.
	previous := forceerror.ExitHandler
	forceerror.ExitHandler = func(message string) {
		writeError(out, errOut, nil, &Error{Message: message, ExitCode: 1}, jsonOutput)
	}
	defer func() { forceerror.ExitHandler = previous }()

	cmd, err := root.ExecuteC()
	if err != nil {
		return writeError(out, errOut, cmd, err, jsonOutput)
	}
	return 0
}

// NewRootCmd builds a new sf command tree.  Each call returns independent
// commands and flag values.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "sf",
		Short:         "sf-compatible mode of the force CLI",
		Args:          cobra.ArbitraryArgs,
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       versionString(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return &unknownCommandError{id: args[0]}
			}
			return cmd.Help()
		},
	}
	root.SetVersionTemplate("{{.Version}}\n")
	root.Flags().BoolP("version", "v", false, "Show the version")
	root.SetGlobalNormalizationFunc(normalizeFlagName)
	root.AddCommand(
		newAliasCmd(),
		newApexCmd(),
		newApiCmd(),
		newCommandsCmd(),
		newConfigCmd(),
		newDataCmd(),
		newForceLegacyCmd(),
		newOrgCmd(),
		newPackageCmd(),
		newProjectCmd(),
		newSobjectCmd(),
		newTemplateCmd(),
		newVersionCmd(),
	)
	return root
}

// newGroup returns a command that only holds subcommands, such as "org" or
// "org login".
func newGroup(use, short string, children ...*cobra.Command) *cobra.Command {
	group := &cobra.Command{
		Use:         use,
		Short:       short,
		Annotations: map[string]string{topicAnnotation: "true"},
		// An unknown command under a topic is reported as an unknown
		// command even when flags follow it.
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				id := strings.Join(append(strings.Fields(cmd.CommandPath())[1:], args...), " ")
				return &unknownCommandError{id: id}
			}
			return cmd.Help()
		},
	}
	group.AddCommand(children...)
	return group
}

// withSubcommands adds children to a command that also runs on its own,
// such as "org list", which has the subcommand "org list auth".
func withSubcommands(cmd *cobra.Command, children ...*cobra.Command) *cobra.Command {
	cmd.AddCommand(children...)
	return cmd
}

// legacyFlagNames maps flag names from sfdx to the sf names.
var legacyFlagNames = map[string]string{
	"targetusername":       "target-org",
	"targetdevhubusername": "target-dev-hub",
	"target-hub-org":       "target-dev-hub",
	"apiversion":           "api-version",
}

func normalizeFlagName(f *pflag.FlagSet, name string) pflag.NormalizedName {
	if current, ok := legacyFlagNames[name]; ok {
		return pflag.NormalizedName(current)
	}
	return pflag.NormalizedName(name)
}

const topicAnnotation = "sfTopic"

// normalizeArgsFor is normalizeArgs for a command tree.  A legacy id is not
// rewritten when the words already name a command of the tree that is at
// least as long: "project generate manifest" is a command, although
// "project generate" is a legacy id of "template generate project".
func normalizeArgsFor(root *cobra.Command, args []string) []string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args
	}
	words := append(strings.Split(args[0], ":"), args[1:]...)
	positional := 0
	for positional < len(words) && !strings.HasPrefix(words[positional], "-") {
		positional++
	}
	if found, _, err := root.Find(words[:positional]); err == nil && found != root && found.Annotations[topicAnnotation] == "" {
		return words
	}
	return normalizeArgs(args)
}

// normalizeArgs converts the colon form of a command id ("org:login:web") to
// separate words and replaces legacy ids ("force:org:list", "deploy
// metadata") with the current id.
func normalizeArgs(args []string) []string {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return args
	}
	words := strings.Split(args[0], ":")
	rest := args[1:]
	// The words of an id can also be given as separate arguments.
	positional := 0
	for positional < len(rest) && !strings.HasPrefix(rest[positional], "-") {
		positional++
	}
	candidates := append(append([]string{}, words...), rest[:positional]...)
	for n := len(candidates); n >= 1; n-- {
		if n < len(words) {
			break
		}
		if current, ok := legacyCommandIds[strings.Join(candidates[:n], ":")]; ok {
			result := strings.Split(current, ":")
			result = append(result, candidates[n:]...)
			return append(result, rest[positional:]...)
		}
	}
	return append(words, rest...)
}

func hasJSONFlag(args []string) bool {
	for _, arg := range args {
		if arg == "--" {
			return false
		}
		if arg == "--json" || arg == "--json=true" {
			return true
		}
	}
	return false
}

// Shared flag helpers.  Commands read the values with cmd.Flags().

func addJSONFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "Format output as json.")
}

// addTargetOrgFlag adds --target-org/-o.  It also accepts the sfdx short
// flag -u when the command does not use -u for another flag, so call it
// after defining the command's own flags.
func addTargetOrgFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("target-org", "o", "", "Username or alias of the target org.")
	if cmd.Flags().ShorthandLookup("u") == nil {
		cmd.Flags().StringP("targetusername-short", "u", "", "")
		cmd.Flags().MarkHidden("targetusername-short")
	}
}

func addTargetDevHubFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("target-dev-hub", "v", "", "Username or alias of the Dev Hub org.")
}

func addAPIVersionFlag(cmd *cobra.Command) {
	cmd.Flags().String("api-version", "", "Override the api version used for api requests made by this command")
}

// missingFlagError is the error sf reports for a required flag that was not
// given.
func missingFlagError(name string) *Error {
	return &Error{Message: fmt.Sprintf("Missing required flag %s\nSee more help with --help", name), ExitCode: 2}
}

// targetOrgFlag returns the value of --target-org, or of -u when only the
// sfdx short flag was given.
func targetOrgFlag(cmd *cobra.Command) string {
	if value, _ := cmd.Flags().GetString("target-org"); value != "" {
		return value
	}
	if cmd.Flags().Lookup("targetusername-short") != nil {
		value, _ := cmd.Flags().GetString("targetusername-short")
		return value
	}
	return ""
}

func versionString() string {
	return fmt.Sprintf("@salesforce/cli/%s %s force-%s", CompatVersion, platform(), forceVersion())
}
