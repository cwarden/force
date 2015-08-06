package main

import (
	"fmt"
	"strings"
)

var cmdQuery = &Command{
	Run:   runQuery,
	Usage: "query <soql statement> [output format]",
	Short: "Execute a SOQL statement",
	Long: `
Execute a SOQL statement

Examples:

  force query "select Id, Name, Account.Name From Contact"

  force query "select Id, Name, Account.Name From Contact" --format:csv

	force query "select Id, Name From Account Where MailingState IN ('CA', 'NY')"
`,
}

func runQuery(cmd *Command, args []string) {
	force, _ := ActiveForce()
	if len(args) < 1 {
		cmd.printUsage()
	} else {
		format := "console"
		formatArg := args[len(args)-1]

		if strings.Contains(formatArg, "format:") {
			args = args[:len(args)-1]
			format = strings.SplitN(formatArg, ":", 2)[1]
		}
		toolingArg := args[len(args)-1]
		useTooling := false
		fmt.Printf("toolingArg %s", toolingArg);
		if strings.Contains(toolingArg, "useTooling") {
			useTooling = true
			args = args[:len(args)-1]
		}

		soql := strings.Join(args, " ")
		var records ForceQueryResult
		var err error
		if (useTooling) {
			records, err = force.QueryTooling(fmt.Sprintf("%s", soql))
		} else {
			records, err = force.Query(fmt.Sprintf("%s", soql))
		}
		if err != nil {
			ErrorAndExit(err.Error())
		} else {
			if format == "console" {
				DisplayForceRecords(records)
			} else {
				DisplayForceRecordsf(records.Records, format)
			}
		}
	}
}
