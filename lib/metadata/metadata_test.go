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
		})

		It("should not identify directories", func() {
			Expect(IsMetadata(tempDir)).To(Equal(false))
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
		})

		It("should not identify non-xml", func() {
			os.MkdirAll(tempDir+"/src/tabs", 0755)
			nonXmlPath := tempDir + "/src/tabs/Unknown.tab"
			nonXmlContents := `junk`
			ioutil.WriteFile(nonXmlPath, []byte(nonXmlContents), 0644)
			Expect(IsMetadata(nonXmlPath)).To(Equal(false))
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
		})
	})
})
