package metadata_test

import (
	"io/ioutil"
	"os"

	. "github.com/ForceCLI/force/lib/metadata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("IsMetadata", func() {
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
			Expect(IsMetadata(tabPath)).To(Equal(true))
			m, err := MetadataFromPath(tabPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&CustomTab{}))
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
			Expect(IsMetadata(tabPath)).To(Equal(true))
			m, err := MetadataFromPath(tabPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&CustomTab{}))
			Expect(m.Name()).To(Equal("MyTab"))
			Expect(m.DeployedType()).To(Equal("CustomTab"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["tabs/MyTab.tab"] = []byte(tabContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should not identify directories", func() {
			Expect(IsMetadata(tempDir)).To(Equal(false))
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
			Expect(IsMetadata(unknownPath)).To(Equal(false))
			_, err := MetadataFromPath(unknownPath)
			Expect(err).To(Not(BeNil()))
		})

		It("should not identify non-xml", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			nonXmlPath := tempDir + "/src/tabs/Unknown.tab"
			nonXmlContents := `junk`
			ioutil.WriteFile(nonXmlPath, []byte(nonXmlContents), 0644)
			Expect(IsMetadata(nonXmlPath)).To(Equal(false))
			_, err := MetadataFromPath(nonXmlPath)
			Expect(err).To(Equal(MetadataFileNotFound))
		})
	})

	Context("Files With Separate Metadata", func() {
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

			Expect(IsMetadata(classPath)).To(Equal(false))
			Expect(IsMetadata(classMetaPath)).To(Equal(true))
			Expect(HasRelatedMetadata(classPath)).To(Equal(true))
			Expect(HasRelatedMetadata(classMetaPath)).To(Equal(false))

			m, err := MetadataFromPath(classPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&ApexClass{}))
			m, err = MetadataFromPath(classMetaPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&ApexClass{}))

			Expect(m.Name()).To(Equal("MyClass"))
			Expect(m.DeployedType()).To(Equal("ApexClass"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["classes/MyClass.cls"] = []byte(classContents)
			expectedMap["classes/MyClass.cls-meta.xml"] = []byte(classMetaContents)
			Expect(m.Files()).To(Equal(expectedMap))
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

			Expect(IsMetadata(folderMetaPath)).To(Equal(true))
			Expect(IsMetadata(folderPath)).To(Equal(false))
			Expect(HasRelatedMetadata(folderPath)).To(Equal(true))
			Expect(HasRelatedMetadata(folderMetaPath)).To(Equal(false))
			m, err := MetadataFromPath(folderMetaPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&ReportFolder{}))

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

			Expect(IsMetadata(folderMetaPath)).To(Equal(true))
			Expect(IsMetadata(folderPath)).To(Equal(false))
			Expect(HasRelatedMetadata(folderPath)).To(Equal(true))
			Expect(HasRelatedMetadata(folderMetaPath)).To(Equal(false))
			m, err := MetadataFromPath(folderMetaPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&ReportFolder{}))

			Expect(m.Name()).To(Equal("MySubfolder"))
			Expect(m.DeployedType()).To(Equal("Report"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["reports/MyFolder/MySubfolder-meta.xml"] = []byte(folderMetaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
	})
})
