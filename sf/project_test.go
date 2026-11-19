package sf

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// The convert output, the queued deploy output and the progress display
// layout in this file were captured from sf 2.138.6.  The tables printed
// after a finished deploy or retrieve follow the sf plugin source.

const greeterMeta = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ApexClass xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <apiVersion>67.0</apiVersion>\n    <status>Active</status>\n</ApexClass>\n"

// newSourceProject creates a project with two Apex classes and a custom
// object with one field, in source format, and makes it the working
// directory.
func newSourceProject(t *testing.T, home string) string {
	t.Helper()
	project := newProject(t, home)
	files := map[string]string{
		"force-app/main/default/classes/Greeter.cls":                                "public class Greeter {}",
		"force-app/main/default/classes/Greeter.cls-meta.xml":                       greeterMeta,
		"force-app/main/default/classes/GreeterTest.cls":                            "@IsTest private class GreeterTest {}",
		"force-app/main/default/classes/GreeterTest.cls-meta.xml":                   greeterMeta,
		"force-app/main/default/objects/Invoice__c/Invoice__c.object-meta.xml":      "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<CustomObject xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <label>Invoice</label>\n    <pluralLabel>Invoices</pluralLabel>\n</CustomObject>\n",
		"force-app/main/default/objects/Invoice__c/fields/Amount__c.field-meta.xml": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<CustomField xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <fullName>Amount__c</fullName>\n    <label>Amount</label>\n    <type>Number</type>\n</CustomField>\n",
	}
	for name, content := range files {
		path := filepath.Join(project, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(project, "sfdx-project.json"), []byte(`{"packageDirectories":[{"path":"force-app","default":true}],"sourceApiVersion":"67.0"}`), 0o644)
	return project
}

func TestProjectConvertSourceWritesMetadataFormat(t *testing.T) {
	home := useTempHome(t)
	newSourceProject(t, home)
	out := filepath.Join(home, "mdapi_out")
	expectRun(t, 0, "Source was successfully converted to Metadata API format and written to the location: "+out+"\n",
		"project", "convert", "source", "-r", "force-app", "-d", out)

	manifest, err := os.ReadFile(filepath.Join(out, "package.xml"))
	if err != nil {
		t.Fatal(err)
	}
	want := `<?xml version="1.0" encoding="UTF-8"?>
<Package xmlns="http://soap.sforce.com/2006/04/metadata">
    <types>
        <members>Greeter</members>
        <members>GreeterTest</members>
        <name>ApexClass</name>
    </types>
    <types>
        <members>Invoice__c.Amount__c</members>
        <name>CustomField</name>
    </types>
    <types>
        <members>Invoice__c</members>
        <name>CustomObject</name>
    </types>
    <version>67.0</version>
</Package>
`
	if string(manifest) != want {
		t.Errorf("package.xml =\n%s\nwant\n%s", manifest, want)
	}
	for _, name := range []string{"classes/Greeter.cls", "classes/Greeter.cls-meta.xml", "classes/GreeterTest.cls"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Errorf("%s is missing: %v", name, err)
		}
	}
	// The object and its field are composed into one metadata format file.
	object, err := os.ReadFile(filepath.Join(out, "objects", "Invoice__c.object"))
	if err != nil || !strings.Contains(string(object), "<fullName>Amount__c</fullName>") || !strings.Contains(string(object), "<label>Invoice</label>") {
		t.Errorf("composed object = %q, err = %v", object, err)
	}

	expectRun(t, 0, `{
  "status": 0,
  "result": {
    "location": "`+filepath.Join(home, "one")+`"
  },
  "warnings": [
    "The \"force source convert\" command has been deprecated. Use \"project convert source\" instead."
  ]
}
`, "force:source:convert", "--metadata", "ApexClass:Greeter", "--output-dir", filepath.Join(home, "one"), "--json")
	if _, err := os.Stat(filepath.Join(home, "one", "classes", "GreeterTest.cls")); !os.IsNotExist(err) {
		t.Errorf("--metadata ApexClass:Greeter also converted GreeterTest: %v", err)
	}
}

func TestProjectConvertMdapiWritesSourceFormat(t *testing.T) {
	home := useTempHome(t)
	newSourceProject(t, home)
	mdapi := filepath.Join(home, "mdapi_out")
	if _, errOut, status := run("project", "convert", "source", "-m", "ApexClass", "-d", mdapi); status != 0 {
		t.Fatalf("convert source: status %d, %s", status, errOut)
	}
	out, errOut, status := run("project", "convert", "mdapi", "-r", mdapi, "-d", filepath.Join(home, "src_out"))
	want := `┌───────┬─────────────┬───────────┬──────────────────────────────────────────────────────────┐
│ STATE │ FULL NAME   │ TYPE      │ PROJECT PATH                                             │
├───────┼─────────────┼───────────┼──────────────────────────────────────────────────────────┤
│ Add   │ Greeter     │ ApexClass │ ../src_out/main/default/classes/Greeter.cls-meta.xml     │
│ Add   │ Greeter     │ ApexClass │ ../src_out/main/default/classes/Greeter.cls              │
│ Add   │ GreeterTest │ ApexClass │ ../src_out/main/default/classes/GreeterTest.cls-meta.xml │
│ Add   │ GreeterTest │ ApexClass │ ../src_out/main/default/classes/GreeterTest.cls          │
└───────┴─────────────┴───────────┴──────────────────────────────────────────────────────────┘

`
	if status != 0 || out != want {
		t.Errorf("status %d, stdout\n%s\nwant\n%s", status, out, want)
	}
	if errOut != "Converting 2 metadata components... done\n" {
		t.Errorf("stderr = %q", errOut)
	}
	out, _, _ = run("project", "convert", "mdapi", "--root-dir", mdapi, "--output-dir", filepath.Join(home, "src_out"), "--json")
	if !strings.Contains(out, "\"filePath\": \"../src_out/main/default/classes/Greeter.cls-meta.xml\",\n      \"state\": \"Unchanged\"") {
		t.Errorf("a second conversion does not report unchanged files:\n%s", out)
	}
	_, errOut, status = run("project", "convert", "mdapi", "-r", filepath.Join(home, "nosuch"))
	if status != 1 || !strings.Contains(errOut, "No such file or directory.") {
		t.Errorf("missing root dir: status %d, stderr %q", status, errOut)
	}
}

// fakeDeploy makes the fake org answer Metadata API deploy and retrieve
// requests.
type fakeDeploy struct {
	requests     []string
	deployed     map[string]string // files of the last deployed zip
	options      string
	statusResult string // content of <result> for checkDeployStatus
	retrieveZip  []byte
}

func soapBody(content string) string {
	return strings.Replace(soapEnvelope, "%s", content, 1)
}

var soapOperationPattern = regexp.MustCompile(`(?s)<(?:[A-Za-z0-9]+:)?Body[^>]*>\s*<(?:[A-Za-z0-9]+:)?([A-Za-z]+)`)

var zipFilePattern = regexp.MustCompile(`(?s)<[A-Za-z:]*[zZ]ipFile>(.*?)</[A-Za-z:]*[zZ]ipFile>`)

func serveMetadata(org *fakeOrg) *fakeDeploy {
	fake := &fakeDeploy{}
	org.handlers["/services/Soap/m/67.0"] = func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body := string(data)
		w.Header().Set("Content-Type", "text/xml")
		// The operation is the first element of the SOAP body.
		operation := ""
		if match := soapOperationPattern.FindStringSubmatch(body); match != nil {
			operation = match[1]
		}
		is := func(name string) bool { return operation == name }
		switch {
		case is("checkStatus"):
			// lib waits for a retrieve with this operation.
			fake.requests = append(fake.requests, "checkStatus")
			io.WriteString(w, soapBody("<checkStatusResponse><result><done>true</done><id>09S000000000001AAA</id><state>Completed</state></result></checkStatusResponse>"))
		case is("checkDeployStatus"):
			fake.requests = append(fake.requests, "checkDeployStatus")
			io.WriteString(w, soapBody("<checkDeployStatusResponse><result>"+fake.statusResult+"</result></checkDeployStatusResponse>"))
		case is("deployRecentValidation"):
			fake.requests = append(fake.requests, "deployRecentValidation")
			io.WriteString(w, soapBody("<deployRecentValidationResponse><result>0Af000000000002AAA</result></deployRecentValidationResponse>"))
		case is("cancelDeploy"):
			fake.requests = append(fake.requests, "cancelDeploy")
			io.WriteString(w, soapBody("<cancelDeployResponse><result><done>false</done><id>0Af000000000001AAA</id></result></cancelDeployResponse>"))
		case is("checkRetrieveStatus"):
			fake.requests = append(fake.requests, "checkRetrieveStatus")
			io.WriteString(w, soapBody("<checkRetrieveStatusResponse><result><done>true</done><id>09S000000000001AAA</id><status>Succeeded</status><success>true</success><zipFile>"+base64.StdEncoding.EncodeToString(fake.retrieveZip)+"</zipFile></result></checkRetrieveStatusResponse>"))
		case is("retrieve"):
			fake.requests = append(fake.requests, "retrieve")
			fake.options = body
			io.WriteString(w, soapBody("<retrieveResponse><result><done>false</done><id>09S000000000001AAA</id><state>Queued</state></result></retrieveResponse>"))
		case is("deploy"):
			fake.requests = append(fake.requests, "deploy")
			fake.options = body
			fake.deployed = map[string]string{}
			if match := zipFilePattern.FindStringSubmatch(body); match != nil {
				if archive, err := base64.StdEncoding.DecodeString(strings.TrimSpace(match[1])); err == nil {
					if reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive))); err == nil {
						for _, file := range reader.File {
							content, _ := file.Open()
							data, _ := io.ReadAll(content)
							content.Close()
							fake.deployed[file.Name] = string(data)
						}
					}
				}
			}
			io.WriteString(w, soapBody("<deployResponse><result><done>false</done><id>0Af000000000001AAA</id><state>Queued</state></result></deployResponse>"))
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	return fake
}

const deploySucceeded = `<checkOnly>false</checkOnly><completedDate>2026-01-02T03:04:06.000Z</completedDate><createdBy>005000000000001</createdBy><createdByName>Ada Lovelace</createdByName><createdDate>2026-01-02T03:04:05.000Z</createdDate>` +
	`<details>` +
	`<componentSuccesses><changed>false</changed><componentType>ApexClass</componentType><created>true</created><createdDate>2026-01-02T03:04:06.000Z</createdDate><deleted>false</deleted><fileName>classes/Greeter.cls</fileName><fullName>Greeter</fullName><id>01p000000000001AAA</id><success>true</success></componentSuccesses>` +
	`<componentSuccesses><changed>true</changed><componentType></componentType><created>false</created><deleted>false</deleted><fileName>package.xml</fileName><fullName>package.xml</fullName><success>true</success></componentSuccesses>` +
	`<runTestResult><numFailures>0</numFailures><numTestsRun>0</numTestsRun><totalTime>0.0</totalTime></runTestResult>` +
	`</details>` +
	`<done>true</done><id>0Af000000000001AAA</id><ignoreWarnings>false</ignoreWarnings><lastModifiedDate>2026-01-02T03:04:06.000Z</lastModifiedDate>` +
	`<numberComponentErrors>0</numberComponentErrors><numberComponentsDeployed>1</numberComponentsDeployed><numberComponentsTotal>1</numberComponentsTotal>` +
	`<numberTestErrors>0</numberTestErrors><numberTestsCompleted>0</numberTestsCompleted><numberTestsTotal>0</numberTestsTotal>` +
	`<rollbackOnError>true</rollbackOnError><runTestsEnabled>false</runTestsEnabled><status>Succeeded</status><success>true</success>`

const deployFailed = `<checkOnly>false</checkOnly>` +
	`<details><componentFailures><changed>false</changed><columnNumber>12</columnNumber><componentType>ApexClass</componentType><created>false</created><deleted>false</deleted><fileName>classes/Greeter.cls</fileName><fullName>Greeter</fullName><lineNumber>3</lineNumber><problem>Unexpected token '}'.</problem><problemType>Error</problemType><success>false</success></componentFailures></details>` +
	`<done>true</done><id>0Af000000000001AAA</id><numberComponentErrors>1</numberComponentErrors><numberComponentsDeployed>0</numberComponentsDeployed><numberComponentsTotal>1</numberComponentsTotal>` +
	`<numberTestErrors>0</numberTestErrors><numberTestsCompleted>0</numberTestsCompleted><numberTestsTotal>0</numberTestsTotal><status>Failed</status><success>false</success>`

func deploySetup(t *testing.T) (*fakeOrg, *fakeDeploy, string) {
	t.Helper()
	org := loggedInFakeOrg(t)
	fake := serveMetadata(org)
	fake.statusResult = deploySucceeded
	project := newSourceProject(t, os.Getenv("HOME"))
	previous := deployPollInterval
	deployPollInterval = time.Millisecond
	t.Cleanup(func() { deployPollInterval = previous })
	return org, fake, project
}

var durationPattern = regexp.MustCompile(`\d+(\.\d+)?m?s`)

func TestProjectDeployStartFromSourceDirectory(t *testing.T) {
	_, fake, project := deploySetup(t)
	out, errOut, status := run("project", "deploy", "start", "-d", "force-app/main/default/classes", "-w", "5")
	if status != 0 {
		t.Fatalf("status %d, stderr %s\n%s", status, errOut, out)
	}
	want := `
 ─────────────── Deploying Metadata ───────────────

 Deploying v67.0 metadata to me@example.com using the v67.0 SOAP API.

 ✔ Preparing T
 ◯ Waiting for the org to respond - Skipped
 ✔ Deploying Metadata T
   ▸ Components: 1/1 (100%)
 ◯ Running Tests - Skipped
 ◯ Updating Source Tracking - Skipped
 ✔ Done T

 Status: Succeeded
 Deploy ID: 0Af000000000001AAA
 Target Org: me@example.com
 Elapsed Time: T


Deployed Source
┌─────────┬─────────┬───────────┬─────────────────────────────────────────────────────┐
│ State   │ Name    │ Type      │ Path                                                │
├─────────┼─────────┼───────────┼─────────────────────────────────────────────────────┤
│ Created │ Greeter │ ApexClass │ force-app/main/default/classes/Greeter.cls          │
│ Created │ Greeter │ ApexClass │ force-app/main/default/classes/Greeter.cls-meta.xml │
└─────────┴─────────┴───────────┴─────────────────────────────────────────────────────┘

`
	if got := durationPattern.ReplaceAllString(out, "T"); got != want {
		t.Errorf("output =\n%s\nwant\n%s", got, want)
	}

	// The zip sent to the org is in metadata format with a manifest.
	for _, name := range []string{"classes/Greeter.cls", "classes/Greeter.cls-meta.xml", "classes/GreeterTest.cls", "package.xml"} {
		if _, ok := fake.deployed[name]; !ok {
			t.Errorf("the deployed zip lacks %s; it has %v", name, keys(fake.deployed))
		}
	}
	if _, ok := fake.deployed["objects/Invoice__c.object"]; ok {
		t.Error("deploying the classes directory also deployed the object")
	}
	if !strings.Contains(fake.deployed["package.xml"], "<members>Greeter</members>") || !strings.Contains(fake.deployed["package.xml"], "<version>67.0</version>") {
		t.Errorf("package.xml = %s", fake.deployed["package.xml"])
	}
	for _, option := range []string{"<checkOnly>false</checkOnly>", "<rollbackOnError>true</rollbackOnError>", "<singlePackage>true</singlePackage>"} {
		if !strings.Contains(fake.options, option) {
			t.Errorf("deploy options lack %s", option)
		}
	}

	out, _, status = run("project", "deploy", "start", "--source-dir", "force-app", "--json")
	var env struct {
		Status int
		Result map[string]interface{}
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	result := env.Result
	if status != 0 || result["status"] != "Succeeded" || result["done"] != true || result["checkOnly"] != false ||
		result["numberComponentsDeployed"] != float64(1) || result["createdByName"] != "Ada Lovelace" {
		t.Errorf("status %d, result %v", status, result)
	}
	details := result["details"].(map[string]interface{})
	if len(details["componentSuccesses"].([]interface{})) != 2 || len(details["componentFailures"].([]interface{})) != 0 {
		t.Errorf("details = %v", details)
	}
	files := result["files"].([]interface{})
	first := files[0].(map[string]interface{})
	if len(files) != 2 || first["filePath"] != filepath.Join(project, "force-app", "main", "default", "classes", "Greeter.cls") || first["state"] != "Created" {
		t.Errorf("files = %v", files)
	}
	if result["zipFileCount"] != float64(6) || !strings.Contains(result["deployUrl"].(string), "asyncId%3D0Af000000000001AAA") {
		t.Errorf("zipFileCount %v, deployUrl %v", result["zipFileCount"], result["deployUrl"])
	}
	if _, ok := fake.deployed["objects/Invoice__c.object"]; !ok {
		t.Errorf("deploying force-app does not include the composed object; zip has %v", keys(fake.deployed))
	}
}

func keys(m map[string]string) []string {
	var names []string
	for name := range m {
		names = append(names, name)
	}
	return names
}

func TestProjectDeployStartByMetadataManifestAndMetadataDir(t *testing.T) {
	home := os.Getenv("HOME")
	_, fake, _ := deploySetup(t)
	home = os.Getenv("HOME")

	if _, errOut, status := run("project", "deploy", "start", "-m", "ApexClass:GreeterTest", "--json"); status != 0 {
		t.Fatalf("--metadata: status %d %s", status, errOut)
	}
	if _, ok := fake.deployed["classes/GreeterTest.cls"]; !ok || len(fake.deployed) != 3 {
		t.Errorf("--metadata ApexClass:GreeterTest deployed %v", keys(fake.deployed))
	}

	manifest := filepath.Join(home, "package.xml")
	os.WriteFile(manifest, []byte(`<?xml version="1.0" encoding="UTF-8"?><Package xmlns="http://soap.sforce.com/2006/04/metadata"><types><members>*</members><name>ApexClass</name></types><version>67.0</version></Package>`), 0o644)
	if _, errOut, status := run("project", "deploy", "start", "--manifest", manifest, "--json"); status != 0 {
		t.Fatalf("--manifest: status %d %s", status, errOut)
	}
	if len(fake.deployed) != 5 {
		t.Errorf("--manifest with ApexClass * deployed %v", keys(fake.deployed))
	}

	mdapi := filepath.Join(home, "mdapi")
	os.MkdirAll(filepath.Join(mdapi, "classes"), 0o755)
	os.WriteFile(filepath.Join(mdapi, "classes", "Other.cls"), []byte("public class Other {}"), 0o644)
	os.WriteFile(filepath.Join(mdapi, "package.xml"), []byte("<Package/>"), 0o644)
	out, _, status := run("project", "deploy", "start", "--metadata-dir", mdapi, "-l", "RunSpecifiedTests", "-t", "OtherTest", "--ignore-errors", "--ignore-warnings")
	if status != 0 || !strings.Contains(out, " Deploying <version specified in manifest> metadata to me@example.com using the\n  v67.0 SOAP API.\n") {
		t.Errorf("--metadata-dir: status %d\n%s", status, out)
	}
	if fake.deployed["classes/Other.cls"] != "public class Other {}" || fake.deployed["package.xml"] != "<Package/>" {
		t.Errorf("--metadata-dir deployed %v", fake.deployed)
	}
	for _, option := range []string{"<testLevel>RunSpecifiedTests</testLevel>", "<runTests>OtherTest</runTests>", "<rollbackOnError>false</rollbackOnError>", "<ignoreWarnings>true</ignoreWarnings>"} {
		if !strings.Contains(fake.options, option) {
			t.Errorf("deploy options lack %s", option)
		}
	}

	_, errOut, status := run("project", "deploy", "start")
	if status != 1 || !strings.HasPrefix(errOut, "Error (NothingToDeploy): No local changes to deploy.") {
		t.Errorf("nothing to deploy: status %d, stderr %q", status, errOut)
	}
	_, errOut, status = run("project", "deploy", "start", "-d", "force-app", "-l", "RunSpecifiedTests")
	if status != 1 || !strings.Contains(errOut, "You must specify tests using the --tests flag if the --test-level flag is set to RunSpecifiedTests.") {
		t.Errorf("no tests: status %d, stderr %q", status, errOut)
	}
}

func TestProjectDeployFailureListsComponentFailures(t *testing.T) {
	_, fake, _ := deploySetup(t)
	fake.statusResult = deployFailed
	out, _, status := run("project", "deploy", "start", "-d", "force-app")
	if status != 1 {
		t.Errorf("status = %d, want 1 for a failed deploy", status)
	}
	for _, want := range []string{
		" ✘ Deploying Metadata ", "   ▸ Components: 0/1 (0%)\n", " ◼ Done\n", " Status: Failed\n",
		"Component Failures [1]\n", "│ Type      │ Name    │ Problem               │ Line:Column │", "│ ApexClass │ Greeter │ Unexpected token '}'. │ 3:12        │",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Deployed Source") {
		t.Errorf("a failed deploy lists deployed source:\n%s", out)
	}
	out, _, status = run("project", "deploy", "start", "-d", "force-app", "--json")
	if status != 1 || !strings.HasPrefix(out, "{\n  \"status\": 1,\n  \"result\": {") || !strings.Contains(out, `"error": "Unexpected token '}'."`) ||
		!strings.Contains(out, `"lineNumber": 3`) || !strings.Contains(out, `"state": "Failed"`) {
		t.Errorf("json: status %d\n%s", status, out)
	}
}

func TestProjectDeployAsyncAndValidate(t *testing.T) {
	_, fake, _ := deploySetup(t)
	out, _, status := run("project", "deploy", "start", "-d", "force-app", "--async")
	want := `
 ─────────────── Deploying Metadata ───────────────

 Deploying v67.0 metadata to me@example.com using the v67.0 SOAP API.

 ✔ Preparing T
 ◯ Waiting for the org to respond - Skipped
 ◯ Deploying Metadata - Skipped
 ◯ Running Tests - Skipped
 ◯ Updating Source Tracking - Skipped
 ✔ Done T

 Status: Queued
 Deploy ID: 0Af000000000001AAA
 Target Org: me@example.com
 Elapsed Time: T

Deploy has been queued.

Run "sf project deploy resume --job-id 0Af000000000001AAA" to resume watching the deploy.
Run "sf project deploy report --job-id 0Af000000000001AAA" to get the latest status.
Run "sf project deploy cancel --job-id 0Af000000000001AAA" to cancel the deploy.
`
	if got := durationPattern.ReplaceAllString(out, "T"); status != 0 || got != want {
		t.Errorf("async: status %d\n%s\nwant\n%s", status, got, want)
	}
	for _, request := range fake.requests {
		if request == "checkDeployStatus" {
			t.Error("--async checked the deploy status")
		}
	}
	out, _, _ = run("project", "deploy", "start", "-d", "force-app", "--async", "--json")
	if !strings.Contains(out, "\"done\": false,\n    \"status\": \"Queued\",\n    \"files\": [],\n    \"zipSize\": ") {
		t.Errorf("async json:\n%s", out)
	}

	fake.statusResult = strings.Replace(deploySucceeded, "<checkOnly>false</checkOnly>", "<checkOnly>true</checkOnly>", 1)
	out, _, status = run("project", "deploy", "validate", "-d", "force-app")
	if status != 0 || !strings.HasPrefix(out, "Validating Deployment of v67.0 metadata to me@example.com using the v67.0 SOAP API.\n\n ───────────── Validating Deployment ─────────────\n\n\n ✔ Preparing") ||
		!strings.Contains(out, "\nValidated Source\n") ||
		!strings.HasSuffix(out, "Successfully validated the deployment (0Af000000000001AAA).\nRun \"sf project deploy quick --job-id 0Af000000000001AAA\" to execute this deploy\n") {
		t.Errorf("validate: status %d\n%s", status, out)
	}
	if !strings.Contains(fake.options, "<checkOnly>true</checkOnly>") || !strings.Contains(fake.options, "<testLevel>RunLocalTests</testLevel>") {
		t.Errorf("validate options = %s", fake.options)
	}
}

func TestProjectDeployQuickReportResumeAndCancel(t *testing.T) {
	_, fake, _ := deploySetup(t)
	out, _, status := run("project", "deploy", "quick", "-i", "0Af000000000001AAA")
	if status != 0 || !strings.HasSuffix(out, "Successfully deployed (0Af000000000001AAA).\n") || fake.requests[0] != "deployRecentValidation" {
		t.Errorf("quick: status %d\n%s", status, out)
	}

	out, _, status = run("project", "deploy", "report", "--job-id", "0Af000000000001AAA")
	if status != 0 || !strings.HasPrefix(out, "0Af000000000001AAA... Succeeded\n\nDeploy Info\n") || !strings.Contains(out, "│ numberComponentsDeployed │ 1 ") ||
		!strings.Contains(out, "\nDeploy Options\n") || !strings.Contains(out, "│ target-org │ me@example.com     │") {
		t.Errorf("report: status %d\n%s", status, out)
	}

	out, errOut, status := run("project", "deploy", "resume", "-i", "0Af000000000001AAA")
	if status != 0 || !strings.Contains(errOut, "Job ID 0Af000000000001AAA is not resumable because it already completed with status: Succeeded. Displaying results...") ||
		!strings.Contains(out, "Deployed Source") {
		t.Errorf("resume: status %d, stderr %q\n%s", status, errOut, out)
	}

	_, errOut, status = run("project", "deploy", "cancel", "-i", "0Af000000000001AAA")
	if status != 1 || !strings.Contains(errOut, "Can't cancel deploy with Job Id 0Af000000000001AAA because it's already completed (status is Succeeded)") {
		t.Errorf("cancel of a finished deploy: status %d, stderr %q", status, errOut)
	}
	fake.statusResult = `<done>false</done><id>0Af000000000001AAA</id><status>InProgress</status><success>false</success>`
	expectRun(t, 0, "Deploy has been queued for cancellation.\n\nRun \"sf project deploy resume --job-id 0Af000000000001AAA\" to resume watching the deploy.\nRun \"sf project deploy report --job-id 0Af000000000001AAA\" to get the latest status.\n",
		"project", "deploy", "cancel", "-i", "0Af000000000001AAA", "--async")

	_, errOut, status = run("project", "deploy", "report", "--use-most-recent")
	if status != 1 || !strings.Contains(errOut, "Error (NoRecentJobIdError): There are no recent job IDs available.") {
		t.Errorf("--use-most-recent with an empty cache: status %d, stderr %q", status, errOut)
	}
}

func TestDeployCacheSuppliesTheMostRecentJobAndItsOrg(t *testing.T) {
	_, fake, _ := deploySetup(t)
	if _, errOut, status := run("project", "deploy", "start", "-d", "force-app", "--async", "--json"); status != 0 {
		t.Fatalf("deploy: status %d %s", status, errOut)
	}
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".sf", "deploy-cache.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cache map[string]map[string]interface{}
	json.Unmarshal(data, &cache)
	entry := cache["0Af000000000001AAA"]
	if entry["target-org"] != "me@example.com" || entry["isMdapi"] != false || entry["timestamp"] == nil {
		t.Errorf("cache entry = %v", entry)
	}

	// The default org is removed, so the report finds the org in the
	// cache entry.
	if _, _, status := run("config", "unset", "target-org", "--global"); status != 0 {
		t.Fatal("config unset failed")
	}
	fake.requests = nil
	out, errOut, status := run("project", "deploy", "report", "--use-most-recent")
	if status != 0 || !strings.HasPrefix(out, "0Af000000000001AAA... Succeeded\n") {
		t.Errorf("report --use-most-recent: status %d, stderr %q\n%s", status, errOut, out)
	}
	out, _, status = run("project", "deploy", "resume", "-r", "--json")
	if status != 0 || !strings.Contains(out, `"id": "0Af000000000001AAA"`) {
		t.Errorf("resume -r: status %d\n%s", status, out)
	}
	_, errOut, status = run("project", "deploy", "report")
	if status != 2 || !strings.Contains(errOut, "Exactly one of the following must be provided: --use-most-recent, --job-id") {
		t.Errorf("no job: status %d, stderr %q", status, errOut)
	}
}

func retrieveZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create("unpackaged/" + name)
		if err != nil {
			t.Fatal(err)
		}
		entry.Write([]byte(content))
	}
	writer.Close()
	return buffer.Bytes()
}

func TestProjectRetrieveStartWritesSourceFormat(t *testing.T) {
	_, fake, project := deploySetup(t)
	fake.retrieveZip = retrieveZip(t, map[string]string{
		"classes/Remote.cls":          "public class Remote {}",
		"classes/Remote.cls-meta.xml": greeterMeta,
		"package.xml":                 "<Package/>",
	})
	out, errOut, status := run("project", "retrieve", "start", "-m", "ApexClass:Remote")
	if status != 0 {
		t.Fatalf("status %d, stderr %s\n%s", status, errOut, out)
	}
	want := `
 ────────────── Retrieving Metadata ──────────────

 Retrieving v67.0 metadata from me@example.com using the v67.0 SOAP API

 ✔ Preparing retrieve request T
 ✔ Sending request to org T
 ✔ Waiting for the org to respond T
 ✔ Done T

 Status: Succeeded
 Elapsed Time: T


Retrieved Source
┌─────────┬────────┬───────────┬────────────────────────────────────────────────────┐
│ State   │ Name   │ Type      │ Path                                               │
├─────────┼────────┼───────────┼────────────────────────────────────────────────────┤
│ Created │ Remote │ ApexClass │ force-app/main/default/classes/Remote.cls-meta.xml │
│ Created │ Remote │ ApexClass │ force-app/main/default/classes/Remote.cls          │
└─────────┴────────┴───────────┴────────────────────────────────────────────────────┘

`
	if got := durationPattern.ReplaceAllString(out, "T"); got != want {
		t.Errorf("output =\n%s\nwant\n%s", got, want)
	}
	if content, _ := os.ReadFile(filepath.Join(project, "force-app", "main", "default", "classes", "Remote.cls")); string(content) != "public class Remote {}" {
		t.Errorf("retrieved class = %q", content)
	}
	if !strings.Contains(fake.options, "<members>Remote</members>") || !strings.Contains(fake.options, "<name>ApexClass</name>") {
		t.Errorf("retrieve request = %s", fake.options)
	}

	out, _, status = run("project", "retrieve", "start", "--metadata", "ApexClass:Remote", "--json")
	if status != 0 || !strings.Contains(out, `"state": "Unchanged"`) ||
		!strings.Contains(out, `"filePath": "`+filepath.Join(project, "force-app", "main", "default", "classes", "Remote.cls")+`"`) {
		t.Errorf("json: status %d\n%s", status, out)
	}

	_, errOut, status = run("project", "retrieve", "start")
	if status != 1 || !strings.HasPrefix(errOut, "Error (NoSourceTrackingError): Unable to track changes in your source files.") {
		t.Errorf("no components: status %d, stderr %q", status, errOut)
	}
}

func TestProjectRetrieveStartToMetadataDirectory(t *testing.T) {
	_, fake, _ := deploySetup(t)
	fake.retrieveZip = retrieveZip(t, map[string]string{"classes/Remote.cls": "public class Remote {}", "package.xml": "<Package/>"})
	target := filepath.Join(os.Getenv("HOME"), "retrieved")
	manifest := filepath.Join(os.Getenv("HOME"), "package.xml")
	os.WriteFile(manifest, []byte(`<Package xmlns="http://soap.sforce.com/2006/04/metadata"><types><members>Remote</members><name>ApexClass</name></types><version>67.0</version></Package>`), 0o644)
	out, _, status := run("project", "retrieve", "start", "-x", manifest, "-t", target, "--unzip")
	if status != 0 || !strings.HasSuffix(out, "Wrote retrieve zip file to "+filepath.Join(target, "unpackaged.zip")+".\nExtracted unpackaged to "+filepath.Join(target, "unpackaged")+".\n") {
		t.Errorf("status %d\n%s", status, out)
	}
	if content, _ := os.ReadFile(filepath.Join(target, "unpackaged", "unpackaged", "classes", "Remote.cls")); string(content) != "public class Remote {}" {
		t.Errorf("extracted class = %q", content)
	}
	archive, err := zip.OpenReader(filepath.Join(target, "unpackaged.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	var names []string
	for _, file := range archive.File {
		names = append(names, file.Name)
	}
	if strings.Join(names, " ") != "unpackaged/classes/Remote.cls unpackaged/package.xml" {
		t.Errorf("zip entries = %v", names)
	}
}
