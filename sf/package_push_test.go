package sf

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPackagePushUpgradeCommands(t *testing.T) {
	org := devHubSetup(t)
	var queries []string
	patched := ""
	status := "Failed"
	org.handlers["/services/data/v67.0/query"] = func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		queries = append(queries, query)
		switch {
		case strings.Contains(query, "033000000000009AAA") || strings.Contains(query, "0DV000000000009AAA"):
			io.WriteString(w, `{"done":true,"totalSize":0,"records":[]}`)
		case strings.Contains(query, "FROM PackagePushJob") && strings.Contains(query, "Status = 'Failed'"):
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"attributes":{"type":"AggregateResult"},"expr0":1}]}`)
		case strings.Contains(query, "FROM PackagePushJob") && strings.Contains(query, "Status = 'Succeeded'"):
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"attributes":{"type":"AggregateResult"},"expr0":2}]}`)
		case strings.Contains(query, "FROM PackagePushJob"):
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"attributes":{"type":"AggregateResult"},"expr0":3}]}`)
		case strings.Contains(query, "from PackagePushError"):
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"ErrorMessage":"Apex compile failure"}]}`)
		case strings.HasPrefix(query, "SELECT Id, Status FROM PackagePushRequest"):
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"Id":"0DV000000000001AAA","Status":"`+status+`"}]}`)
		default:
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"attributes":{"type":"PackagePushRequest"},"Id":"0DV000000000001AAA","PackageVersionId":"04t000000000001AAA",`+
				`"PackageVersion":{"Name":"Spring","MajorVersion":1,"MinorVersion":2,"MetadataPackageId":"033000000000001AAA","MetadataPackage":{"Name":"Billing","NamespacePrefix":"acme"}},`+
				`"Status":"Failed","ScheduledStartTime":"2026-01-02T03:04:05.000+0000","StartTime":null,"EndTime":null,"DurationSeconds":7,"IsMigration":false}]}`)
		}
	}
	org.handlers["/services/data/v67.0/sobjects/packagepushrequest/0DV000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		patched = r.Method + " " + string(body)
		w.WriteHeader(http.StatusNoContent)
	}

	// Compared with sf: a package without push upgrade requests.
	out, errOut, code := run("package", "push-upgrade", "list", "-p", "033000000000009AAA")
	if code != 0 || out != "" || errOut != "Warning: No results found\n" {
		t.Errorf("empty list: status %d, stdout %q, stderr %q", code, out, errOut)
	}

	queries = nil
	expectRun(t, 0, `┌────────────────────┬────────────────────┬──────────────────────┬────────────────────────┬────────┬──────────────────────────────┬────────────┬──────────┬────────────────────┬────────────────────────┬───────────────────────────┐
│ Id                 │ Package Version Id │ Package Version Name │ Package Version Number │ Status │ Scheduled Start Time         │ Start Time │ End Time │ Num Orgs Scheduled │ Num Orgs Upgraded Fail │ Num Orgs Upgraded Success │
├────────────────────┼────────────────────┼──────────────────────┼────────────────────────┼────────┼──────────────────────────────┼────────────┼──────────┼────────────────────┼────────────────────────┼───────────────────────────┤
│ 0DV000000000001AAA │ 04t000000000001AAA │ Spring               │ 1.2                    │ Failed │ 2026-01-02T03:04:05.000+0000 │            │          │ 3                  │ 1                      │ 2                         │
└────────────────────┴────────────────────┴──────────────────────┴────────────────────────┴────────┴──────────────────────────────┴────────────┴──────────┴────────────────────┴────────────────────────┴───────────────────────────┘

`, "package", "push-upgrade", "list", "-p", "033000000000001AAA", "-s", "Failed", "--show-push-migrations-only")
	if len(queries) == 0 || !strings.HasSuffix(queries[0], "FROM PackagePushRequest WHERE PackageVersion.MetadataPackageId = '033000000000001AAA' AND Status = 'Failed' AND IsMigration = true") {
		t.Errorf("list queries = %q", queries)
	}
	_, errOut, code = run("package", "push-upgrade", "list", "-p", "033000000000001AAA", "-l", "0")
	if code != 1 || errOut != "Error (1): Invalid value for --scheduled-last-days. It must be a positive integer.\n\n\n" {
		t.Errorf("days: status %d, stderr %q", code, errOut)
	}

	out, errOut, code = run("package", "push-upgrade", "report", "-i", "0DV000000000001AAA")
	for _, want := range []string{"│ Package Name             │ Billing ", "│ Package Version          │ 1.2 ", "│ Namespace                │ acme ", "│ Duration Seconds         │ 7 ",
		"│ # Orgs Scheduled         │ 3 ", "│ # Orgs Upgrade Succeeded │ 2 ", "│ # Orgs Upgrade Failed    │ 1 ", "\n=== Errors\n"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
	if code != 0 || !strings.Contains(errOut, "Warning: (1) Apex compile failure") {
		t.Errorf("report: status %d, stderr %q", code, errOut)
	}
	out, errOut, code = run("package", "push-upgrade", "report", "-i", "0DV000000000009AAA", "--json")
	if code != 0 || !strings.Contains(out, `"result": null`) || !strings.Contains(out, `"No results found"`) {
		t.Errorf("missing report: status %d, stderr %q\n%s", code, errOut, out)
	}

	_, errOut, code = run("package", "push-upgrade", "abort", "-i", "0DV000000000001AAA")
	if code != 1 || !strings.Contains(errOut, "Error (1): Can’t abort package push upgrade request with status 'Failed'.") || patched != "" {
		t.Errorf("abort of a failed request: status %d, stderr %q, request %q", code, errOut, patched)
	}
	status = "Pending"
	expectRun(t, 0, "Scheduled push upgrade ID [0DV000000000001AAA] was cancelled.\n", "package", "push-upgrade", "abort", "-i", "0DV000000000001AAA")
	if patched != `PATCH {"Status":"Canceled"}` {
		t.Errorf("abort request = %q", patched)
	}
	_, errOut, code = run("package", "push-upgrade", "abort", "-i", "0DV000000000009AAA")
	if code != 1 || !strings.Contains(errOut, "The specified push upgrade ID isn’t valid.") {
		t.Errorf("abort of a missing request: status %d, stderr %q", code, errOut)
	}
	if _, errOut, code = run("package", "push-upgrade", "abort", "-i", "033000000000001AAA"); code == 0 || !strings.Contains(errOut, "The id must begin with 0DV.") {
		t.Errorf("abort with another id: status %d, stderr %q", code, errOut)
	}
}
