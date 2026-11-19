package templates

import (
	"os"
	"strings"
	"testing"
)

func TestArchivesHoldTheTemplateFiles(t *testing.T) {
	names := strings.Join(Names(), " ")
	for _, want := range []string{"apexclass", "apextrigger", "lightningcomponent", "project", "visualforcepage"} {
		if !strings.Contains(" "+names+" ", " "+want+" ") {
			t.Errorf("no archive named %s; archives: %s", want, names)
		}
	}
	files, err := Files("apexclass")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files["DefaultApexClass.cls"]), "public with sharing class <%= apiName %>") {
		t.Errorf("DefaultApexClass.cls = %q", files["DefaultApexClass.cls"])
	}
	project, err := Files("project")
	if err != nil {
		t.Fatal(err)
	}
	// Files in subdirectories and files whose names start with a dot are
	// kept.
	for _, path := range []string{"standard/README.md", "sfdx-project.json", ".forceignore"} {
		if _, ok := project[path]; !ok {
			t.Errorf("the project templates have no file %s", path)
		}
	}
	if _, err := Files("nosuch"); err == nil {
		t.Error("Files returns no error for an unknown name")
	}
	if _, err := File("apexclass", "nosuch.cls"); err == nil {
		t.Error("File returns no error for an unknown file")
	}
}

// The license of the templates has to accompany the binary.  The release
// zips include THIRD-PARTY-LICENSES.txt, so it has to contain the license
// and name the source the archives were built from.
func TestThirdPartyLicensesFileCoversTheTemplates(t *testing.T) {
	if !strings.Contains(License, "Redistribution and use in source and binary forms") || !strings.Contains(License, "Salesforce") {
		t.Fatalf("LICENSE.txt is not the license of the templates:\n%s", License)
	}
	notice, err := os.ReadFile("../../THIRD-PARTY-LICENSES.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notice), License) {
		t.Error("THIRD-PARTY-LICENSES.txt does not contain sf/templates/LICENSE.txt; run make update-sf-templates")
	}
	for _, line := range strings.Split(strings.TrimSpace(Source), "\n") {
		_, value, ok := strings.Cut(line, ": ")
		if !ok || !strings.Contains(string(notice), value) {
			t.Errorf("THIRD-PARTY-LICENSES.txt does not name %q from SOURCE", line)
		}
	}
}

// The release zips are built by Makefile rules that add the license files.
func TestReleaseZipsIncludeTheLicenseFile(t *testing.T) {
	makefile, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	text := string(makefile)
	if !strings.Contains(text, "LICENSE_FILES=THIRD-PARTY-LICENSES.txt") {
		t.Fatal("the Makefile does not define LICENSE_FILES with THIRD-PARTY-LICENSES.txt")
	}
	rules := strings.Count(text, ".zip: ")
	withLicense := strings.Count(text, "\tzip $@ $< $(LICENSE_FILES)\n")
	if rules == 0 || rules != withLicense {
		t.Errorf("%d zip rules, %d of them add $(LICENSE_FILES)", rules, withLicense)
	}
}
