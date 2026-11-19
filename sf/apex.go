package sf

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/lib/bayeux"
)

func newApexCmd() *cobra.Command {
	return newGroup("apex", "Use the apex commands to create Apex classes, execute anonymous blocks, view your logs, run Apex tests, and view Apex test results.",
		withSubcommands(newApexRunCmd(), newApexRunTestCmd()),
		newGroup("get", "Fetch Apex debug logs and test results.", newApexGetLogCmd(), newApexGetTestCmd()),
		newGroup("list", "List Apex debug logs.", newApexListLogCmd()),
		newGroup("tail", "Tail Apex debug logs.", newApexTailLogCmd()),
		newGroup("trace", "Manage trace flags.", newApexTraceCreateCmd(), newApexTraceListCmd(), newApexTraceDeleteCmd()),
	)
}

func newApexRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "run",
		Short:       "Execute anonymous Apex code entered on the command line or from a local file.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Run"},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			var code []byte
			if file, _ := cmd.Flags().GetString("file"); file != "" {
				code, err = os.ReadFile(file)
				if err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Start typing Apex code. Press the Enter key after each line, then press CTRL+D when finished.")
				code, err = io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
			}
			executed, err := org.Force.Partner.ExecuteAnonymousResult(string(code))
			if err != nil {
				return apiError(err)
			}
			result := newOrderedMap()
			result.Set("success", executed.Success)
			result.Set("compiled", executed.Compiled)
			result.Set("compileProblem", executed.CompileProblem)
			result.Set("exceptionMessage", executed.ExceptionMessage)
			result.Set("exceptionStackTrace", executed.ExceptionStackTrace)
			result.Set("line", executed.Line)
			result.Set("column", executed.Column)
			result.Set("logs", executed.Log)
			if !executed.Compiled || !executed.Success {
				// sf reports the position as text in a failure.
				result.Set("line", fmt.Sprintf("%d", executed.Line))
				result.Set("column", fmt.Sprintf("%d", executed.Column))
				if !executed.Compiled {
					return &Error{
						Name:     "executeCompileFailure",
						Message:  fmt.Sprintf("Compilation failed at Line %d column %d with the error:\n\n%s", executed.Line, executed.Column, executed.CompileProblem),
						ExitCode: 1,
						Data:     result,
					}
				}
				return &Error{
					Name:     "executeRuntimeFailure",
					Message:  fmt.Sprintf("Execution failed at this code:\n\n%s", executed.ExceptionMessage),
					ExitCode: 1,
					Data:     result,
				}
			}
			return emit(cmd, result, func(w io.Writer) {
				fmt.Fprintf(w, "Compiled successfully.\nExecuted successfully.\n\n%s\n", executed.Log)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("file", "f", "", "Path to a local file that contains Apex code.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

const logFields = "Id, Application, DurationMilliseconds, Location, LogLength, LogUser.Name, Operation, Request, StartTime, Status"

// formatLogStartTime removes the milliseconds from a log start time, as sf
// does: 2026-01-02T03:04:05.000+0000 becomes 2026-01-02T03:04:05+0000.
func formatLogStartTime(value string) string {
	index := strings.Index(value, ".")
	if index < 0 || index+4 > len(value) {
		return value
	}
	return value[:index] + value[index+4:]
}

func queryLogRecords(org *targetOrg, limit int) ([]*orderedMap, error) {
	soql := "Select " + logFields + " from ApexLog Order By StartTime DESC"
	if limit > 0 {
		soql += fmt.Sprintf(" LIMIT %d", limit)
	}
	result, err := runQuery(org, "/tooling", "query", soql)
	if err != nil {
		return nil, queryError(err)
	}
	records, _ := result.values["records"].([]interface{})
	logs := make([]*orderedMap, 0, len(records))
	for _, record := range records {
		if object, ok := record.(*orderedMap); ok {
			if start, ok := object.values["StartTime"].(string); ok {
				object.Set("StartTime", formatLogStartTime(start))
			}
			logs = append(logs, object)
		}
	}
	return logs, nil
}

func newApexListLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "log",
		Short:       "Display a list of IDs and general information about debug logs.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			logs, err := queryLogRecords(org, 0)
			if err != nil {
				return err
			}
			if len(logs) == 0 {
				return emit(cmd, []interface{}{}, func(w io.Writer) {
					fmt.Fprintln(w, "No debug logs found in org")
				})
			}
			rows := make([][]string, 0, len(logs))
			for _, log := range logs {
				user := ""
				if logUser, ok := log.values["LogUser"].(*orderedMap); ok {
					user = displayValue(logUser.values["Name"])
				}
				rows = append(rows, []string{
					displayValue(log.values["Application"]),
					displayValue(log.values["DurationMilliseconds"]),
					displayValue(log.values["Id"]),
					displayValue(log.values["Location"]),
					displayValue(log.values["LogLength"]),
					user,
					displayValue(log.values["Operation"]),
					displayValue(log.values["Request"]),
					displayValue(log.values["StartTime"]),
					displayValue(log.values["Status"]),
				})
			}
			return emit(cmd, logs, func(w io.Writer) {
				printTable(w, "", []string{"Application", "Duration (ms)", "Id", "Location", "Size (B)", "Log User", "Operation", "Request", "Start Time", "Status"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newApexGetLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "log",
		Short:       "Fetch the specified log or given number of most recent logs from the org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			logID, _ := cmd.Flags().GetString("log-id")
			number, _ := cmd.Flags().GetInt("number")
			outputDir, _ := cmd.Flags().GetString("output-dir")
			if cmd.Flags().Changed("number") && (number < 1 || number > 25) {
				return &Error{Message: fmt.Sprintf("Parsing --number \n\tExpected an integer between 1 and 25 but received: %d\nSee more help with --help", number), ExitCode: 2}
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			var ids []string
			if logID != "" {
				ids = []string{logID}
			} else {
				if number == 0 {
					number = 1
				}
				logs, err := queryLogRecords(org, number)
				if err != nil {
					return err
				}
				for _, log := range logs {
					ids = append(ids, displayValue(log.values["Id"]))
				}
			}
			if len(ids) == 0 {
				return emit(cmd, []interface{}{}, func(w io.Writer) {
					fmt.Fprintln(w, "No results found")
				})
			}
			bodies := make([]string, 0, len(ids))
			for _, id := range ids {
				body, err := org.Force.RetrieveLog(id)
				if err != nil {
					return apiError(err)
				}
				bodies = append(bodies, body)
			}
			if outputDir != "" {
				if err := os.MkdirAll(outputDir, 0o755); err != nil {
					return err
				}
				paths := make([]string, 0, len(ids))
				for i, id := range ids {
					path := filepath.Join(outputDir, id+".log")
					if err := os.WriteFile(path, []byte(bodies[i]), 0o644); err != nil {
						return err
					}
					paths = append(paths, path)
				}
				return emit(cmd, paths, func(w io.Writer) {
					fmt.Fprintf(w, "Log files written to %s\n", outputDir)
				})
			}
			result := make([]map[string]string, 0, len(bodies))
			for _, body := range bodies {
				result = append(result, map[string]string{"log": body})
			}
			return emit(cmd, result, func(w io.Writer) {
				for _, body := range bodies {
					fmt.Fprintln(w, body)
				}
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("log-id", "i", "", "ID of the specific log to display.")
	cmd.Flags().IntP("number", "n", 0, "Number of the most recent logs to display.")
	cmd.Flags().StringP("output-dir", "d", "", "Directory for saving the log files.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newApexTailLogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "log",
		Short:       "Activate debug logging and display logs in the terminal.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Log"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "debug-level"); err != nil {
				return err
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			if skip, _ := cmd.Flags().GetBool("skip-trace-flag"); !skip {
				// StartTrace reports an existing trace flag as an error;
				// logging is already active in that case.
				if _, err, _ := org.Force.StartTrace(); err != nil && !strings.Contains(err.Error(), "already") {
					return apiError(err)
				}
			}
			client := bayeux.NewClient(org.Force)
			messages := make(chan *bayeux.Message)
			if err := client.Subscribe("/systemTopic/Logging", messages); err != nil {
				return err
			}
			defer client.Close()
			seen := map[string]bool{}
			out := cmd.OutOrStdout()
			for message := range messages {
				var event struct {
					Sobject struct {
						Id string `json:"Id"`
					} `json:"sobject"`
				}
				if err := json.Unmarshal(message.Data, &event); err != nil || event.Sobject.Id == "" || seen[event.Sobject.Id] {
					continue
				}
				seen[event.Sobject.Id] = true
				body, err := org.Force.RetrieveLog(event.Sobject.Id)
				if err != nil {
					return apiError(err)
				}
				fmt.Fprintln(out, body)
			}
			fmt.Fprintln(out, "Finished tailing logs")
			return nil
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().BoolP("color", "c", false, "Apply default colors to noteworthy log lines.")
	cmd.Flags().StringP("debug-level", "d", "", "Debug level to set on the DEVELOPER_LOG trace flag for your user.")
	cmd.Flags().BoolP("skip-trace-flag", "s", false, "Skip trace flag setup.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newApexTraceCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "create",
		Short:       "Create a trace flag for a user or automated process.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "CreateTrace"},
		RunE: func(cmd *cobra.Command, args []string) error {
			entityID, _ := cmd.Flags().GetString("traced-entity-id")
			debugLevel, _ := cmd.Flags().GetString("debug-level")
			logType, _ := cmd.Flags().GetString("log-type")
			duration, _ := cmd.Flags().GetInt("duration")
			if entityID == "" {
				return missingFlagError("traced-entity-id")
			}
			if debugLevel == "" {
				return missingFlagError("debug-level")
			}
			if logType != "DEVELOPER_LOG" && logType != "USER_DEBUG" {
				return &Error{Message: fmt.Sprintf("Expected --log-type=%s to be one of: DEVELOPER_LOG, USER_DEBUG\nSee more help with --help", logType), ExitCode: 2}
			}
			if duration < 1 || duration > 1440 {
				return &Error{Message: fmt.Sprintf("Parsing --duration \n\tExpected an integer between 1 and 1440 but received: %d\nSee more help with --help", duration), ExitCode: 2}
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			soql := fmt.Sprintf("SELECT Id FROM DebugLevel WHERE DeveloperName = '%s'", strings.ReplaceAll(debugLevel, "'", `\'`))
			levels, err := runQuery(org, "/tooling", "query", soql)
			if err != nil {
				return apiError(err)
			}
			records, _ := levels.values["records"].([]interface{})
			if len(records) == 0 {
				return &Error{Message: fmt.Sprintf("Debug level \"%s\" not found. Create one in Setup or specify an existing debug level.", debugLevel), ExitCode: 1}
			}
			level, _ := records[0].(*orderedMap)
			start := time.Now().UTC()
			request := newOrderedMap()
			request.Set("TracedEntityId", entityID)
			request.Set("LogType", logType)
			request.Set("DebugLevelId", level.values["Id"])
			request.Set("StartDate", start.Format("2006-01-02T15:04:05.000Z"))
			request.Set("ExpirationDate", start.Add(time.Duration(duration)*time.Minute).Format("2006-01-02T15:04:05.000Z"))
			content, err := encodeJSON(request)
			if err != nil {
				return err
			}
			body, err := org.Force.PostREST("/tooling/sobjects/TraceFlag", content)
			if err != nil {
				return apiError(err)
			}
			var created struct {
				Id      string `json:"id"`
				Success bool   `json:"success"`
			}
			if err := json.Unmarshal([]byte(body), &created); err != nil || !created.Success {
				return &Error{Message: "Failed to create trace flag.", ExitCode: 1}
			}
			result := newOrderedMap()
			result.Set("id", created.Id)
			result.Set("success", true)
			return emit(cmd, result, func(w io.Writer) {
				fmt.Fprintf(w, "Successfully created trace flag %s.\n", created.Id)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("traced-entity-id", "i", "", "ID of the user or automated process to trace.")
	cmd.Flags().StringP("debug-level", "d", "", "Developer name of the debug level to assign to the trace flag.")
	cmd.Flags().StringP("log-type", "l", "DEVELOPER_LOG", "Type of log to generate.")
	cmd.Flags().Int("duration", 30, "Number of minutes the trace flag remains active.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newApexTraceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "list",
		Short:       "Display a list of trace flags in the org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ListTrace"},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			result, err := runQuery(org, "/tooling", "query", "SELECT Id, TracedEntityId, LogType, DebugLevelId, StartDate, ExpirationDate, DebugLevel.DeveloperName, TracedEntity.Name FROM TraceFlag ORDER BY CreatedDate DESC")
			if err != nil {
				return apiError(err)
			}
			records, _ := result.values["records"].([]interface{})
			if len(records) == 0 {
				return emit(cmd, []interface{}{}, func(w io.Writer) {
					fmt.Fprintln(w, "No trace flags found in org.")
				})
			}
			now := time.Now()
			nameOr := func(record *orderedMap, relationship, field, fallback string) string {
				if related, ok := record.values[relationship].(*orderedMap); ok {
					if name, ok := related.values[field].(string); ok {
						return name
					}
				}
				return displayValue(record.values[fallback])
			}
			flags := make([]*orderedMap, 0, len(records))
			rows := make([][]string, 0, len(records))
			for _, item := range records {
				record, ok := item.(*orderedMap)
				if !ok {
					continue
				}
				status := "Expired"
				expiration, _ := record.values["ExpirationDate"].(string)
				for _, layout := range []string{"2006-01-02T15:04:05.000-0700", time.RFC3339Nano, "2006-01-02T15:04:05.000Z"} {
					if expires, err := time.Parse(layout, expiration); err == nil {
						if expires.After(now) {
							status = "Active"
						}
						break
					}
				}
				record.Set("Status", status)
				flags = append(flags, record)
				rows = append(rows, []string{
					displayValue(record.values["Id"]),
					nameOr(record, "TracedEntity", "Name", "TracedEntityId"),
					displayValue(record.values["LogType"]),
					nameOr(record, "DebugLevel", "DeveloperName", "DebugLevelId"),
					status,
					expiration,
				})
			}
			return emit(cmd, flags, func(w io.Writer) {
				printTable(w, "", []string{"Id", "Traced Entity", "Log Type", "Debug Level", "Status", "Expiration Date"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newApexTraceDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "delete",
		Short:       "Delete a trace flag from the org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "DeleteTrace"},
		RunE: func(cmd *cobra.Command, args []string) error {
			id, _ := cmd.Flags().GetString("trace-flag-id")
			if id == "" {
				return missingFlagError("trace-flag-id")
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			if _, err := org.Force.DeleteREST("/tooling/sobjects/TraceFlag/" + id); err != nil {
				return apiError(err)
			}
			result := newOrderedMap()
			result.Set("id", id)
			result.Set("success", true)
			return emit(cmd, result, func(w io.Writer) {
				fmt.Fprintf(w, "Successfully deleted trace flag %s.\n", id)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("trace-flag-id", "i", "", "ID of the trace flag to delete.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}
