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
		return writeJSON(cmd.OutOrStdout(), envelope{Status: status, Result: result, Warnings: warningsOf(cmd)})
	}
	if human != nil {
		human(cmd.OutOrStdout())
	}
	return nil
}

type runState struct {
	warnings []string
}

type runStateKey struct{}

func stateOf(cmd *cobra.Command) *runState {
	if cmd == nil || cmd.Context() == nil {
		return nil
	}
	state, _ := cmd.Context().Value(runStateKey{}).(*runState)
	return state
}

func warningsOf(cmd *cobra.Command) []string {
	if state := stateOf(cmd); state != nil && len(state.warnings) > 0 {
		return state.warnings
	}
	return []string{}
}

// warn reports a warning.  With --json it is added to the "warnings" array
// of the envelope; otherwise it is printed on stderr.
func warn(cmd *cobra.Command, message string) {
	if jsonRequested(cmd) {
		warnJSON(cmd, message)
		return
	}
	errOut := cmd.ErrOrStderr()
	fmt.Fprintf(errOut, "%s %s\n", paint(errOut, styleBoldYellow, "Warning:"), message)
}

// warnJSON adds a warning to the envelope and prints nothing in human
// output.
func warnJSON(cmd *cobra.Command, message string) {
	if state := stateOf(cmd); state != nil && jsonRequested(cmd) {
		state.warnings = append(state.warnings, message)
	}
}

// printTable prints a table in the format sf uses: an optional title line,
// a box-drawing border, and a blank line.
func printTable(w io.Writer, title string, headers []string, rows [][]string) {
	printStyledTable(w, title, headers, rows, nil)
}

// printStyledTable is printTable with a style for each row.  The style
// applies to the text of each cell and not to its padding.
func printStyledTable(w io.Writer, title string, headers []string, rows [][]string, rowStyle func(row int) *style) {
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
	line := func(cells []string, cellStyle *style) string {
		parts := make([]string, len(widths))
		for i, width := range widths {
			cell := ""
			if i < len(cells) {
				cell = cells[i]
			}
			text := cell
			if cellStyle != nil && cell != "" {
				text = paint(w, *cellStyle, cell)
			}
			parts[i] = " " + text + strings.Repeat(" ", width-runewidth.StringWidth(cell)) + " "
		}
		return "│" + strings.Join(parts, "│") + "│"
	}
	if title != "" {
		fmt.Fprintln(w, title)
	}
	fmt.Fprintln(w, border("┌", "┬", "┐"))
	fmt.Fprintln(w, line(headers, nil))
	fmt.Fprintln(w, border("├", "┼", "┤"))
	for i, row := range rows {
		var cellStyle *style
		if rowStyle != nil {
			cellStyle = rowStyle(i)
		}
		fmt.Fprintln(w, line(row, cellStyle))
	}
	fmt.Fprintln(w, border("└", "┴", "┘"))
	fmt.Fprintln(w)
}
