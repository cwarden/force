package main_test

import (
	. "github.com/heroku/force"
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("IsSubdirectory", func() {
		It("should check whether subdirectory is within base", func() {
			Expect(IsSubdirectory("/path/to/src", "/path/to/src")).To(BeTrue())
			Expect(IsSubdirectory("/path/to/root", "/path/to/root/src")).To(BeTrue())
			Expect(IsSubdirectory("/path/to/root", "/path/to/src")).To(BeFalse())
			Expect(IsSubdirectory("/path/to/root", "/src")).To(BeFalse())
			Expect(IsSubdirectory("/", "/src")).To(BeTrue())
		})
	})

	Describe("GetSourceDir", func() {
		var (
			tempDir string
		)

		BeforeEach(func() {
			tempDir, _ = ioutil.TempDir("", "config-test")
		})

		AfterEach(func() {
			os.RemoveAll(tempDir)
		})

		Context("when a src dir exists in path", func() {

			BeforeEach(func() {
				os.MkdirAll(tempDir+"/src/classes", 0755)
				os.MkdirAll(tempDir+"/src/reports/MyReports", 0755)
			})

			It("should find src dir in current directory", func() {
				os.Chdir(tempDir)
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/src"))
			})
			It("should find src dir that is current directory", func() {
				os.Chdir(tempDir + "/src")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/src"))
			})
			It("should find src dir that is parent of current directory", func() {
				os.Chdir(tempDir + "/src/classes")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/src"))
			})
			It("should find src dir that is grandparent of current directory", func() {
				os.Chdir(tempDir + "/src/reports/MyReports")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/src"))
			})
			It("should find not find src dir that is great-grandparent of current directory", func() {
				os.MkdirAll(tempDir+"/src/reports/MyReports/tooDeep", 0755)
				os.Chdir(tempDir + "/src/reports/MyReports/tooDeep")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/src/reports/MyReports/tooDeep/metadata"))
			})
		})

		Context("when a metadata dir exists in path", func() {

			BeforeEach(func() {
				os.MkdirAll(tempDir+"/metadata/classes", 0755)
				os.MkdirAll(tempDir+"/metadata/reports/MyReports", 0755)
			})

			It("should find metadata dir in current directory", func() {
				os.Chdir(tempDir)
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
			})
			It("should find metadata dir that is current directory", func() {
				os.Chdir(tempDir + "/metadata")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
			})
			It("should find metadata dir that is parent of current directory", func() {
				os.Chdir(tempDir + "/metadata/classes")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
			})
			It("should find metadata dir that is grandparent of current directory", func() {
				os.Chdir(tempDir + "/metadata/reports/MyReports")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
			})
			It("should find not find metadata dir that is great-grandparent of current directory", func() {
				os.MkdirAll(tempDir+"/metadata/reports/MyReports/tooDeep", 0755)
				os.Chdir(tempDir + "/metadata/reports/MyReports/tooDeep")
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata/reports/MyReports/tooDeep/metadata"))
			})
		})

		Context("when neither a src nor metadata dir exists in path", func() {

			It("should create a metadata symlink in current directory", func() {
				os.Chdir(tempDir)
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
				_, err := os.Stat(dir)
				Expect(err).To(BeNil())
			})
			It("should create a src dir in current directory", func() {
				os.Chdir(tempDir)
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
				_, err := os.Stat(tempDir + "/src")
				Expect(err).To(BeNil())
			})
			It("should return metadata symlink in current directory", func() {
				os.Chdir(tempDir)
				dir, _ := GetSourceDir()
				Expect(dir).To(Equal(tempDir + "/metadata"))
			})
		})
	})
})
