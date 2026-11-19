package sf

import (
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type commandInfo struct {
	Id      string   `json:"id"`
	Summary string   `json:"summary"`
	Flags   []string `json:"flags"`
}

// listCommands returns the commands of the tree that run something, with
// ids in sf's colon form.
func listCommands(root *cobra.Command) []commandInfo {
	var commands []commandInfo
	var walk func(cmd *cobra.Command, path []string)
	walk = func(cmd *cobra.Command, path []string) {
		for _, child := range cmd.Commands() {
			if child.Hidden || child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			childPath := append(append([]string{}, path...), child.Name())
			// A topic that only holds subcommands has flags of its own
			// only when it is also a command.
			if child.Flags().HasFlags() || !child.HasSubCommands() {
				info := commandInfo{Id: strings.Join(childPath, ":"), Summary: child.Short, Flags: []string{}}
				child.Flags().VisitAll(func(flag *pflag.Flag) {
					if !flag.Hidden {
						info.Flags = append(info.Flags, flag.Name)
					}
				})
				sort.Strings(info.Flags)
				commands = append(commands, info)
			}
			walk(child, childPath)
		}
	}
	walk(root, nil)
	sort.Slice(commands, func(i, j int) bool { return commands[i].Id < commands[j].Id })
	return commands
}

func newCommandsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commands",
		Short: "List all sf commands.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			commands := listCommands(cmd.Root())
			if jsonRequested(cmd) {
				// sf prints this array without the status envelope.
				return writeJSON(cmd.OutOrStdout(), commands)
			}
			rows := make([][]string, 0, len(commands))
			for _, command := range commands {
				rows = append(rows, []string{strings.ReplaceAll(command.Id, ":", " "), command.Summary})
			}
			return emit(cmd, commands, func(w io.Writer) {
				printTable(w, "", []string{"Id", "Summary"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	return cmd
}
