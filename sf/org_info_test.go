package sf

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOrgOpenURLOnly(t *testing.T) {
	org := loggedInFakeOrg(t)
	var redirect string
	org.handlers["/services/oauth2/singleaccess"] = func(w http.ResponseWriter, r *http.Request) {
		redirect = r.URL.Query().Get("redirect_uri")
		io.WriteString(w, `{"frontdoor_uri":"https://example.my.salesforce.com/secur/frontdoor.jsp?otp=abc"}`)
	}
	out, errOut, status := run("org", "open", "--url-only", "--path", "/lightning/setup/SetupOneHome/home")
	want := "\nAccess org 00D000000000000 as user me@example.com with the following URL: https://example.my.salesforce.com/secur/frontdoor.jsp?otp=abc\n"
	if status != 0 || out != want {
		t.Errorf("status %d, stdout %q", status, out)
	}
	if redirect != "/lightning/setup/SetupOneHome/home" {
		t.Errorf("redirect_uri = %q", redirect)
	}
	if !strings.HasPrefix(errOut, "Warning: This command will expose sensitive information") {
		t.Errorf("stderr = %q", errOut)
	}

	expectRun(t, 0, `{
  "status": 0,
  "result": {
    "orgId": "00D000000000000",
    "url": "https://example.my.salesforce.com/secur/frontdoor.jsp?otp=abc",
    "username": "me@example.com"
  },
  "warnings": [
    "This command will expose sensitive information that allows for subsequent activity using your current authenticated session.\nSharing this information is equivalent to logging someone in under the current credential, resulting in unintended access and escalation of privilege."
  ]
}
`, "org", "open", "-r", "--json")
}

func TestOrgOpenStartsTheBrowser(t *testing.T) {
	org := loggedInFakeOrg(t)
	org.handlers["/services/oauth2/singleaccess"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"frontdoor_uri":"https://example.my.salesforce.com/secur/frontdoor.jsp?otp=abc"}`)
	}
	var opened string
	previous := openBrowser
	openBrowser = func(url string) error {
		opened = url
		return nil
	}
	defer func() { openBrowser = previous }()
	expectRun(t, 0, "Opening org 00D000000000000 as user me@example.com\n", "org", "open")
	if opened != "https://example.my.salesforce.com/secur/frontdoor.jsp?otp=abc" {
		t.Errorf("opened %q", opened)
	}
	_, errOut, status := run("org", "open", "--private")
	if status != 1 || !strings.Contains(errOut, "The --private flag is not supported") {
		t.Errorf("--private: status %d, stderr %q", status, errOut)
	}
}

func TestOrgListLimits(t *testing.T) {
	org := loggedInFakeOrg(t)
	org.handlers["/services/data/v67.0/limits"] = func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"DailyApiRequests":{"Max":15000,"Remaining":14998},"ActiveScratchOrgs":{"Max":3,"Remaining":3}}`)
	}
	expectRun(t, 0, `┌───────────────────┬───────────┬───────┐
│ Name              │ Remaining │ Max   │
├───────────────────┼───────────┼───────┤
│ DailyApiRequests  │ 14998     │ 15000 │
│ ActiveScratchOrgs │ 3         │ 3     │
└───────────────────┴───────────┴───────┘

`, "org", "list", "limits")
	expectRun(t, 0, `{
  "status": 0,
  "result": [
    {
      "name": "DailyApiRequests",
      "max": 15000,
      "remaining": 14998
    },
    {
      "name": "ActiveScratchOrgs",
      "max": 3,
      "remaining": 3
    }
  ],
  "warnings": []
}
`, "force:limits:api:display", "--json")
}

const soapEnvelope = `<?xml version="1.0" encoding="UTF-8"?><soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://soap.sforce.com/2006/04/metadata"><soapenv:Body>%s</soapenv:Body></soapenv:Envelope>`

func TestOrgListMetadata(t *testing.T) {
	org := loggedInFakeOrg(t)
	org.handlers["/services/Soap/m/67.0"] = func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/xml")
		if strings.Contains(string(body), "<type>ApexClass</type>") {
			io.WriteString(w, strings.Replace(soapEnvelope, "%s", `<listMetadataResponse>`+
				`<result><createdByName>Ada</createdByName><createdDate>2026-01-02T03:04:05.000Z</createdDate><fileName>classes/B.cls</fileName><fullName>B</fullName><id>01p000000000002AAA</id><lastModifiedByName>Ada</lastModifiedByName><lastModifiedDate>2026-02-03T00:00:00.000Z</lastModifiedDate><manageableState>unmanaged</manageableState><type>ApexClass</type></result>`+
				`<result><fileName>classes/A.cls</fileName><fullName>A</fullName><id>01p000000000001AAA</id><manageableState>installed</manageableState><namespacePrefix>ns</namespacePrefix><type>ApexClass</type></result>`+
				`</listMetadataResponse>`, 1))
			return
		}
		io.WriteString(w, strings.Replace(soapEnvelope, "%s", `<listMetadataResponse/>`, 1))
	}
	expectRun(t, 0, `ApexClass
┌────────────┬──────────────┬───────────┬────────────────────┬──────────────────┬───────────────┬──────────────────┬──────────────────┐
│ Created By │ Created Date │ Full Name │ Id                 │ Last Modified By │ Last Modified │ Manageable State │ Namespace Prefix │
├────────────┼──────────────┼───────────┼────────────────────┼──────────────────┼───────────────┼──────────────────┼──────────────────┤
│            │              │ A         │ 01p000000000001AAA │                  │               │ installed        │ ns               │
│ Ada        │ 2026-01-02   │ B         │ 01p000000000002AAA │ Ada              │ 2026-02-03    │ unmanaged        │                  │
└────────────┴──────────────┴───────────┴────────────────────┴──────────────────┴───────────────┴──────────────────┴──────────────────┘

`, "org", "list", "metadata", "-m", "ApexClass")

	out, _, status := run("org", "list", "metadata", "--metadata-type", "ApexClass", "--json")
	if status != 0 || !strings.Contains(out, `"fileName": "classes/A.cls",
      "fullName": "A",
      "id": "01p000000000001AAA",
      "manageableState": "installed",
      "namespacePrefix": "ns",
      "type": "ApexClass",
      "createdById": "",`) {
		t.Errorf("json: status %d\n%s", status, out)
	}

	out, errOut, status := run("org", "list", "metadata", "-m", "Flow")
	if status != 0 || out != "" || errOut != "Warning: No metadata found for type: Flow in org: me@example.com.\n" {
		t.Errorf("no results: status %d, stdout %q, stderr %q", status, out, errOut)
	}
	_, errOut, status = run("org", "list", "metadata")
	if status != 2 || errOut != "Error (2): The following error occurred:\n  Missing required flag metadata-type\nSee more help with --help\n\n\n" {
		t.Errorf("missing flag: status %d, stderr %q", status, errOut)
	}
}

func TestOrgListMetadataTypes(t *testing.T) {
	org := loggedInFakeOrg(t)
	org.handlers["/services/Soap/m/67.0"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		io.WriteString(w, strings.Replace(soapEnvelope, "%s", `<describeMetadataResponse><result>`+
			`<metadataObjects><childXmlNames>CustomField</childXmlNames><childXmlNames>ListView</childXmlNames><directoryName>objects</directoryName><inFolder>false</inFolder><metaFile>false</metaFile><suffix>object</suffix><xmlName>CustomObject</xmlName></metadataObjects>`+
			`<metadataObjects><directoryName>classes</directoryName><inFolder>false</inFolder><metaFile>true</metaFile><suffix>cls</suffix><xmlName>ApexClass</xmlName></metadataObjects>`+
			`<organizationNamespace>ns</organizationNamespace><partialSaveAllowed>true</partialSaveAllowed><testRequired>false</testRequired>`+
			`</result></describeMetadataResponse>`, 1))
	}
	expectRun(t, 0, `Metadata
┌──────────────┬──────────────────────────┬────────────────┬───────────┬───────────┬────────┐
│ Xml Names    │ Child Xml Names          │ Directory Name │ In Folder │ Meta File │ Suffix │
├──────────────┼──────────────────────────┼────────────────┼───────────┼───────────┼────────┤
│ ApexClass    │                          │ classes        │ false     │ true      │ cls    │
│ CustomObject │ [ CustomField ListView ] │ objects        │ false     │ false     │ object │
└──────────────┴──────────────────────────┴────────────────┴───────────┴───────────┴────────┘

Organizational Namespace: ns
Partial Save Allowed: true
Test Required: false
`, "org", "list", "metadata-types")
	out, _, status := run("org", "list", "metadata-types", "--json")
	if status != 0 || !strings.Contains(out, `"organizationNamespace": "ns"`) || !strings.Contains(out, `"metadataVersion": "67.0"`) ||
		!strings.Contains(out, "\"xmlName\": \"CustomObject\",\n        \"childXmlNames\": [\n          \"CustomField\",\n          \"ListView\"\n        ]") {
		t.Errorf("json: status %d\n%s", status, out)
	}
}
