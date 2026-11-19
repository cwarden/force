package sf

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestPackageCreateOptions(t *testing.T) {
	org := devHubSetup(t)
	var request map[string]interface{}
	org.handlers["/services/data/v67.0/tooling/sobjects/Package2"] = func(w http.ResponseWriter, r *http.Request) {
		request = nil
		json.NewDecoder(r.Body).Decode(&request)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"0Ho000000000005AAA","success":true,"errors":[]}`)
	}
	if _, errOut, status := run("package", "create", "-n", "Tools", "-t", "Unlocked", "-r", "force-app", "--no-namespace", "--org-dependent", "-o", "admin@example.com"); status != 0 {
		t.Fatalf("create: status %d, stderr %q", status, errOut)
	}
	if request["NamespacePrefix"] != "" || request["IsOrgDependent"] != true || request["PackageErrorUsername"] != "admin@example.com" {
		t.Errorf("request = %v", request)
	}
	// Without the flags the namespace of the project is used and the
	// other two fields are not sent.
	run("package", "create", "-n", "Tools2", "-t", "Unlocked", "-r", "force-app")
	if _, sent := request["IsOrgDependent"]; sent || request["NamespacePrefix"] != "acme" {
		t.Errorf("request without the flags = %v", request)
	}
}

func TestPackageInstallSkipHandlers(t *testing.T) {
	org := devHubSetup(t)
	var request map[string]interface{}
	org.handlers["/services/data/v67.0/tooling/sobjects/PackageInstallRequest"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&request)
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"0Hf000000000001AAA","success":true,"errors":[]}`)
	}
	org.handlers["/services/data/v67.0/tooling/sobjects/PackageInstallRequest/0Hf000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"Id":"0Hf000000000001AAA","Status":"IN_PROGRESS","SubscriberPackageVersionKey":"04t000000000001AAA"}`)
	}
	if _, errOut, status := run("package", "install", "-p", "04t000000000001AAA", "-l", "FeatureEnforcement", "-o", "hub"); status != 0 || request["SkipHandlers"] != "FeatureEnforcement" {
		t.Errorf("install: status %d, stderr %q, request %v", status, errOut, request)
	}
	if _, errOut, status := run("package", "install", "-p", "04t000000000001AAA", "-l", "Everything", "-o", "hub"); status == 0 || !strings.Contains(errOut, "Expected --skip-handlers=Everything to be one of: FeatureEnforcement") {
		t.Errorf("other handler: status %d, stderr %q", status, errOut)
	}
}

func TestPackageUpdateRecommendedVersionAncestorCheck(t *testing.T) {
	org := devHubSetup(t)
	prior := `"04t000000000001AAA"`
	// 1 is the ancestor of 2; 3 has no ancestor.
	versions := []string{
		`{"SubscriberPackageVersionId":"04t000000000001AAA","AncestorId":null}`,
		`{"SubscriberPackageVersionId":"04t000000000002AAA","AncestorId":"04t000000000001AAA"}`,
		`{"SubscriberPackageVersionId":"04t000000000003AAA","AncestorId":null}`,
	}
	org.handlers["/services/data/v67.0/tooling/query"] = func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		var records []string
		switch {
		case strings.HasPrefix(query, "SELECT RecommendedVersionId FROM Package2"):
			records = []string{`{"RecommendedVersionId":` + prior + `}`}
		case strings.HasPrefix(query, "SELECT SubscriberPackageVersionId, AncestorId FROM Package2Version"):
			records = versions
		}
		io.WriteString(w, fmt.Sprintf(`{"done":true,"totalSize":%d,"records":[%s]}`, len(records), strings.Join(records, ",")))
	}
	var patches []string
	org.handlers["/services/data/v67.0/tooling/sobjects/Package2/0Ho000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		patches = append(patches, string(body))
		w.WriteHeader(http.StatusNoContent)
	}
	update := func(args ...string) (string, int) {
		_, errOut, status := run(append([]string{"package", "update", "-p", "billing"}, args...)...)
		return errOut, status
	}
	if errOut, status := update("-r", "04t000000000002AAA"); status != 0 || len(patches) != 1 {
		t.Errorf("descendant: status %d, stderr %q, patches %q", status, errOut, patches)
	}
	if errOut, status := update("-r", "04t000000000003AAA"); status != 1 || !strings.Contains(errOut, "Error (RecommendedVersionNotAncestorOfPriorVersionError): The new recommended version is not a descendant of the previous recommended version.") || len(patches) != 1 {
		t.Errorf("unrelated version: status %d, stderr %q, patches %q", status, errOut, patches)
	}
	if errOut, status := update("-r", "04t000000000003AAA", "--skip-ancestor-check"); status != 0 || len(patches) != 2 {
		t.Errorf("skipped check: status %d, stderr %q, patches %q", status, errOut, patches)
	}
	if errOut, status := update("-r", "04t000000000009AAA"); status != 1 || !strings.Contains(errOut, "Error (UnassociatedRecommendedVersionError): The provided recommended version isn't associated with this package.") {
		t.Errorf("version of another package: status %d, stderr %q", status, errOut)
	}
	if errOut, status := update("--skip-ancestor-check", "-n", "Billing"); status != 1 || !strings.Contains(errOut, "Error (SkipAncestorCheckRequiresRecommendedVersionIdError): The skip ancestor check requires a recommended version ID.") {
		t.Errorf("skip without a version: status %d, stderr %q", status, errOut)
	}
	if errOut, status := update("-r", "05i000000000001AAA"); status != 1 || !strings.Contains(errOut, "Provide a valid subscriber package version (04t) for the recommended version.") {
		t.Errorf("other id: status %d, stderr %q", status, errOut)
	}
	// A package without a recommended version accepts any version.
	prior = "null"
	if errOut, status := update("-r", "04t000000000003AAA"); status != 0 || len(patches) != 3 {
		t.Errorf("first recommended version: status %d, stderr %q", status, errOut)
	}
}

func TestPackageVersionListsShowConversions(t *testing.T) {
	org := devHubSetup(t)
	var queries []string
	org.handlers["/services/data/v67.0/tooling/query"] = func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		queries = append(queries, query)
		var records []string
		switch {
		case strings.Contains(query, "FROM Package2VersionCreateRequest"):
			records = []string{`{"Id":"08c000000000001AAA","Status":"Success","Package2Id":"0Ho000000000001AAA","Package2":{"Name":"Billing"},"Package2VersionId":"05i000000000001AAA","Package2Version":{"SubscriberPackageVersionId":"04t000000000001AAA","HasMetadataRemoved":false,"MajorVersion":1,"MinorVersion":2,"PatchVersion":0,"BuildNumber":3,"CodeCoverage":null,"HasPassedCodeCoverageCheck":false,"ConvertedFromVersionId":"04t000000000008AAA"},"Tag":null,"Branch":null,"CreatedDate":"2026-01-02T03:04:05.000+0000","CreatedById":"005000000000001AAA","IsConversionRequest":true}`}
		case strings.HasPrefix(query, "SELECT Id, Name, MajorVersion"):
			records = []string{`{"Id":"05i000000000001AAA","Name":"Spring","MajorVersion":1,"MinorVersion":2,"PatchVersion":0,"BuildNumber":3}`}
		case strings.Contains(query, "FROM Package2Version WHERE"):
			records = []string{strings.Replace(packageVersionRecord, `"ConvertedFromVersionId":null`, `"ConvertedFromVersionId":"04t000000000008AAA"`, 1)}
		}
		io.WriteString(w, fmt.Sprintf(`{"done":true,"totalSize":%d,"records":[%s]}`, len(records), strings.Join(records, ",")))
	}

	out, _, status := run("package", "version", "create", "list", "--show-conversions-only", "--verbose")
	if status != 0 || !strings.Contains(queries[0], "WHERE IsConversionRequest = true  ORDER BY CreatedDate desc") ||
		!strings.Contains(out, "│ Created By         │ Converted From Version Id │ Version Name │ Version Number │") ||
		!strings.Contains(out, "│ 005000000000001AAA │ 04t000000000008AAA        │ Spring       │ 1.2.0.3        │") {
		t.Errorf("create list: status %d, queries %q\n%s", status, queries, out)
	}
	out, _, _ = run("package", "version", "create", "list", "--verbose", "--json")
	if !strings.Contains(out, `"VersionName": "Spring"`) {
		t.Errorf("create list json:\n%s", out)
	}

	queries = nil
	out, _, status = run("package", "version", "list", "--show-conversions-only")
	if status != 0 || !strings.Contains(queries[0], "ConvertedFromVersionId != null") || !strings.Contains(out, "│ Branch │ Converted From Version Id │") || !strings.Contains(out, "│ 04t000000000008AAA        │") {
		t.Errorf("version list: status %d, queries %q\n%s", status, queries, out)
	}
}
