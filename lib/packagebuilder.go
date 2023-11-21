package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ForceCLI/force/config"
	"github.com/ForceCLI/force/lib/metadata"
)

// Structs for XML building
type Package struct {
	Xmlns   string     `xml:"xmlns,attr"`
	Types   []MetaType `xml:"types"`
	Version string     `xml:"version"`
}

type MetadataType string
type MetadataTypeDirectory string

type MetaType struct {
	Members []string     `xml:"members"`
	Name    MetadataType `xml:"name"`
}

func createPackageXml() Package {
	return Package{
		Version: strings.TrimPrefix(apiVersion, "v"),
		Xmlns:   "http://soap.sforce.com/2006/04/metadata",
	}
}

type metapath struct {
	path       MetadataTypeDirectory
	name       MetadataType
	hasFolder  bool
	onlyFolder bool
	extension  string
}

var metapaths = []metapath{
	{path: "actionLinkGroupTemplates", name: "ActionLinkGroupTemplate"},
	{path: "analyticSnapshots", name: "AnalyticSnapshot"},
	{path: "apexEmailNotifications", name: "ApexEmailNotifications"},
	{path: "applications", name: "CustomApplication"},
	{path: "appMenus", name: "AppMenu"},
	{path: "approvalProcesses", name: "ApprovalProcess"},
	{path: "assignmentRules", name: "AssignmentRules"},
	{path: "audience", name: "Audience"},
	{path: "authproviders", name: "AuthProvider"},
	{path: "aura", name: "AuraDefinitionBundle", hasFolder: true, onlyFolder: true},
	{path: "autoResponseRules", name: "AutoResponseRules"},
	{path: "callCenters", name: "CallCenter"},
	{path: "cachePartitions", name: "PlatformCachePartition"},
	{path: "certs", name: "Certificate"},
	{path: "channelLayouts", name: "ChannelLayout"},
	{path: "classes", name: "ApexClass"},
	{path: "cleanDataServices", name: "CleanDataService"},
	{path: "communities", name: "Community"},
	{path: "components", name: "ApexComponent"},
	{path: "connectedApps", name: "ConnectedApp"},
	{path: "contentassets", name: "ContentAsset"},
	{path: "corsWhitelistOrigins", name: "CorsWhitelistOrigin"},
	{path: "customApplicationComponents", name: "CustomApplicationComponent"},
	{path: "customMetadata", name: "CustomMetadata"},
	{path: "notificationtypes", name: "CustomNotificationType"},
	{path: "customHelpMenuSections", name: "CustomHelpMenuSection"},
	{path: "customPermissions", name: "CustomPermission"},
	{path: "dashboards", name: "Dashboard", hasFolder: true},
	{path: "dataSources", name: "ExternalDataSource"},
	{path: "datacategorygroups", name: "DataCategoryGroup"},
	{path: "delegateGroups", name: "DelegateGroup"},
	{path: "documents", name: "Document", hasFolder: true},
	{path: "duplicateRules", name: "DuplicateRule"},
	{path: "dw", name: "DataWeaveResource"},
	{path: "EmbeddedServiceConfig", name: "EmbeddedServiceConfig"},
	{path: "email", name: "EmailTemplate", hasFolder: true},
	{path: "escalationRules", name: "EscalationRules"},
	{path: "experiences", name: "ExperienceBundle"},
	{path: "externalCredentials", name: "ExternalCredential"},
	{path: "feedFilters", name: "CustomFeedFilter"},
	{path: "flexipages", name: "FlexiPage"},
	{path: "flowDefinitions", name: "FlowDefinition"},
	{path: "flows", name: "Flow"},
	{path: "flowtests", name: "FlowTest"},
	{path: "globalPicklists", name: "GlobalPicklist"},
	{path: "globalValueSets", name: "GlobalValueSet"},
	{path: "globalValueSetTranslations", name: "GlobalValueSetTranslation"},
	{path: "groups", name: "Group"},
	{path: "homePageComponents", name: "HomePageComponent"},
	{path: "homePageLayouts", name: "HomePageLayout"},
	{path: "installedPackages", name: "InstalledPackage"},
	{path: "labels", name: "CustomLabels"},
	{path: "layouts", name: "Layout"},
	{path: "LeadConvertSettings", name: "LeadConvertSettings"},
	{path: "letterhead", name: "Letterhead"},
	{path: "lwc", name: "LightningComponentBundle", hasFolder: true, onlyFolder: true},
	{path: "matchingRules", name: "MatchingRules"},
	{path: "matchingRules", name: "MatchingRule"},
	{path: "messageChannels", name: "LightningMessageChannel"},
	{path: "namedCredentials", name: "NamedCredential"},
	{path: "notificationTypeConfig", name: "NotificationTypeConfig"},
	{path: "networks", name: "Network"},
	{path: "objects", name: "CustomObject"},
	{path: "objectTranslations", name: "CustomObjectTranslation"},
	{path: "omniDataTransforms", name: "OmniDataTransform"},
	{path: "omniIntegrationProcedures", name: "OmniIntegrationProcedure"},
	{path: "omniScripts", name: "OmniScript"},
	{path: "omniUiCard", name: "OmniUiCard"},
	{path: "pages", name: "ApexPage"},
	{path: "pathAssistants", name: "PathAssistant"},
	{path: "permissionsets", name: "PermissionSet"},
	{path: "permissionsetgroups", name: "PermissionSetGroup"},
	{path: "platformEventChannels", name: "PlatformEventChannel"},
	{path: "platformEventChannelMembers", name: "PlatformEventChannelMember"},
	{path: "PlatformEventSubscriberConfigs", name: "PlatformEventSubscriberConfig"},
	{path: "postTemplates", name: "PostTemplate"},
	{path: "profiles", name: "Profile", extension: ".profile"},
	{path: "postTemplates", name: "PostTemplate"},
	{path: "postTemplates", name: "PostTemplate"},
	{path: "profiles", name: "Profile"},
	{path: "profileSessionSettings", name: "ProfileSessionSetting"},
	{path: "queues", name: "Queue"},
	{path: "quickActions", name: "QuickAction"},
	{path: "restrictionRules", name: "RestrictionRule"},
	{path: "remoteSiteSettings", name: "RemoteSiteSetting"},
	{path: "reports", name: "Report", hasFolder: true},
	{path: "reportTypes", name: "ReportType"},
	{path: "roles", name: "Role"},
	{path: "scontrols", name: "Scontrol"},
	{path: "settings", name: "Settings"},
	{path: "sharingRules", name: "SharingRules"},
	{path: "sharingSets", name: "SharingSet"},
	{path: "siteDotComSites", name: "SiteDotCom"},
	{path: "sites", name: "CustomSite"},
	{path: "standardValueSets", name: "StandardValueSet"},
	{path: "staticresources", name: "StaticResource"},
	{path: "synonymDictionaries", name: "SynonymDictionary"},
	{path: "tabs", name: "CustomTab"},
	{path: "translations", name: "Translations"},
	{path: "triggers", name: "ApexTrigger"},
	{path: "weblinks", name: "CustomPageWebLink"},
	{path: "workflows", name: "Workflow"},
	{path: "cspTrustedSites", name: "CspTrustedSite"},
}

type PackageBuilder struct {
	metadata []metadata.Metadata
}

func (pb PackageBuilder) Size() int {
	return len(pb.metadata)
}

func NewPushBuilder() PackageBuilder {
	pb := PackageBuilder{}
	return pb
}

func NewFetchBuilder() PackageBuilder {
	pb := PackageBuilder{}
	return pb
}

// Build and return package.xml
func (pb PackageBuilder) PackageXml() []byte {
	p := createPackageXml()

	types := make(map[string][]string)

	for _, m := range pb.metadata {
		if members, ok := types[m.DeployedType()]; ok {
			types[m.DeployedType()] = append(members, m.Name())
		} else {
			types[m.DeployedType()] = []string{m.Name()}
		}
	}

	for k, v := range types {
		p.Types = append(p.Types, MetaType{Name: MetadataType(k), Members: v})
	}

	byteXml, _ := xml.MarshalIndent(p, "", "    ")
	byteXml = append([]byte(xml.Header), byteXml...)
	return byteXml
}

func (pb *PackageBuilder) PackageFiles() (ForceMetadataFiles, error) {
	f := make(ForceMetadataFiles)
	f["package.xml"] = pb.PackageXml()
	for _, m := range pb.metadata {
		files, err := m.Files()
		if err != nil {
			return f, err
		}
		for name, content := range files {
			f[name] = content
		}
	}

	return f, nil
}

func (pb *PackageBuilder) AddMetadata(m metadata.Metadata) {
	pb.metadata = append(pb.metadata, m)
}

func (pb *PackageBuilder) AddMetadataType(metadataType string) error {
	metaFolder, err := pb.MetadataDir(metadataType)
	if err != nil {
		return fmt.Errorf("Could not get metadata directry: %w", err)
	}
	return pb.AddDirectory(metaFolder)
}

func (pb *PackageBuilder) AddMetadataItem(metadataType string, name string) error {
	metaFolder, err := pb.MetadataDir(metadataType)
	if err != nil {
		return fmt.Errorf("Could not get metadata directry: %w", err)
	}
	if filePath, err := findMetadataPath(metaFolder, name); err != nil {
		return fmt.Errorf("Could not find path for %s of type %s: %w", name, metadataType, err)
	} else {
		return pb.Add(filePath)
	}
}

func (pb *PackageBuilder) Add(path string) error {
	f, err := os.Stat(path)
	if err != nil {
		return err
	}
	if f.Mode().IsDir() {
		return pb.AddDirectory(path)
	} else {
		return pb.AddFile(path)
	}
}

func (pb *PackageBuilder) AddFile(fpath string) error {
	if lwcJsTestFile.MatchString(fpath) {
		// If this is a JS test file, just ignore it entirely,
		// don't consider it bad.
		return nil
	}
	m, err := metadata.MetadataFromPath(fpath)
	if err != nil {
		return fmt.Errorf("Could not add file: %w", err)
	}
	pb.AddMetadata(m)
	return nil
}

// AddDirectory Recursively add files contained in provided directory
func (pb *PackageBuilder) AddDirectory(fpath string) error {
	files, err := ioutil.ReadDir(fpath)
	if err != nil {
		return err
	}

	for _, f := range files {
		dirOrFilePath := fpath + "/" + f.Name()
		if strings.HasPrefix(f.Name(), ".") {
			Log.Info("Ignoring hidden file: " + dirOrFilePath)
			continue
		}

		if f.IsDir() {
			if lwcJsTestDir.MatchString(dirOrFilePath) {
				// Normally malformed paths would indicate invalid metadata,
				// but LWC tests should never be deployed. We may want to consider this logic/behavior,
				// such that we don't call `addFile` on directories in some cases; if we could
				// avoid the addFile call on the __tests__ dir, we could avoid this check.
				continue
			}
			err := pb.AddDirectory(dirOrFilePath)
			if err != nil {
				return err
			}
			continue
		}

		err = pb.AddFile(dirOrFilePath)
		if err != nil {
			return err
		}

	}
	return err
}

func (pb *PackageBuilder) MetadataDir(metadataType string) (path string, err error) {
	sourceDir, err := config.GetSourceDir()
	if err != nil {
		return "", fmt.Errorf("Could not identify source directory: %w", err)
	}

	for _, mp := range metapaths {
		if strings.ToLower(metadataType) == strings.ToLower(string(mp.name)) {
			return filepath.Join(sourceDir, string(mp.path)), nil
		}
	}
	return "", fmt.Errorf("Unknown metadata type: %s", metadataType)
}

// Get the path to a metadata file from the source folder and metadata name
func findMetadataPath(folder string, metadataName string) (string, error) {
	info, err := os.Stat(folder)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Invalid directory %s", folder)
	}
	filePath := ""
	err = filepath.Walk(folder, func(path string, f os.FileInfo, err error) error {
		ext := filepath.Ext(f.Name())
		if err != nil {
			Log.Info("Error looking for metadata: " + err.Error())
			return nil
		}
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			return err
		}
		if strings.ToLower(strings.TrimSuffix(rel, ext)) == strings.ToLower(metadataName) {
			filePath = path
		}
		return nil
	})
	if err != nil {
		Log.Info("Error looking for metadata: " + err.Error())
		return "", err
	}
	if filePath == "" {
		return "", fmt.Errorf("Failed to find %s in %s", metadataName, folder)
	}
	return filePath, nil
}

var lwcJsTestFile = regexp.MustCompile(".*\\.test\\.js$")
var lwcJsTestDir = regexp.MustCompile(fmt.Sprintf("%s__tests__$", regexp.QuoteMeta(string(os.PathSeparator))))
