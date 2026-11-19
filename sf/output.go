package sf

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
)

type envelope struct {
	Status   int         `json:"status"`
	Result   interface{} `json:"result"`
	Warnings []string    `json:"warnings"`
}

func writeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func jsonRequested(cmd *cobra.Command) bool {
	flag := cmd.Flags().Lookup("json")
	return flag != nil && flag.Value.String() == "true"
}

// emit prints the result of a command.  With --json it prints sf's envelope
// around result; otherwise it calls human to print the human-readable form.
func emit(cmd *cobra.Command, result interface{}, human func(w io.Writer)) error {
	return emitStatus(cmd, 0, result, human)
}

func emitStatus(cmd *cobra.Command, status int, result interface{}, human func(w io.Writer)) error {
	if jsonRequested(cmd) {
		return writeJSON(cmd.OutOrStdout(), envelope{Status: status, Result: result, Warnings: []string{}})
	}
	if human != nil {
		human(cmd.OutOrStdout())
	}
	return nil
}

// printTable prints a table in the format sf uses when stdout is not a
// terminal: an optional title line, a box-drawing border, and a blank line.
func printTable(w io.Writer, title string, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = runewidth.StringWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && runewidth.StringWidth(cell) > widths[i] {
				widths[i] = runewidth.StringWidth(cell)
			}
		}
	}
	border := func(left, middle, right string) string {
		parts := make([]string, len(widths))
		for i, width := range widths {
			parts[i] = strings.Repeat("─", width+2)
		}
		return left + strings.Join(parts, middle) + right
	}
	line := func(cells []string) string {
		parts := make([]string, len(widths))
		for i, width := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			parts[i] = " " + cell + strings.Repeat(" ", width-runewidth.StringWidth(cell)) + " "
		}
		return "│" + strings.Join(parts, "│") + "│"
	}
	if title != "" {
		fmt.Fprintln(w, title)
	}
	fmt.Fprintln(w, border("┌", "┬", "┐"))
	fmt.Fprintln(w, line(headers))
	fmt.Fprintln(w, border("├", "┼", "┤"))
	for _, row := range rows {
		fmt.Fprintln(w, line(row))
	}
	fmt.Fprintln(w, border("└", "┴", "┘"))
	fmt.Fprintln(w)
}
