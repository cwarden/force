package sf

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ForceCLI/force/desktop"
	"github.com/ForceCLI/force/lib"
)

const securityWarning = "This command will expose sensitive information that allows for subsequent activity using your current authenticated session.\nSharing this information is equivalent to logging someone in under the current credential, resulting in unintended access and escalation of privilege."

type orgOpenResult struct {
	OrgId    string `json:"orgId"`
	URL      string `json:"url"`
	Username string `json:"username"`
}

// openBrowser is replaced in tests.
var openBrowser = desktop.Open

func newOrgOpenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "open",
		Short: "Open your default scratch org, or another specified org, in a browser.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := unsupportedFlags(cmd, "private", "browser", "source-file"); err != nil {
				return err
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			path, _ := cmd.Flags().GetString("path")
			if path == "" {
				path = os.Getenv("FORCE_OPEN_URL")
			}
			// sf requests a single-use URL from the UI Bridge API.
			request := "/services/oauth2/singleaccess"
			if path != "" {
				request += "?redirect_uri=" + url.QueryEscape(path)
			}
			body, err := org.Force.GetAbsolute(request)
			if err != nil {
				return apiError(err)
			}
			var response struct {
				FrontdoorURI string `json:"frontdoor_uri"`
			}
			if err := json.Unmarshal([]byte(body), &response); err != nil || response.FrontdoorURI == "" {
				return newError("FrontdoorURLError", "Failed to generate a frontdoor URL.")
			}
			result := orgOpenResult{OrgId: org.Fields.String("orgId"), URL: response.FrontdoorURI, Username: org.Username}
			urlOnly, _ := cmd.Flags().GetBool("url-only")
			if urlOnly {
				warn(cmd, securityWarning)
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintln(w)
					fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Access org %s as user %s with the following URL: %s", result.OrgId, result.Username, result.URL)))
				})
			}
			if err := emit(cmd, result, func(w io.Writer) {
				fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Opening org %s as user %s", result.OrgId, result.Username)))
			}); err != nil {
				return err
			}
			return openBrowser(result.URL)
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().Bool("private", false, "Open the org in the default browser using private (incognito) mode.")
	cmd.Flags().StringP("browser", "b", "", "Browser where the org opens.")
	cmd.Flags().StringP("path", "p", "", "Navigation URL path to open a specific page.")
	cmd.Flags().BoolP("url-only", "r", false, "Display navigation URL, but don’t launch browser.")
	cmd.Flags().StringP("source-file", "f", "", "Path to an ApexPage or FlexiPage to open in Lightning App Builder.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

func newOrgListLimitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "limits",
		Short:       "Display information about limits in your org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "LimitsApiDisplayCommand"},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			body, err := org.Force.GetREST("/limits")
			if err != nil {
				// sf wraps this error, which drops the "data" field
				// and keeps the error name as the code.
				converted := apiError(err)
				if e, ok := converted.(*Error); ok {
					e.Label = ""
					e.Data = nil
				}
				return converted
			}
			parsed, err := parseOrderedJSON([]byte(body))
			if err != nil {
				return err
			}
			limits, ok := parsed.(*orderedMap)
			if !ok {
				return fmt.Errorf("unexpected response from the limits resource")
			}
			result := make([]*orderedMap, 0, len(limits.keys))
			rows := make([][]string, 0, len(limits.keys))
			for _, name := range limits.keys {
				limit, ok := limits.values[name].(*orderedMap)
				if !ok {
					continue
				}
				item := newOrderedMap()
				item.Set("name", name)
				item.Set("max", limit.values["Max"])
				item.Set("remaining", limit.values["Remaining"])
				result = append(result, item)
				rows = append(rows, []string{name, displayValue(limit.values["Remaining"]), displayValue(limit.values["Max"])})
			}
			return emit(cmd, result, func(w io.Writer) {
				printTable(w, "", []string{"Name", "Remaining", "Max"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

// soapResults returns the <result> elements of a SOAP response as ordered
// maps of their child elements.  A child element that repeats becomes an
// array.
func soapResults(body []byte) ([]*orderedMap, error) {
	dec := xml.NewDecoder(bytes.NewReader(body))
	var results []*orderedMap
	for {
		token, err := dec.Token()
		if err == io.EOF {
			return results, nil
		}
		if err != nil {
			return nil, err
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "result" {
			value, err := decodeXMLElement(dec)
			if err != nil {
				return nil, err
			}
			if object, ok := value.(*orderedMap); ok {
				results = append(results, object)
			}
		}
	}
}

// decodeXMLElement reads the content of the element whose start tag was
// just read.  An element with child elements becomes an ordered map; other
// elements become their text.
func decodeXMLElement(dec *xml.Decoder) (interface{}, error) {
	object := newOrderedMap()
	var text strings.Builder
	for {
		token, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			child, err := decodeXMLElement(dec)
			if err != nil {
				return nil, err
			}
			name := t.Name.Local
			if existing, ok := object.Get(name); ok {
				if list, isList := existing.([]interface{}); isList {
					object.Set(name, append(list, child))
				} else {
					object.Set(name, []interface{}{existing, child})
				}
			} else {
				object.Set(name, child)
			}
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			if len(object.keys) > 0 {
				return object, nil
			}
			return text.String(), nil
		}
	}
}

func stringField(object *orderedMap, key string) string {
	value, _ := object.values[key].(string)
	return value
}

func newOrgListMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "metadata",
		Short:       "List the metadata components and properties of a specified type.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ListMetadata"},
		RunE: func(cmd *cobra.Command, args []string) error {
			metadataType, _ := cmd.Flags().GetString("metadata-type")
			if metadataType == "" {
				return missingFlagError("metadata-type")
			}
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			query := metadataType
			if folder, _ := cmd.Flags().GetString("folder"); folder != "" {
				query += ":" + folder
			}
			body, err := org.Force.Metadata.ListMetadata(query)
			if err != nil {
				return apiError(err)
			}
			result, err := soapResults(body)
			if err != nil {
				return err
			}
			if result == nil {
				result = []*orderedMap{}
			}
			// sf reports these properties as empty strings when the
			// response leaves them out.
			for _, item := range result {
				for _, key := range []string{"createdById", "createdByName", "createdDate", "fileName", "fullName", "id", "lastModifiedById", "lastModifiedByName", "lastModifiedDate", "type"} {
					if _, ok := item.Get(key); !ok {
						item.Set(key, "")
					}
				}
			}
			if outputFile, _ := cmd.Flags().GetString("output-file"); outputFile != "" {
				var encoded bytes.Buffer
				enc := json.NewEncoder(&encoded)
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				if err := enc.Encode(result); err != nil {
					return err
				}
				if err := os.WriteFile(outputFile, bytes.TrimRight(encoded.Bytes(), "\n"), 0o644); err != nil {
					return err
				}
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Wrote result file to %s.", outputFile)))
				})
			}
			if len(result) == 0 {
				warn(cmd, fmt.Sprintf("No metadata found for type: %s in org: %s.", metadataType, org.Username))
				return emit(cmd, result, nil)
			}
			date := func(value string) string {
				day, _, _ := strings.Cut(value, "T")
				return day
			}
			sorted := append([]*orderedMap{}, result...)
			sort.SliceStable(sorted, func(i, j int) bool {
				return stringField(sorted[i], "manageableState") < stringField(sorted[j], "manageableState")
			})
			rows := make([][]string, 0, len(sorted))
			for _, item := range sorted {
				rows = append(rows, []string{
					stringField(item, "createdByName"),
					date(stringField(item, "createdDate")),
					stringField(item, "fullName"),
					stringField(item, "id"),
					stringField(item, "lastModifiedByName"),
					date(stringField(item, "lastModifiedDate")),
					stringField(item, "manageableState"),
					stringField(item, "namespacePrefix"),
				})
			}
			return emit(cmd, result, func(w io.Writer) {
				printTable(w, metadataType, []string{"Created By", "Created Date", "Full Name", "Id", "Last Modified By", "Last Modified", "Manageable State", "Namespace Prefix"}, rows)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("output-file", "f", "", "File where the results of the command are stored.")
	cmd.Flags().StringP("metadata-type", "m", "", "Metadata type to be retrieved, such as CustomObject.")
	cmd.Flags().String("folder", "", "Folder associated with the component.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}

type metadataTypeResult struct {
	DirectoryName string   `json:"directoryName"`
	InFolder      bool     `json:"inFolder"`
	MetaFile      bool     `json:"metaFile"`
	Suffix        string   `json:"suffix,omitempty"`
	XmlName       string   `json:"xmlName"`
	ChildXmlNames []string `json:"childXmlNames"`
}

type metadataTypesResult struct {
	OrganizationNamespace string               `json:"organizationNamespace"`
	PartialSaveAllowed    bool                 `json:"partialSaveAllowed"`
	TestRequired          bool                 `json:"testRequired"`
	MetadataObjects       []metadataTypeResult `json:"metadataObjects"`
	MetadataVersion       string               `json:"metadataVersion"`
}

func newOrgListMetadataTypesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "metadata-types",
		Short:       "Display details about the metadata types that are enabled for your org.",
		Args:        cobra.NoArgs,
		Annotations: map[string]string{commandNameAnnotation: "ListMetadataTypes"},
		RunE: func(cmd *cobra.Command, args []string) error {
			org, err := resolveTargetOrg(cmd)
			if err != nil {
				return err
			}
			describe, err := org.Force.Metadata.DescribeMetadata()
			if err != nil {
				return apiError(err)
			}
			result := metadataTypesResult{
				OrganizationNamespace: describe.NamespacePrefix,
				PartialSaveAllowed:    describe.PartialSaveAllowed,
				TestRequired:          describe.TestRequired,
				MetadataObjects:       make([]metadataTypeResult, 0, len(describe.MetadataObjects)),
				MetadataVersion:       lib.ApiVersionNumber(),
			}
			for _, object := range describe.MetadataObjects {
				children := object.ChildXmlNames
				if children == nil {
					children = []string{}
				}
				result.MetadataObjects = append(result.MetadataObjects, metadataTypeResult{
					DirectoryName: object.DirectoryName,
					InFolder:      object.InFolder,
					MetaFile:      object.MetaFile,
					Suffix:        object.Suffix,
					XmlName:       object.XmlName,
					ChildXmlNames: children,
				})
			}
			if outputFile, _ := cmd.Flags().GetString("output-file"); outputFile != "" {
				encoded, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return err
				}
				if err := os.WriteFile(outputFile, encoded, 0o644); err != nil {
					return err
				}
				return emit(cmd, result, func(w io.Writer) {
					fmt.Fprintln(w, paint(w, styleGreen, fmt.Sprintf("Wrote result file to %s.", outputFile)))
				})
			}
			sorted := append([]metadataTypeResult{}, result.MetadataObjects...)
			sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].XmlName < sorted[j].XmlName })
			rows := make([][]string, 0, len(sorted))
			for _, object := range sorted {
				children := ""
				if len(object.ChildXmlNames) > 0 {
					children = "[ " + strings.Join(object.ChildXmlNames, " ") + " ]"
				}
				rows = append(rows, []string{
					object.XmlName, children, object.DirectoryName,
					fmt.Sprintf("%t", object.InFolder), fmt.Sprintf("%t", object.MetaFile), object.Suffix,
				})
			}
			return emit(cmd, result, func(w io.Writer) {
				printTable(w, "Metadata", []string{"Xml Names", "Child Xml Names", "Directory Name", "In Folder", "Meta File", "Suffix"}, rows)
				fmt.Fprintf(w, "Organizational Namespace: %s\n", result.OrganizationNamespace)
				fmt.Fprintf(w, "Partial Save Allowed: %t\n", result.PartialSaveAllowed)
				fmt.Fprintf(w, "Test Required: %t\n", result.TestRequired)
			})
		},
	}
	addJSONFlag(cmd)
	cmd.Flags().StringP("output-file", "f", "", "File where the results of the command are stored.")
	addAPIVersionFlag(cmd)
	addTargetOrgFlag(cmd)
	return cmd
}
