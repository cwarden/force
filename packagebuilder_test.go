package main_test

import (
	. "github.com/heroku/force"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Packagebuilder", func() {
	Describe("NewPushBuilder", func() {
		It("should return a Packagebuilder", func() {
			pb := NewPushBuilder()
			Expect(pb).To(BeAssignableToTypeOf(PackageBuilder{IsPush: true}))
		})
	})

	Describe("AddFile", func() {
		var (
			pb      PackageBuilder
			tempDir string
		)

		BeforeEach(func() {
			pb = NewPushBuilder()
			tempDir, _ = ioutil.TempDir("", "packagebuilder-test")
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		Context("when adding a metadata file", func() {
			var apexClassPath string

			BeforeEach(func() {
				os.MkdirAll(tempDir+"/src/classes", 0755)
				apexClassPath = tempDir + "/src/classes/Test.cls"
				apexClassContents := "class Test {}"
				ioutil.WriteFile(apexClassPath, []byte(apexClassContents), 0644)
			})

			It("should add the file to package", func() {
				_, err := pb.AddFile(apexClassPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(pb.Files).To(HaveKey("classes/Test.cls"))
			})
			It("should add the file to the package.xml", func() {
				pb.AddFile(apexClassPath)
				Expect(pb.Metadata).To(HaveKey("ApexClass"))
				Expect(pb.Metadata["ApexClass"].Members[0]).To(Equal("Test"))
			})
		})

		Context("when adding a folder", func() {
			var reportFolderPath string

			BeforeEach(func() {
				reportFolderPath = tempDir + "/src/reports/Test"
				os.MkdirAll(reportFolderPath, 0755)
				reportFolderMetaPath := reportFolderPath + "-meta.xml"
				reportFolderMetaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<ReportFolder xmlns="http://soap.sforce.com/2006/04/metadata">
	<name>Test</name>
</ReportFolder>`
				ioutil.WriteFile(reportFolderMetaPath, []byte(reportFolderMetaContents), 0644)
			})

			It("should add the folder metadata to package", func() {
				_, err := pb.AddFile(reportFolderPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(pb.Files).To(HaveKey("reports/Test-meta.xml"))
			})
			It("should add the folder to the package.xml", func() {
				pb.AddFile(reportFolderPath)
				Expect(pb.Metadata).To(HaveKey("Report"))
				Expect(pb.Metadata["Report"].Members[0]).To(Equal("Test"))
			})
		})

		Context("when adding a non-existent file", func() {
			It("should not add the file to package", func() {
				_, err := pb.AddFile(tempDir + "/no/such/file")
				Expect(err).To(HaveOccurred())
				Expect(pb.Files).To(BeEmpty())
			})
			It("should not add the file to the package.xml", func() {
				pb.AddFile(tempDir + "/no/such/file")
				Expect(pb.Metadata).To(BeEmpty())
			})
		})

		Context("when adding a destructiveChanges file", func() {
			var destructiveChangesPath string

			BeforeEach(func() {
				pb = NewPushBuilder()
				tempDir, _ := ioutil.TempDir("", "packagebuilder-test")
				destructiveChangesPath = tempDir + "/destructiveChanges.xml"
				destructiveChangesXml := `<?xml version="1.0" encoding="UTF-8"?>
					<Package xmlns="http://soap.sforce.com/2006/04/metadata">
					<version>34.0</version>
					</Package>
				`
				ioutil.WriteFile(destructiveChangesPath, []byte(destructiveChangesXml), 0644)
			})

			It("should add the file to package", func() {
				_, err := pb.AddFile(destructiveChangesPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(pb.Files).To(HaveKey("destructiveChanges.xml"))
			})
			It("should not add the file to the package.xml", func() {
				pb.AddFile(destructiveChangesPath)
				Expect(pb.Metadata).To(BeEmpty())
			})
		})
	})

	Describe("GetMetaForPath", func() {
		Context("when passed an Apex class", func() {
			It("should return the ApexClass metadata type", func() {
				metaName, _ := GetMetaForPath("/path/to/src/classes/Test.cls")
				Expect(metaName).To(Equal("ApexClass"))
			})
			It("should return the class name with extension", func() {
				_, objectName := GetMetaForPath("/path/to/src/classes/Test.cls")
				Expect(objectName).To(Equal("Test.cls"))
			})
		})
		Context("when passed an aura component", func() {
			It("should return the AuraDefinitionBundle metadata type", func() {
				metaName, _ := GetMetaForPath("/path/to/src/aura/Test/file.js")
				Expect(metaName).To(Equal("AuraDefinitionBundle"))
			})
			It("should return the aura folder name", func() {
				_, objectName := GetMetaForPath("/path/to/src/aura/Test/file.js")
				Expect(objectName).To(Equal("Test"))
			})
		})
		Context("when passed a folder", func() {
			It("should return the related metadata type", func() {
				metaName, _ := GetMetaForPath("/path/to/src/reports/Test")
				Expect(metaName).To(Equal("Report"))
			})
			It("should return the folder name", func() {
				_, objectName := GetMetaForPath("/path/to/src/reports/Test")
				Expect(objectName).To(Equal("Test"))
			})
		})
		Context("when passed a report", func() {
			It("should return the Report metadata type", func() {
				metaName, _ := GetMetaForPath("/path/to/src/reports/Test/My_Report.report")
				Expect(metaName).To(Equal("Report"))
			})
			It("should return the report name with folder", func() {
				_, objectName := GetMetaForPath("/path/to/src/reports/Test/My_Report.report")
				Expect(objectName).To(Equal("Test/My_Report.report"))
			})
		})
	})
})
