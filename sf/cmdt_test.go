package sf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The output and the generated files of these commands were compared with
// those of sf 2.138.6 and its custom-metadata plugin 3.3.104.

// cmdtProject makes the temporary home directory a Salesforce project and
// returns the objects directory.
func cmdtProject(t *testing.T) string {
	t.Helper()
	home := useTempHome(t)
	if err := os.WriteFile(filepath.Join(home, "sfdx-project.json"), []byte(`{"packageDirectories":[{"path":"force-app","default":true}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	objects := filepath.Join("force-app", "main", "default", "objects")
	if err := os.MkdirAll(objects, 0o755); err != nil {
		t.Fatal(err)
	}
	return objects
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestCmdtCommandsRequireAProject(t *testing.T) {
	useTempHome(t)
	for _, args := range [][]string{{"cmdt", "generate", "object", "-n", "X"}, {"cmdt", "generate", "object", "-n", "X", "--json"}} {
		out, errOut, status := run(args...)
		want := "    RequiresProjectError: This command is required to run from within a \n    Salesforce project directory.\n    Code: RequiresProjectError\n"
		if status != 1 || out != "" || errOut != want {
			t.Errorf("%v: status %d, stdout %q, stderr %q", args, status, out, errOut)
		}
	}
}

func TestCmdtGenerateObject(t *testing.T) {
	objects := cmdtProject(t)
	directory := filepath.Join(objects, "MySetting__mdt")
	expectRun(t, 0, "target dir = "+directory+"\ncreated MySetting__mdt.object-meta.xml\n", "cmdt", "generate", "object", "-n", "MySetting__mdt", "-d", objects+"/")
	want := `<?xml version="1.0" encoding="UTF-8"?>
<CustomObject xmlns="http://soap.sforce.com/2006/04/metadata">
	<label>MySetting</label>
	<pluralLabel>MySetting</pluralLabel>
	<visibility>Public</visibility>
</CustomObject>
`
	if got := readFile(t, filepath.Join(directory, "MySetting__mdt.object-meta.xml")); got != want {
		t.Errorf("object file =\n%s", got)
	}
	expectRun(t, 0, "target dir = "+directory+"\nupdated MySetting__mdt.object-meta.xml\n", "cmdt", "generate", "object", "-n", "MySetting", "-d", objects)
	expectRun(t, 0, "target dir = Here__mdt\ncreated Here__mdt.object-meta.xml\n", "cmdt", "generate", "object", "--typename", "Here")
	expectRun(t, 0, `{
  "status": 0,
  "result": {
    "typename": "Country",
    "label": "Country Name",
    "pluralLabel": "Countries",
    "visibility": "Protected"
  },
  "warnings": []
}
`, "cmdt", "generate", "object", "-n", "Country", "-l", "Country Name", "-p", "Countries", "-v", "Protected", "-d", objects, "--json")
	if got := readFile(t, filepath.Join(objects, "Country__mdt", "Country__mdt.object-meta.xml")); !strings.Contains(got, "<pluralLabel>Countries</pluralLabel>\n\t<visibility>Protected</visibility>") {
		t.Errorf("object file =\n%s", got)
	}

	long := strings.Repeat("x", 41)
	for _, c := range []struct {
		args   []string
		status int
		want   string
	}{
		{[]string{"-n", "bad name"}, 2, "Error (SfError): Parsing --type-name \n\t'bad name' is not a valid API name for a custom metadata type. Metadata names can contain only underscores and alphanumeric characters, must begin with a letter, cannot end with an underscore, and cannot contain two consecutive underscore characters.\nSee more help with --help\n\n"},
		{[]string{"-n", "Abc", "-l", long}, 2, "Error (SfError): Parsing --label \n\t'" + long + "' is too long to be a label. The maximum length of the label is 40 characters.\nSee more help with --help\n\n"},
		{[]string{"-n", "Abc", "-p", long}, 2, "Error (SfError): Parsing --plural-label \n\t'" + long + "' is too long to be a plural label. The maximum length of the plural label is 40 characters.\nSee more help with --help\n\n"},
		{[]string{"-n", "Abc", "-v", "Private"}, 1, "Error (1): Expected --visibility=Private to be one of: PackageProtected, Protected, Public\nSee more help with --help\n\n"},
		{nil, 2, "Error (2): The following error occurred:\n  Missing required flag type-name\nSee more help with --help\n\n"},
	} {
		_, errOut, status := run(append([]string{"cmdt", "generate", "object"}, c.args...)...)
		if status != c.status || errOut != c.want {
			t.Errorf("%v: status %d, stderr %q, want %q", c.args, status, errOut, c.want)
		}
	}
}

func TestCmdtGenerateField(t *testing.T) {
	objects := cmdtProject(t)
	typeDir := filepath.Join(objects, "MySetting__mdt")
	fields := filepath.Join(typeDir, "fields")
	expectRun(t, 0, "target dir = "+fields+"\ncreated Code__c.field-meta.xml\n", "cmdt", "generate", "field", "-n", "Code", "-f", "Text", "-d", typeDir)
	expectRun(t, 0, "target dir = "+fields+"\nupdated Code__c.field-meta.xml\n", "cmdt", "generate", "field", "--fieldname", "Code__c", "--fieldtype", "Text", "--outputdir", typeDir)
	head := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<CustomField xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n"
	// The second run used the name with its suffix, which sf also uses as
	// the label.
	want := head + "\t<fullName>Code__c</fullName>\n\t<fieldManageability>DeveloperControlled</fieldManageability>\n\t<label>Code__c</label>\n\t<type>Text</type>\n\t<unique>false</unique>\n\t<length>100</length>\n</CustomField>\n"
	if got := readFile(t, filepath.Join(fields, "Code__c.field-meta.xml")); got != want {
		t.Errorf("text field =\n%s", got)
	}
	bodies := map[string]string{
		"Checkbox":     "\t<defaultValue>false</defaultValue>\n",
		"Date":         "",
		"DateTime":     "",
		"Email":        "\t<unique>false</unique>\n",
		"Number":       "\t<unique>false</unique>\n\t<precision>18</precision>\n\t<scale>0</scale>\n",
		"Percent":      "\t<precision>18</precision>\n\t<scale>0</scale>\n",
		"Phone":        "",
		"TextArea":     "",
		"LongTextArea": "\t<length>32768</length>\n\t<visibleLines>3</visibleLines>\n",
		"Url":          "",
	}
	for fieldType, body := range bodies {
		name := "F" + fieldType
		if _, errOut, status := run("cmdt", "generate", "field", "-n", name, "-f", fieldType, "-d", typeDir); status != 0 {
			t.Fatalf("%s: status %d %s", fieldType, status, errOut)
		}
		want := head + "\t<fullName>" + name + "__c</fullName>\n\t<fieldManageability>DeveloperControlled</fieldManageability>\n\t<label>" + name + "</label>\n\t<type>" + fieldType + "</type>\n" + body + "</CustomField>\n"
		if got := readFile(t, filepath.Join(fields, name+"__c.field-meta.xml")); got != want {
			t.Errorf("%s field =\n%s\nwant\n%s", fieldType, got, want)
		}
	}
	expectRun(t, 0, `{
  "status": 0,
  "result": {
    "fieldName": "Amount",
    "label": "The Amount",
    "fieldtype": "Number"
  },
  "warnings": []
}
`, "cmdt", "generate", "field", "-n", "Amount", "-f", "Number", "-s", "2", "-l", "The Amount", "-d", typeDir, "--json")
	if got := readFile(t, filepath.Join(fields, "Amount__c.field-meta.xml")); !strings.Contains(got, "\t<label>The Amount</label>\n\t<type>Number</type>\n\t<unique>false</unique>\n\t<precision>16</precision>\n\t<scale>2</scale>\n") {
		t.Errorf("number field =\n%s", got)
	}

	out, errOut, status := run("cmdt", "generate", "field", "-n", "Kind", "-f", "Picklist", "-p", "A,B", "-p", "C", "-d", typeDir)
	if status != 0 || out != "target dir = "+fields+"\ncreated Kind__c.field-meta.xml\n" ||
		errOut != "Warning: The input format for array arguments has changed. Use this format: --array-flag value1 --array-flag value2 --array-flag value3\n" {
		t.Errorf("picklist: status %d, stdout %q, stderr %q", status, out, errOut)
	}
	wantValues := "\t<type>Picklist</type>\n\t<valueSet>\n\t\t<restricted>true</restricted>\n\t\t<valueSetDefinition>\n\t\t\t<sorted>false</sorted>\n" +
		"\t\t\t<value>\n\t\t\t\t<fullName>A</fullName>\n\t\t\t\t<default>false</default>\n\t\t\t\t<label>A</label>\n\t\t\t</value>\n" +
		"\t\t\t<value>\n\t\t\t\t<fullName>B</fullName>\n\t\t\t\t<default>false</default>\n\t\t\t\t<label>B</label>\n\t\t\t</value>\n" +
		"\t\t\t<value>\n\t\t\t\t<fullName>C</fullName>\n\t\t\t\t<default>false</default>\n\t\t\t\t<label>C</label>\n\t\t\t</value>\n" +
		"\t\t</valueSetDefinition>\n\t</valueSet>\n</CustomField>\n"
	if got := readFile(t, filepath.Join(fields, "Kind__c.field-meta.xml")); !strings.HasSuffix(got, wantValues) {
		t.Errorf("picklist field =\n%s", got)
	}
	expectRun(t, 0, "target dir = fields\ncreated Loose__c.field-meta.xml\n", "cmdt", "generate", "field", "-n", "Loose", "-f", "Text")

	for _, c := range []struct {
		args   []string
		status int
		want   string
	}{
		{[]string{"-n", "NoVals", "-f", "Picklist"}, 1, "Error (SfError): You must specify at least one picklist value when the field type is picklist.\n\n"},
		{[]string{"-n", "bad_", "-f", "Text"}, 2, "Error (SfError): Parsing --name \n\t'bad_' is an invalid field. Custom fields must contain only alphanumeric characters, must begin with a letter, can't end with an underscore, and can't contain two consecutive underscore characters.\nSee more help with --help\n\n"},
		{[]string{"-n", "Neg", "-f", "Number", "-s", "-1"}, 2, "Error (2): Parsing --decimal-places \n\tExpected an integer greater than or equal to 0 but received: -1\nSee more help with --help\n\n"},
		{[]string{"-n", "Code", "-f", "Bogus"}, 1, "Error (1): Expected --type=Bogus to be one of: Checkbox, Date, DateTime, Email, Number, Percent, Phone, Picklist, Text, TextArea, LongTextArea, Url\nSee more help with --help\n\n"},
		{[]string{"-n", "Loose"}, 2, "Error (2): The following error occurred:\n  Missing required flag type\nSee more help with --help\n\n"},
	} {
		_, errOut, status := run(append([]string{"cmdt", "generate", "field"}, c.args...)...)
		if status != c.status || errOut != c.want {
			t.Errorf("%v: status %d, stderr %q, want %q", c.args, status, errOut, c.want)
		}
	}
}

// cmdtType creates the type MySetting with the fields the record tests use.
func cmdtType(t *testing.T) {
	t.Helper()
	objects := cmdtProject(t)
	typeDir := filepath.Join(objects, "MySetting__mdt")
	for _, args := range [][]string{
		{"object", "-n", "MySetting", "-d", objects},
		{"field", "-n", "Code", "-f", "Text", "-d", typeDir},
		{"field", "-n", "Amount", "-f", "Number", "-s", "2", "-d", typeDir},
		{"field", "-n", "Whole", "-f", "Number", "-d", typeDir},
		{"field", "-n", "Flag", "-f", "Checkbox", "-d", typeDir},
		{"field", "-n", "Kind", "-f", "Picklist", "-p", "A", "-p", "B", "-d", typeDir},
		{"field", "-n", "One", "-f", "Picklist", "-p", "Only", "-d", typeDir},
	} {
		if _, errOut, status := run(append([]string{"cmdt", "generate"}, args...)...); status != 0 {
			t.Fatalf("%v: status %d %s", args, status, errOut)
		}
	}
}

const cmdtRecordHead = `<?xml version="1.0" encoding="UTF-8"?>
<CustomMetadata xmlns="http://soap.sforce.com/2006/04/metadata" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
`

func TestCmdtGenerateRecord(t *testing.T) {
	cmdtType(t)
	records := filepath.Join("force-app", "main", "default", "customMetadata")
	expectRun(t, 0, "Created custom metadata record of the type 'MySetting' with record name 'Full', label 'Full', and protected 'false' at '"+records+"'.\n",
		"cmdt", "generate", "record", "-t", "MySetting__mdt", "-n", "Full", "Code__c=a <b> & c", "Flag__c=true", "Amount__c=1.5", "Whole__c=7", "Kind__c=A", "Unknown__c=zz", "Empty__c=")
	want := cmdtRecordHead + `    <label>Full</label>
    <protected>false</protected>
    <values>
        <field>Code__c</field>
        <value xsi:type="xsd:string">a &lt;b&gt; &amp; c</value>
    </values>
    <values>
        <field>Flag__c</field>
        <value xsi:type="xsd:boolean">true</value>
    </values>
    <values>
        <field>Amount__c</field>
        <value xsi:type="xsd:double">1.5</value>
    </values>
    <values>
        <field>Whole__c</field>
        <value xsi:type="xsd:int">7</value>
    </values>
    <values>
        <field>Kind__c</field>
        <value xsi:type="xsd:string">A</value>
    </values>
    <values>
        <field>Unknown__c</field>
        <value xsi:type="xsd:string">zz</value>
    </values>
</CustomMetadata>`
	if got := readFile(t, filepath.Join(records, "MySetting.Full.md-meta.xml")); got != want {
		t.Errorf("record =\n%s\nwant\n%s", got, want)
	}

	out, errOut, status := run("cmdt", "generate", "record", "-t", "MySetting", "-n", "Json", "-l", "Second One", "-p", "true", "Code__c=x", "Whole__c=3", "--json")
	if status != 0 {
		t.Fatalf("json: status %d %s", status, errOut)
	}
	for _, want := range []string{
		`    "typename": "MySetting",
    "recordname": "Json",
    "label": "Second One",
    "inputdir": "force-app/main/default/objects",
    "outputdir": "force-app/main/default/customMetadata",
    "protectedFlag": true,
    "varargs": {
      "Code__c": "x",
      "Whole__c": "3"
    },
    "fileData": [
      {
        "fullName": "Amount__c",
        "fieldManageability": "DeveloperControlled",
        "label": "Amount",
        "type": "Number",
        "unique": false,
        "precision": 16,
        "scale": 2
      },`,
		// One picklist value is an object and several are a list, as the
		// XML parser of sf returns them.
		`            "value": {
              "fullName": "Only",
              "default": false,
              "label": "Only"
            }`,
		`            "value": [
              {
                "fullName": "A",`,
		`        "type": "Checkbox",
        "defaultValue": false`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("json output lacks\n%s\nin\n%s", want, out)
		}
	}
	if got := readFile(t, filepath.Join(records, "MySetting.Json.md-meta.xml")); !strings.Contains(got, "    <label>Second One</label>\n    <protected>true</protected>\n") {
		t.Errorf("protected record =\n%s", got)
	}

	long := strings.Repeat("x", 41)
	for _, c := range []struct {
		args   []string
		status int
		want   string
	}{
		{[]string{"-t", "MySetting__mdt", "-n", "bad_"}, 2, "Error (SfError): Parsing --record-name \n\t'bad_' is not a valid record name for a custom metadata record. Record names can only contain alphanumeric characters, must begin with a letter, cannot end with an underscore, and cannot contain two consecutive underscore characters.\nSee more help with --help\n\n"},
		{[]string{"-t", "MySetting__mdt", "-n", "Lbl", "-l", long}, 2, "Error (SfError): Parsing --label \n\t'" + long + "' is too long to be a label. The maximum length of the label is 40 characters.\nSee more help with --help\n\n"},
		{[]string{"-t", "MySetting__mdt", "-n", "Prot", "-p", "maybe"}, 1, "Error (1): Expected --protected=maybe to be one of: true, false\nSee more help with --help\n\n"},
		{[]string{"-t", "MySetting__mdt", "-n", "NoIn", "-i", "nosuchdir"}, 2, "Error (2): Parsing --input-directory \n\tNo directory found at nosuchdir\nSee more help with --help\n\n"},
		{[]string{"-t", "MySetting__mdt"}, 2, "Error (2): The following error occurred:\n  Missing required flag record-name\nSee more help with --help\n\n"},
		{[]string{"-t", "NoSuch__mdt", "-n", "X"}, 1, "Error (ENOENT): ENOENT: no such file or directory, scandir 'force-app/main/default/objects/NoSuch__mdt/fields'\n\n"},
		{[]string{"-t", "MySetting__mdt", "-n", "V1", "novalue"}, 1, "Error (InvalidArgumentFormatError): Error in the following argument\nnovalue\nSet varargs with this format: key=value or key=\"value with spaces\"\n\n"},
		{[]string{"-t", "MySetting__mdt", "-n", "V2", "a=1", "a=2"}, 1, "Error (DuplicateArgumentError): Found duplicate argument a.\n\n"},
	} {
		_, errOut, status := run(append([]string{"cmdt", "generate", "record"}, c.args...)...)
		if status != c.status || errOut != c.want {
			t.Errorf("%v: status %d, stderr %q, want %q", c.args, status, errOut, c.want)
		}
	}
}

func TestCmdtGenerateRecords(t *testing.T) {
	cmdtType(t)
	records := filepath.Join("force-app", "main", "default", "customMetadata")
	os.WriteFile("rows.csv", []byte("Name,Code__c,Whole__c,Flag__c\n\"Row One!\",x1,2,true\n2nd row,\"a,b\",,false\n"), 0o644)
	out, errOut, status := run("cmdt", "generate", "records", "-f", "rows.csv", "-t", "MySetting__mdt", "--json")
	if status != 0 {
		t.Fatalf("status %d %s", status, errOut)
	}
	for _, want := range []string{
		`  "result": [
    {
      "typename": "MySetting",
      "recordname": "Row_One",
      "label": "Row One!",
      "inputdir": "force-app/main/default/objects",
      "outputdir": "force-app/main/default/customMetadata",
      "protected": false,
      "varargs": {
        "Code__c": "x1",
        "Flag__c": "true",
        "Whole__c": "2"
      },
      "fileData": [`,
		`      "recordname": "X2nd_row",
      "label": "2nd row",`,
		`      "varargs": {
        "Code__c": "a,b",
        "Flag__c": "false"
      },`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("json output lacks\n%s\nin\n%s", want, out)
		}
	}
	// Values are written in the order of the field files, not of the
	// columns, and an empty value is left out.
	want := cmdtRecordHead + `    <label>Row One!</label>
    <protected>false</protected>
    <values>
        <field>Code__c</field>
        <value xsi:type="xsd:string">x1</value>
    </values>
    <values>
        <field>Flag__c</field>
        <value xsi:type="xsd:boolean">true</value>
    </values>
    <values>
        <field>Whole__c</field>
        <value xsi:type="xsd:int">2</value>
    </values>
</CustomMetadata>`
	if got := readFile(t, filepath.Join(records, "MySetting.Row_One.md-meta.xml")); got != want {
		t.Errorf("record =\n%s\nwant\n%s", got, want)
	}

	os.WriteFile("named.csv", []byte("Title,Code__c\nT1,c1\n"), 0o644)
	expectRun(t, 0, "Created custom metadata type records from 'named.csv' at '"+records+"'.\n", "cmdt", "generate", "records", "-f", "named.csv", "-t", "MySetting", "-n", "Title")
	if got := readFile(t, filepath.Join(records, "MySetting.T1.md-meta.xml")); !strings.Contains(got, "<label>T1</label>") {
		t.Errorf("record named by another column =\n%s", got)
	}

	os.WriteFile("badcol.csv", []byte("Name,Nope__c\nT1,c1\n"), 0o644)
	os.WriteFile("dupes.csv", []byte("Name,Code__c\nSame,c1\nSame,c2\nTwo,1\nTwo,2\n"), 0o644)
	for _, c := range []struct {
		args   []string
		status int
		want   string
	}{
		{[]string{"-f", "badcol.csv", "-t", "MySetting__mdt"}, 1, "Error (SfError): The column Nope__c is not found on the custom metadata type MySetting.\n\n"},
		{[]string{"-f", "dupes.csv", "-t", "MySetting__mdt"}, 1, "Error (SfError): Your CSV has duplicate values:  Same, Two.  CMDT require unique names in the name field.\n\n"},
		{[]string{"-f", "missing.csv", "-t", "MySetting__mdt"}, 1, "Error (ENOENT): ENOENT: no such file or directory, open 'missing.csv'\n\n"},
		{[]string{"-t", "MySetting__mdt"}, 2, "Error (2): The following error occurred:\n  Missing required flag csv\nSee more help with --help\n\n"},
	} {
		_, errOut, status := run(append([]string{"cmdt", "generate", "records"}, c.args...)...)
		if status != c.status || errOut != c.want {
			t.Errorf("%v: status %d, stderr %q, want %q", c.args, status, errOut, c.want)
		}
	}
}

func TestLegacyAliasReportsDeprecation(t *testing.T) {
	objects := cmdtProject(t)
	out, errOut, status := run("force:cmdt:create", "-n", "Legacy", "-d", objects)
	if status != 0 || !strings.HasSuffix(out, "created Legacy__mdt.object-meta.xml\n") ||
		errOut != "Warning: The \"force cmdt create\" command has been deprecated. Use \"cmdt generate object\" instead.\n" {
		t.Errorf("force:cmdt:create: status %d, stdout %q, stderr %q", status, out, errOut)
	}
	_, errOut, _ = run("cmdt", "field", "create", "-n", "Leg", "-f", "Text", "-d", filepath.Join(objects, "Legacy__mdt"))
	if errOut != "Warning: The \"cmdt field create\" command has been deprecated. Use \"cmdt generate field\" instead.\n" {
		t.Errorf("alias given as words: stderr %q", errOut)
	}
	expectRun(t, 0, `{
  "status": 0,
  "result": [],
  "warnings": [
    "The \"force alias list\" command has been deprecated. Use \"alias list\" instead."
  ]
}
`, "force:alias:list", "--json")
	// The current id has no warning, and neither has an alias of a
	// command this mode does not have.
	if _, errOut, _ := run("alias", "list"); errOut != "" {
		t.Errorf("alias list: stderr %q", errOut)
	}
	if _, errOut, status := run("force:cmdt:generate"); status != 127 || strings.Contains(errOut, "deprecated") {
		t.Errorf("alias of a missing command: status %d, stderr %q", status, errOut)
	}
}
