package sf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// templateAPIVersion is the API version sf 2.138.6 writes into generated
// metadata files when --api-version is not given.
const templateAPIVersion = "66.0"

func newTemplateCmd() *cobra.Command {
	return newGroup("template", "Commands to create metadata from templates.",
		newGroup("generate", "Create metadata from templates.",
			newGroup("apex", "Create Apex classes and triggers.",
				newTemplateApexClassCmd(),
				newTemplateApexTriggerCmd(),
			),
			newGroup("visualforce", "Create Visualforce pages and components.",
				newTemplateVisualforceCmd("page", "ApexPage", "Generate a Visualforce Page.",
					"<apex:page >\n<!-- Begin Default Content REMOVE THIS -->\n<h1>Congratulations</h1>\nThis is your new Page\n<!-- End Default Content REMOVE THIS -->\n</apex:page>",
					// The irregular spacing is what sf writes.
					"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ApexPage xmlns=\"http://soap.sforce.com/2006/04/metadata\"> \n    <apiVersion>%s</apiVersion>\n    <availableInTouch>false</availableInTouch>\n     <confirmationTokenRequired>false</confirmationTokenRequired>\n    <label>%s</label>\n</ApexPage>\n"),
				newTemplateVisualforceCmd("component", "ApexComponent", "Generate a Visualforce Component.",
					"<apex:component >\n<!-- Begin Default Content REMOVE THIS -->\n<h1>Congratulations</h1>\nThis is your new Component\n<!-- End Default Content REMOVE THIS -->\n</apex:component>",
					"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ApexComponent xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <apiVersion>%s</apiVersion>\n    <label>%s</label>\n</ApexComponent>\n"),
			),
		),
	)
}

type generatedFile struct {
	name    string
	content string
}

type templateResult struct {
	OutputDir string   `json:"outputDir"`
	Created   []string `json:"created"`
	RawOutput string   `json:"rawOutput"`
}

// writeTemplates writes files into --output-dir and reports them the way
// sf does, with paths relative to the working directory.
func writeTemplates(cmd *cobra.Command, files []generatedFile) error {
	outputDir, _ := cmd.Flags().GetString("output-dir")
	absolute, err := filepath.Abs(outputDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(absolute, 0o755); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	result := templateResult{OutputDir: absolute, Created: []string{}}
	for _, file := range files {
		path := filepath.Join(absolute, file.name)
		if err := os.WriteFile(path, []byte(file.content), 0o644); err != nil {
			return err
		}
		relative, err := filepath.Rel(wd, path)
		if err != nil {
			relative = path
		}
		result.Created = append(result.Created, filepath.ToSlash(relative))
	}
	var raw strings.Builder
	fmt.Fprintf(&raw, "target dir = %s\n", absolute)
	for _, created := range result.Created {
		fmt.Fprintf(&raw, "  create %s\n", created)
	}
	result.RawOutput = raw.String()
	return emit(cmd, result, func(w io.Writer) {
		fmt.Fprintln(w, result.RawOutput)
	})
}

func addTemplateFlags(cmd *cobra.Command) {
	addJSONFlag(cmd)
	cmd.Flags().StringP("name", "n", "", "Name of the generated metadata.")
	cmd.Flags().StringP("output-dir", "d", ".", "Directory for saving the created files.")
	cmd.Flags().String("api-version", "", "Override the api version used for api requests made by this command")
}

func templateFlags(cmd *cobra.Command) (name string, apiVersion string, err error) {
	name, _ = cmd.Flags().GetString("name")
	if name == "" {
		return "", "", missingFlagError("name")
	}
	apiVersion, _ = cmd.Flags().GetString("api-version")
	if apiVersion == "" {
		apiVersion = templateAPIVersion
	}
	return name, apiVersion, nil
}

func newTemplateApexClassCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "class",
		Short:       "Generate an Apex class.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ApexClass"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, apiVersion, err := templateFlags(cmd)
			if err != nil {
				return err
			}
			if template, _ := cmd.Flags().GetString("template"); template != "DefaultApexClass" {
				return newError("UnsupportedFlagError", "The --template value %s is not supported by the sf-compatible mode of force. Only DefaultApexClass is supported.", template)
			}
			return writeTemplates(cmd, []generatedFile{
				{name + ".cls", fmt.Sprintf("public with sharing class %s {\n    public %s() {\n\n    }\n}", name, name)},
				{name + ".cls-meta.xml", fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ApexClass xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n    <apiVersion>%s</apiVersion>\n    <status>Active</status>\n</ApexClass>\n", apiVersion)},
			})
		},
	}
	addTemplateFlags(cmd)
	cmd.Flags().StringP("template", "t", "DefaultApexClass", "Template to use for file creation.")
	return cmd
}

var triggerEvents = []string{"before insert", "before update", "before delete", "after insert", "after update", "after delete", "after undelete"}

func newTemplateApexTriggerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "trigger",
		Short:       "Generate an Apex trigger.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ApexTrigger"},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, apiVersion, err := templateFlags(cmd)
			if err != nil {
				return err
			}
			sobject, _ := cmd.Flags().GetString("sobject")
			events := splitList(cmd, "event")
			if len(events) == 0 {
				events = []string{"before insert"}
			}
			for _, event := range events {
				valid := false
				for _, known := range triggerEvents {
					if event == known {
						valid = true
					}
				}
				if !valid {
					return &Error{Message: fmt.Sprintf("Expected --event=%s to be one of: %s\nSee more help with --help", event, strings.Join(triggerEvents, ", ")), ExitCode: 2}
				}
			}
			return writeTemplates(cmd, []generatedFile{
				{name + ".trigger", fmt.Sprintf("trigger %s on %s (%s) {\n\n}", name, sobject, strings.Join(events, ","))},
				{name + ".trigger-meta.xml", fmt.Sprintf("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<ApexTrigger xmlns=\"http://soap.sforce.com/2006/04/metadata\">\n  <apiVersion>%s</apiVersion>\n  <status>Active</status>\n</ApexTrigger>\n", apiVersion)},
			})
		},
	}
	addTemplateFlags(cmd)
	cmd.Flags().StringP("template", "t", "ApexTrigger", "Template to use for file creation.")
	cmd.Flags().StringP("sobject", "s", "SOBJECT", "Salesforce object to generate a trigger on.")
	cmd.Flags().StringArrayP("event", "e", nil, "Events that fire the trigger.")
	return cmd
}

func newTemplateVisualforceCmd(kind, className, short, body, meta string) *cobra.Command {
	cmd := &cobra.Command{
		Use:         kind,
		Short:       short,
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "Visualforce" + strings.ToUpper(kind[:1]) + kind[1:]},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, apiVersion, err := templateFlags(cmd)
			if err != nil {
				return err
			}
			label, _ := cmd.Flags().GetString("label")
			if label == "" {
				return missingFlagError("label")
			}
			return writeTemplates(cmd, []generatedFile{
				{name + "." + kind, body},
				{name + "." + kind + "-meta.xml", fmt.Sprintf(meta, apiVersion, label)},
			})
		},
	}
	addTemplateFlags(cmd)
	cmd.Flags().StringP("template", "t", "", "Template to use for file creation.")
	cmd.Flags().StringP("label", "l", "", "Visualforce "+strings.ToUpper(kind[:1])+kind[1:]+" label.")
	return cmd
}
