package sf

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// The layouts and messages in this file follow the sf 2.138.6 plugin
// source.  They were not compared with sf against a Dev Hub.

func TestPackageInstallAndUninstallReports(t *testing.T) {
	org := devHubSetup(t)
	org.handlers["/services/data/v67.0/tooling/sobjects/PackageInstallRequest/0Hf000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"Id":"0Hf000000000001AAA","Status":"SUCCESS","SubscriberPackageVersionKey":"04t000000000001AAA","Errors":null}`)
	}
	org.handlers["/services/data/v67.0/tooling/sobjects/SubscriberPackageVersionUninstallRequest/06y000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"Id":"06y000000000001AAA","Status":"InProgress","SubscriberPackageVersionId":"04t000000000001AAA"}`)
	}
	expectRun(t, 0, "Successfully installed package [04t000000000001AAA]\n", "package", "install", "report", "-i", "0Hf000000000001AAA")
	expectRun(t, 0, "PackageUninstallRequest is currently InProgress.\nYou can continue to query the status using sf package uninstall report -i 06y000000000001AAA -o me@example.com\n",
		"package", "uninstall", "report", "--request-id", "06y000000000001AAA")
	out, _, status := run("force:package:install:report", "-i", "0Hf000000000001AAA", "--json")
	if status != 0 || !strings.Contains(out, `"Status": "SUCCESS"`) {
		t.Errorf("json: status %d\n%s", status, out)
	}
}

func TestPackageUpdateAndDelete(t *testing.T) {
	org := devHubSetup(t)
	var packagePatch, versionPatch string
	org.handlers["/services/data/v67.0/tooling/sobjects/Package2/0Ho000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		packagePatch = string(body)
		w.WriteHeader(http.StatusNoContent)
	}
	org.handlers["/services/data/v67.0/tooling/sobjects/Package2Version/05i000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		versionPatch = string(body)
		w.WriteHeader(http.StatusNoContent)
	}
	org.handlers["/services/data/v67.0/tooling/query"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"done":true,"totalSize":1,"records":[`+packageVersionRecord+`]}`)
	}

	expectRun(t, 0, "Successfully updated the package. 0Ho000000000001AAA\n", "package", "update", "-p", "billing", "-n", "Billing 2", "-d", "")
	if packagePatch != `{"Name":"Billing 2","Description":""}` {
		t.Errorf("package update = %s", packagePatch)
	}
	expectRun(t, 0, "Successfully deleted the package with ID: 0Ho000000000001AAA\n", "package", "delete", "-p", "billing", "--no-prompt")
	if packagePatch != `{"IsDeprecated":true}` {
		t.Errorf("package delete = %s", packagePatch)
	}
	_, errOut, status := run("package", "delete", "-p", "billing")
	if status != 1 || !strings.Contains(errOut, "Deleted packages can't be recovered.") || !strings.Contains(errOut, "The request to delete this package was canceled") {
		t.Errorf("declined delete: status %d, stderr %q", status, errOut)
	}

	expectRun(t, 0, "Successfully updated the package version. 05i000000000001AAA\n", "package", "version", "update", "-p", "billing@1.0.0-1", "-a", "Summer", "-t", "rc1")
	if versionPatch != `{"Name":"Summer","Tag":"rc1"}` {
		t.Errorf("version update = %s", versionPatch)
	}
	expectRun(t, 0, `{
  "status": 0,
  "result": {
    "id": "05i000000000001AAA",
    "success": true,
    "errors": []
  },
  "warnings": []
}
`, "package", "version", "delete", "-p", "04t000000000001AAA", "--json")
	if versionPatch != `{"IsDeprecated":true}` {
		t.Errorf("version delete = %s", versionPatch)
	}
}

func TestPackageVersionCreateRequests(t *testing.T) {
	org := devHubSetup(t)
	var soql string
	org.handlers["/services/data/v67.0/tooling/query"] = func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if strings.Contains(query, "Package2VersionCreateRequestError") {
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"Message":"Apex compile failure"}]}`)
			return
		}
		soql = query
		if strings.Contains(query, "Status = 'Error'") || strings.Contains(query, "08c000000000002AAA") {
			io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"Id":"08c000000000002AAA","Status":"Error","Package2Id":"0Ho000000000001AAA","Package2":{"Name":"Billing"},"Package2VersionId":null,"Package2Version":null,"Tag":null,"Branch":null,"CreatedDate":"2026-01-02T03:04:05.000+0000","CreatedById":"005000000000001AAA"}]}`)
			return
		}
		io.WriteString(w, `{"done":true,"totalSize":1,"records":[{"Id":"08c000000000001AAA","Status":"Success","Package2Id":"0Ho000000000001AAA","Package2":{"Name":"Billing"},"Package2VersionId":"05i000000000001AAA","Package2Version":{"SubscriberPackageVersionId":"04t000000000001AAA","HasMetadataRemoved":false},"Tag":"rc1","Branch":null,"CreatedDate":"2026-01-02T03:04:05.000+0000","CreatedById":"005000000000001AAA"}]}`)
	}

	expectRun(t, 0, tableText(t, "Package Version Create Request", [][]string{
		{"ID", "08c000000000001AAA"}, {"Status", "Success"}, {"Package Id", "0Ho000000000001AAA"}, {"Package Version Id", "05i000000000001AAA"},
		{"Subscriber Package Version Id", "04t000000000001AAA"}, {"Tag", "rc1"}, {"Branch", ""}, {"Created Date", "2026-01-02 03:04"},
		{"Installation URL", "https://login.salesforce.com/packaging/installPackage.apexp?p0=04t000000000001AAA"}, {"Created By", "005000000000001AAA"},
	}), "package", "version", "create", "report", "-i", "08c000000000001AAA")

	out, errOut, status := run("package", "version", "create", "report", "-i", "08c000000000002AAA")
	if status != 0 || !strings.Contains(out, "│ Status                        │ Error ") || !strings.Contains(out, "\n=== Errors\n") || !strings.Contains(errOut, "Warning: (1) Apex compile failure") {
		t.Errorf("error report: status %d, stderr %q\n%s", status, errOut, out)
	}
	out, _, _ = run("package", "version", "create", "report", "-i", "08c000000000002AAA", "--json")
	if !strings.Contains(out, "\"Error\": [\n        \"Apex compile failure\"\n      ]") || !strings.Contains(out, `"Package2Name": "Billing"`) {
		t.Errorf("error report json:\n%s", out)
	}

	out, _, status = run("package", "version", "create", "list", "-c", "7")
	if status != 0 || !strings.HasPrefix(out, "Package Version Create Requests  [1]\n") ||
		!strings.Contains(out, "│ Id                 │ Status  │ Package Id         │ Package Version Id │ Subscriber Package Version Id │ Tag │ Branch │ Created Date     │ Created By         │") {
		t.Errorf("list: status %d\n%s", status, out)
	}
	if !strings.Contains(soql, "CreatedDate = LAST_N_DAYS:7") {
		t.Errorf("list query = %s", soql)
	}
	if _, _, status := run("package", "version", "create", "list", "-s", "Error", "--json"); status != 0 || !strings.Contains(soql, "Status = 'Error'") {
		t.Errorf("status filter: status %d, query %s", status, soql)
	}
}

// tableText returns the output of printTable for a Name/Value table.
func tableText(t *testing.T, title string, rows [][]string) string {
	t.Helper()
	var b strings.Builder
	printTable(&b, title, []string{"Name", "Value"}, rows)
	return b.String()
}

func TestApiRequestGraphql(t *testing.T) {
	org := loggedInFakeOrg(t)
	var payload string
	response := `{"data":{"uiapi":{"query":{"Account":{"edges":[]}}}},"errors":[]}`
	org.handlers["/services/data/v67.0/graphql"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		payload = string(body)
		io.WriteString(w, response)
	}
	out, errOut, status := run("api", "request", "graphql", "--body", "query { uiapi { query { Account { edges { node { Id } } } } } }")
	if status != 0 || !strings.HasPrefix(out, "{\n  \"data\": {\n    \"uiapi\": {") || !strings.HasPrefix(errOut, "Warning: This command is currently in beta.") {
		t.Errorf("status %d, stdout %s, stderr %q", status, out, errOut)
	}
	if payload != `{"query":"query { uiapi { query { Account { edges { node { Id } } } } } }"}` {
		t.Errorf("payload = %s", payload)
	}
	response = `{"data":null,"errors":[{"message":"Validation error"}]}`
	if _, _, status := run("api", "request", "graphql", "--body", "query { nope }"); status != 1 {
		t.Errorf("a response with errors exits with status %d, want 1", status)
	}
}
