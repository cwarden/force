package metadata_test

import (
	"io/ioutil"
	"os"

	. "github.com/ForceCLI/force/lib/metadata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	var (
		tempDir string
	)
	BeforeEach(func() {
		tempDir, _ = ioutil.TempDir("", "metadata-test")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})
	Context("Files", func() {
		It("should identify tabs", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			tabPath := tempDir + "/src/tabs/MyTab.tab"
			tabContents := `
<?xml version="1.0" encoding="UTF-8"?>
<CustomTab xmlns="http://soap.sforce.com/2006/04/metadata">
	<frameHeight>900</frameHeight>
	<hasSidebar>false</hasSidebar>
	<label>Palo</label>
	<motif>Custom41: Stack of Cash</motif>
	<urlEncodingKey>UTF-8</urlEncodingKey>
</CustomTab>
`
			ioutil.WriteFile(tabPath, []byte(tabContents), 0644)
			Expect(IsDeployable(tabPath)).To(Equal(true))
			m, err := MetadataFromPath(tabPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BaseMetadata{}))
			Expect(m.Name()).To(Equal("MyTab"))
			Expect(m.DeployedType()).To(Equal("CustomTab"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["tabs/MyTab.tab"] = []byte(tabContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should handle source format", func() {
			os.MkdirAll(tempDir+"/sfdx/main/default/tabs", 0755)
			tabPath := tempDir + "/sfdx/main/default/tabs/MyTab.tab-meta.xml"
			tabContents := `
<?xml version="1.0" encoding="UTF-8"?>
<CustomTab xmlns="http://soap.sforce.com/2006/04/metadata">
	<frameHeight>900</frameHeight>
	<hasSidebar>false</hasSidebar>
	<label>Palo</label>
	<motif>Custom41: Stack of Cash</motif>
	<urlEncodingKey>UTF-8</urlEncodingKey>
</CustomTab>
`
			ioutil.WriteFile(tabPath, []byte(tabContents), 0644)
			Expect(IsDeployable(tabPath)).To(Equal(true))
			m, err := MetadataFromPath(tabPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BaseMetadata{}))
			Expect(m.Name()).To(Equal("MyTab"))
			Expect(m.DeployedType()).To(Equal("CustomTab"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["tabs/MyTab.tab-meta.xml"] = []byte(tabContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should handle custom metadata", func() {
			os.MkdirAll(tempDir+"/src/customMetadata", 0755)
			customMetadataPath := tempDir + "/src/customMetadata/My_Type.My_Record.md"
			customMetadataContents := `
<?xml version="1.0" encoding="UTF-8"?>
<CustomMetadata xmlns="http://soap.sforce.com/2006/04/metadata" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
</CustomMetadata>
`
			ioutil.WriteFile(customMetadataPath, []byte(customMetadataContents), 0644)
			Expect(IsDeployable(customMetadataPath)).To(Equal(true))
			m, err := MetadataFromPath(customMetadataPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BaseMetadata{}))
			Expect(m.Name()).To(Equal("My_Type.My_Record"))
			Expect(m.DeployedType()).To(Equal("CustomMetadata"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["customMetadata/My_Type.My_Record.md"] = []byte(customMetadataContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should not identify directories", func() {
			Expect(IsDeployable(tempDir)).To(Equal(false))
			_, err := MetadataFromPath(tempDir)
			Expect(err).To(Not(BeNil()))
		})

		It("should not identify unknown types", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			unknownPath := tempDir + "/src/tabs/Unknown.tab"
			unknownContents := `
<?xml version="1.0" encoding="UTF-8"?>
<Unknown xmlns="http://soap.sforce.com/2006/04/metadata">
</Unknown>
`
			ioutil.WriteFile(unknownPath, []byte(unknownContents), 0644)
			Expect(IsDeployable(unknownPath)).To(Equal(false))
			_, err := MetadataFromPath(unknownPath)
			Expect(err).To(Not(BeNil()))
		})

		It("should not identify non-xml", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			nonXmlPath := tempDir + "/src/tabs/Unknown.tab"
			nonXmlContents := `junk`
			ioutil.WriteFile(nonXmlPath, []byte(nonXmlContents), 0644)
			Expect(IsDeployable(nonXmlPath)).To(Equal(false))
			_, err := MetadataFromPath(nonXmlPath)
			Expect(err).To(MatchError(MetadataFileNotFound))
		})

		It("should not mis-identify non-xml", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			nonXmlPath := tempDir + "/src/tabs/Unknown.tab"
			nonXmlContents := `
@IsTest
public class Group_Test {
	@TestSetup
	static void testData() {
		insert new List<Group>{ new Group(Name = 'Test'); };
	}
}
`
			ioutil.WriteFile(nonXmlPath, []byte(nonXmlContents), 0644)
			Expect(IsDeployable(nonXmlPath)).To(Equal(false))
			_, err := MetadataFromPath(nonXmlPath)
			Expect(err).To(MatchError(MetadataFileNotFound))
		})
	})

	Context("Files With Separate Metadata", func() {
		Context("In Metadata Format", func() {
			It("should identify classes", func() {
				os.MkdirAll(tempDir+"/src/classes", 0755)
				classPath := tempDir + "/src/classes/MyClass.cls"
				classContents := `
public class MyClass {}
`
				classMetaPath := tempDir + "/src/classes/MyClass.cls-meta.xml"
				classMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<ApexClass xmlns="http://soap.sforce.com/2006/04/metadata">
	 <apiVersion>59.0</apiVersion>
	 <status>Active</status>
</ApexClass>
`
				ioutil.WriteFile(classPath, []byte(classContents), 0644)
				ioutil.WriteFile(classMetaPath, []byte(classMetaContents), 0644)

				Expect(IsDeployable(classPath)).To(Equal(false))
				Expect(IsDeployable(classMetaPath)).To(Equal(true))

				m, err := MetadataFromPath(classPath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))
				m, err = MetadataFromPath(classMetaPath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))

				Expect(m.Name()).To(Equal("MyClass"))
				Expect(m.DeployedType()).To(Equal("ApexClass"))

				expectedMap := make(ForceMetadataFiles)
				expectedMap["classes/MyClass.cls"] = []byte(classContents)
				expectedMap["classes/MyClass.cls-meta.xml"] = []byte(classMetaContents)
				Expect(m.Files()).To(Equal(expectedMap))
			})

			It("should identify static resources", func() {
				os.MkdirAll(tempDir+"/src/staticresources", 0755)
				resourcePath := tempDir + "/src/staticresources/MyText.resource"
				resourceContents := `
MyText File
`
				resourceMetaPath := tempDir + "/src/staticresources/MyText.resource-meta.xml"
				resourceMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<StaticResource xmlns="http://soap.sforce.com/2006/04/metadata">
	 <cacheControl>Private</cacheControl>
	 <contentType>text/plain</contentType>
</StaticResource>
`
				ioutil.WriteFile(resourcePath, []byte(resourceContents), 0644)
				ioutil.WriteFile(resourceMetaPath, []byte(resourceMetaContents), 0644)

				Expect(IsDeployable(resourcePath)).To(Equal(false))
				Expect(IsDeployable(resourceMetaPath)).To(Equal(true))

				m, err := MetadataFromPath(resourcePath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))
				m, err = MetadataFromPath(resourceMetaPath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))

				Expect(m.Name()).To(Equal("MyText"))
				Expect(m.DeployedType()).To(Equal("StaticResource"))

				expectedMap := make(ForceMetadataFiles)
				expectedMap["staticresources/MyText.resource"] = []byte(resourceContents)
				expectedMap["staticresources/MyText.resource-meta.xml"] = []byte(resourceMetaContents)
				Expect(m.Files()).To(Equal(expectedMap))
			})
		})
		Context("In Source Format", func() {
			It("should identify static resources", func() {
				os.MkdirAll(tempDir+"/sfdx/main/default/staticresources", 0755)
				resourcePath := tempDir + "/sfdx/main/default/staticresources/MyText.txt"
				resourceContents := `
MyText File
`
				resourceMetaPath := tempDir + "/sfdx/main/default/staticresources/MyText.resource-meta.xml"
				resourceMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<StaticResource xmlns="http://soap.sforce.com/2006/04/metadata">
	 <cacheControl>Private</cacheControl>
	 <contentType>text/plain</contentType>
</StaticResource>
`
				ioutil.WriteFile(resourcePath, []byte(resourceContents), 0644)
				ioutil.WriteFile(resourceMetaPath, []byte(resourceMetaContents), 0644)

				Expect(IsDeployable(resourcePath)).To(Equal(false))
				Expect(IsDeployable(resourceMetaPath)).To(Equal(true))

				m, err := MetadataFromPath(resourcePath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))
				m, err = MetadataFromPath(resourceMetaPath)
				Expect(err).To(BeNil())
				Expect(m).To(BeAssignableToTypeOf(&ContentMetadata{}))

				Expect(m.Name()).To(Equal("MyText"))
				Expect(m.DeployedType()).To(Equal("StaticResource"))

				expectedMap := make(ForceMetadataFiles)
				expectedMap["staticresources/MyText.resource-meta.xml"] = []byte(resourceMetaContents)
				expectedMap["staticresources/MyText.resource"] = []byte(resourceContents)
				Expect(m.Files()).To(Equal(expectedMap))
			})
		})
	})

	Context("Folder with Separate Metadata", func() {
		It("should identify folders", func() {
			folderPath := tempDir + "/src/reports/MyFolder"
			os.MkdirAll(folderPath, 0755)
			folderMetaPath := tempDir + "/src/reports/MyFolder-meta.xml"
			folderMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<ReportFolder xmlns="http://soap.sforce.com/2006/04/metadata">
	<folderShares>
		<accessLevel>Manage</accessLevel>
		<sharedTo>System_Administrator</sharedTo>
		<sharedToType>Role</sharedToType>
	</folderShares>
	<name>My Folder</name>
</ReportFolder>
`
			ioutil.WriteFile(folderMetaPath, []byte(folderMetaContents), 0644)

			Expect(IsDeployable(folderMetaPath)).To(Equal(true))
			Expect(IsDeployable(folderPath)).To(Equal(false))
			m, err := MetadataFromPath(folderMetaPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&FolderedMetadata{}))

			Expect(m.Name()).To(Equal("MyFolder"))
			Expect(m.DeployedType()).To(Equal("Report"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["reports/MyFolder-meta.xml"] = []byte(folderMetaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
		It("should support nested folders", func() {
			folderPath := tempDir + "/src/reports/MyFolder/MySubfolder"
			os.MkdirAll(folderPath, 0755)
			folderMetaPath := tempDir + "/src/reports/MyFolder/MySubfolder-meta.xml"
			folderMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<ReportFolder xmlns="http://soap.sforce.com/2006/04/metadata">
	<folderShares>
		<accessLevel>Manage</accessLevel>
		<sharedTo>System_Administrator</sharedTo>
		<sharedToType>Role</sharedToType>
	</folderShares>
	<name>My Folder</name>
</ReportFolder>
`
			ioutil.WriteFile(folderMetaPath, []byte(folderMetaContents), 0644)

			Expect(IsDeployable(folderMetaPath)).To(Equal(true))
			Expect(IsDeployable(folderPath)).To(Equal(false))
			m, err := MetadataFromPath(folderMetaPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&FolderedMetadata{}))

			Expect(m.Name()).To(Equal("MyFolder/MySubfolder"))
			Expect(m.DeployedType()).To(Equal("Report"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["reports/MyFolder/MySubfolder-meta.xml"] = []byte(folderMetaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
	})

	Context("Reports", func() {
		It("should should include folder in path", func() {
			folderPath := tempDir + "/src/reports/MyFolder"
			os.MkdirAll(folderPath, 0755)
			reportPath := tempDir + "/src/reports/MyFolder/MyReport.report"
			reportContents := `
<?xml version="1.0" encoding="UTF-8"?>
<Report xmlns="http://soap.sforce.com/2006/04/metadata">
</Report>
`
			ioutil.WriteFile(reportPath, []byte(reportContents), 0644)

			Expect(IsDeployable(reportPath)).To(Equal(true))

			m, err := MetadataFromPath(reportPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&FolderedMetadata{}))

			Expect(m.Name()).To(Equal("MyFolder/MyReport"))
			Expect(m.DeployedType()).To(Equal("Report"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["reports/MyFolder/MyReport.report"] = []byte(reportContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
		It("should support nested folders", func() {
			folderPath := tempDir + "/src/reports/MyFolder/MySubfolder"
			os.MkdirAll(folderPath, 0755)
			reportPath := tempDir + "/src/reports/MyFolder/MySubfolder/MyReport.report"
			reportContents := `
<?xml version="1.0" encoding="UTF-8"?>
<Report xmlns="http://soap.sforce.com/2006/04/metadata">
</Report>
`
			ioutil.WriteFile(reportPath, []byte(reportContents), 0644)

			Expect(IsDeployable(reportPath)).To(Equal(true))
			m, err := MetadataFromPath(reportPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&FolderedMetadata{}))

			Expect(m.Name()).To(Equal("MyFolder/MySubfolder/MyReport"))
			Expect(m.DeployedType()).To(Equal("Report"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["reports/MyFolder/MySubfolder/MyReport.report"] = []byte(reportContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
	})

	Context("Destructive Changes", func() {
		It("should be deployed without path", func() {
			folderPath := tempDir + "/src"
			os.MkdirAll(folderPath, 0755)
			destructiveChangesPath := tempDir + "/src/destructiveChanges.xml"
			destructiveChangesContents := `
<?xml version="1.0" encoding="UTF-8"?>
<Package xmlns="http://soap.sforce.com/2006/04/metadata">
</Package>
`
			ioutil.WriteFile(destructiveChangesPath, []byte(destructiveChangesContents), 0644)

			Expect(IsDeployable(destructiveChangesPath)).To(Equal(true))

			m, err := DeployableFromPath(destructiveChangesPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&Package{}))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["destructiveChanges.xml"] = []byte(destructiveChangesContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
	})
})
