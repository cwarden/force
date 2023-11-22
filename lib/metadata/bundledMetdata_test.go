package metadata_test

import (
	"io/ioutil"
	"os"

	. "github.com/ForceCLI/force/lib/metadata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bundled Metadata", func() {
	var (
		tempDir string
	)
	BeforeEach(func() {
		tempDir, _ = ioutil.TempDir("", "metadata-test")
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})
	Context("LWC", func() {
		It("should identify components", func() {
			os.MkdirAll(tempDir+"/src/lwc/MyComponent", 0755)
			jsPath := tempDir + "/src/lwc/MyComponent/MyComponent.js"
			jsContents := `export {}`
			metaPath := tempDir + "/src/lwc/MyComponent/MyComponent.js-meta.xml"
			metaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<LightningComponentBundle xmlns="http://soap.sforce.com/2006/04/metadata">
</LightningComponentBundle>
`
			ioutil.WriteFile(jsPath, []byte(jsContents), 0644)
			ioutil.WriteFile(metaPath, []byte(metaContents), 0644)
			Expect(IsDeployable(jsPath)).To(Equal(false))
			Expect(IsDeployable(metaPath)).To(Equal(true))

			m, err := MetadataFromPath(jsPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BundledMetadata{}))
			Expect(m.Name()).To(Equal("MyComponent"))
			Expect(m.DeployedType()).To(Equal("LightningComponentBundle"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["lwc/MyComponent/MyComponent.js"] = []byte(jsContents)
			expectedMap["lwc/MyComponent/MyComponent.js-meta.xml"] = []byte(metaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should handle html", func() {
			os.MkdirAll(tempDir+"/src/lwc/MyComponent", 0755)
			jsPath := tempDir + "/src/lwc/MyComponent/MyComponent.js"
			jsContents := `export {}`
			htmlPath := tempDir + "/src/lwc/MyComponent/MyComponent.html"
			htmlContents := `<template><div/></template>`
			metaPath := tempDir + "/src/lwc/MyComponent/MyComponent.js-meta.xml"
			metaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<LightningComponentBundle xmlns="http://soap.sforce.com/2006/04/metadata">
</LightningComponentBundle>
`
			ioutil.WriteFile(jsPath, []byte(jsContents), 0644)
			ioutil.WriteFile(metaPath, []byte(metaContents), 0644)
			ioutil.WriteFile(htmlPath, []byte(htmlContents), 0644)
			Expect(IsDeployable(htmlPath)).To(Equal(false))

			m, err := MetadataFromPath(htmlPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BundledMetadata{}))
			Expect(m.Name()).To(Equal("MyComponent"))
			Expect(m.DeployedType()).To(Equal("LightningComponentBundle"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["lwc/MyComponent/MyComponent.js"] = []byte(jsContents)
			expectedMap["lwc/MyComponent/MyComponent.html"] = []byte(htmlContents)
			expectedMap["lwc/MyComponent/MyComponent.js-meta.xml"] = []byte(metaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})

		It("should handle source format", func() {
			os.MkdirAll(tempDir+"/sfdx/main/default/lwc/MyComponent", 0755)
			jsPath := tempDir + "/sfdx/main/default/lwc/MyComponent/MyComponent.js"
			jsContents := `export {}`
			metaPath := tempDir + "/sfdx/main/default/lwc/MyComponent/MyComponent.js-meta.xml"
			metaContents := `
<?xml version="1.0" encoding="UTF-8"?>
<LightningComponentBundle xmlns="http://soap.sforce.com/2006/04/metadata">
</LightningComponentBundle>
`
			ioutil.WriteFile(jsPath, []byte(jsContents), 0644)
			ioutil.WriteFile(metaPath, []byte(metaContents), 0644)
			Expect(IsDeployable(jsPath)).To(Equal(false))
			Expect(IsDeployable(metaPath)).To(Equal(true))

			m, err := MetadataFromPath(jsPath)
			Expect(err).To(BeNil())
			Expect(m).To(BeAssignableToTypeOf(&BundledMetadata{}))
			Expect(m.Name()).To(Equal("MyComponent"))
			Expect(m.DeployedType()).To(Equal("LightningComponentBundle"))

			expectedMap := make(ForceMetadataFiles)
			expectedMap["lwc/MyComponent/MyComponent.js"] = []byte(jsContents)
			expectedMap["lwc/MyComponent/MyComponent.js-meta.xml"] = []byte(metaContents)
			Expect(m.Files()).To(Equal(expectedMap))
		})
	})
})
