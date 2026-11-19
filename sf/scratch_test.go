package sf

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ForceCLI/force/lib"
)

// The layout and messages in this file follow the sf 2.138.6 org plugin
// source.  They were not compared with sf against a Dev Hub.

func TestScratchSettingsFiles(t *testing.T) {
	parsed, err := parseOrderedJSON([]byte(`{"lightningExperienceSettings":{"enableS1DesktopEnabled":true},"securitySettings":{"passwordPolicies":{"minimumPasswordLength":"10"},"sessionSettings":{"trustedIpRanges":[{"start":"1.1.1.1"},{"start":"2.2.2.2"}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	files := scratchSettingsFiles(parsed.(*orderedMap))
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<LightningExperienceSettings xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <enableS1DesktopEnabled>true</enableS1DesktopEnabled>\n</LightningExperienceSettings>\n"
	if got := string(files["settings/LightningExperience.settings"]); got != want {
		t.Errorf("LightningExperience.settings =\n%s\nwant\n%s", got, want)
	}
	security := string(files["settings/Security.settings"])
	if !strings.Contains(security, "    <passwordPolicies>\n        <minimumPasswordLength>10</minimumPasswordLength>\n    </passwordPolicies>\n") ||
		strings.Count(security, "<trustedIpRanges>") != 2 {
		t.Errorf("Security.settings =\n%s", security)
	}
	manifest := string(files["package.xml"])
	if !strings.Contains(manifest, "<members>LightningExperience</members>\n        <members>Security</members>\n        <name>Settings</name>") {
		t.Errorf("package.xml =\n%s", manifest)
	}
}

func TestOrgCreateScratchFromDefinitionFile(t *testing.T) {
	org := devHubSetup(t)
	var request map[string]interface{}
	org.handlers["/services/data/v67.0/sobjects/ScratchOrgInfo"] = func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&request)
		// The scratch org answers as a different user from here on.
		org.username = "test-scratch@example.com"
		w.WriteHeader(http.StatusCreated)
		io.WriteString(w, `{"id":"2SR000000000001AAA","success":true,"errors":[]}`)
	}
	org.handlers["/services/data/v67.0/sobjects/ScratchOrgInfo/2SR000000000001AAA"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"attributes":{"type":"ScratchOrgInfo"},"Id":"2SR000000000001AAA","Status":"Active","SignupUsername":"test-scratch@example.com","LoginUrl":"`+org.URL+`","AuthCode":"auth-code","ScratchOrg":"00D000000000000","ExpirationDate":"2026-12-31","SignupInstance":"CS99","Edition":"Developer"}`)
	}
	var deployed bool
	org.handlers["/services/Soap/m/67.0"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		if strings.Contains(string(body), "checkDeployStatus") {
			io.WriteString(w, soapBody("<checkDeployStatusResponse><result><done>true</done><id>0Af000000000001AAA</id><status>Succeeded</status><success>true</success></result></checkDeployStatusResponse>"))
			return
		}
		deployed = true
		io.WriteString(w, soapBody("<deployResponse><result><done>false</done><id>0Af000000000001AAA</id></result></deployResponse>"))
	}
	definition := filepath.Join(os.Getenv("HOME"), "project-scratch-def.json")
	os.WriteFile(definition, []byte(`{
  "orgName": "Acme Company",
  "edition": "enterprise",
  "features": ["EnableSetPasswordInApi", "Communities"],
  "settings": {"lightningExperienceSettings": {"enableS1DesktopEnabled": true}}
}`), 0o644)

	out, errOut, status := run("org", "create", "scratch", "-f", definition, "-a", "scratch", "-d", "-y", "3")
	if status != 0 {
		t.Fatalf("status %d, stderr %s\n%s", status, errOut, out)
	}
	for _, want := range []string{
		" ────────────── Creating Scratch Org ──────────────\n", " ✔ Wait For Org ", " ✔ Deploy Settings ", " ✔ Done ",
		" Request Id: 2SR000000000001AAA (" + org.URL + "/2SR000000000001AAA)\n", " OrgId: 00D000000000000\n", " Username: test-scratch@example.com\n", " Alias: scratch\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if !strings.HasSuffix(out, "Your scratch org is ready.\n") {
		t.Errorf("output does not end with the success message:\n%s", out)
	}
	if request["OrgName"] != "Acme Company" || request["Edition"] != "Enterprise" || request["Features"] != "EnableSetPasswordInApi;Communities" ||
		request["DurationDays"] != float64(3) || request["ConnectedAppConsumerKey"] != "PlatformCLI" || request["Namespace"] != "acme" {
		t.Errorf("ScratchOrgInfo request = %v", request)
	}
	if !deployed {
		t.Error("the settings were not deployed")
	}

	fields, err := lib.ReadSFDXAuthFields("test-scratch@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if fields.String("devHubUsername") != "me@example.com" || !fields.Bool("isScratch") || fields.String("expirationDate") != "2026-12-31" ||
		fields.String("refreshToken") != "new-refresh-token" || fields.String("clientId") != "PlatformCLI" {
		t.Errorf("scratch auth fields = %v", fields)
	}
	if lib.ResolveSFDXUsername("scratch") != "test-scratch@example.com" {
		t.Error("the alias was not set")
	}
	if value, _, _ := lib.GetSFConfig(lib.SFConfigTargetOrg); value.Value != "scratch" {
		t.Errorf("target-org = %+v", value)
	}
	// The Dev Hub authorization is unchanged.
	if hub, err := lib.ReadSFDXAuthFields("me@example.com"); err != nil || hub.Bool("isScratch") {
		t.Errorf("Dev Hub auth = %v, err = %v", hub, err)
	}

	// The scratch org is listed as a scratch org.
	out, _, _ = run("org", "list", "--json", "--skip-connection-status")
	if !strings.Contains(out, `"scratchOrgs": [
      {`) || !strings.Contains(out, `"devHubUsername": "me@example.com"`) {
		t.Errorf("org list:\n%s", out)
	}
}

func TestOrgCreateScratchValidation(t *testing.T) {
	devHubSetup(t)
	_, errOut, status := run("org", "create", "scratch")
	if status != 2 || !strings.Contains(errOut, "Exactly one of the following must be provided: --definition-file, --edition, --snapshot, --source-org") {
		t.Errorf("no definition: status %d, stderr %q", status, errOut)
	}
	definition := filepath.Join(os.Getenv("HOME"), "def.json")
	os.WriteFile(definition, []byte(`{"edition":"Developer","objectSettings":{}}`), 0o644)
	_, errOut, status = run("org", "create", "scratch", "-f", definition)
	if status != 1 || !strings.Contains(errOut, `The scratch org definition key "objectSettings" is not supported`) {
		t.Errorf("unsupported key: status %d, stderr %q", status, errOut)
	}
	_, errOut, status = run("org", "create", "scratch", "-e", "developer", "--async")
	if status != 1 || !strings.Contains(errOut, "The --async flag is not supported") {
		t.Errorf("--async: status %d, stderr %q", status, errOut)
	}
}
