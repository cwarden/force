package main_test

import (
	"encoding/xml"
	. "github.com/heroku/force"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type MockForceMetadata struct {
	ForceMetadata
}

type stubSoapExecuter struct {
	action string
	query  string
}

func (fm *stubSoapExecuter) SoapExecute(action, query string) (response []byte, err error) {
	fm.action = action
	fm.query = "<query>" + query + "</query>"
	switch action {
	case "readMetadata":
		response = append(response, "readMetadata"...)
	}
	return
}

var _ = Describe("Metadata", func() {
	var (
		stub stubSoapExecuter
	)

	BeforeEach(func() {
		stub = stubSoapExecuter{}
	})

	Describe("ReadMetadata", func() {
		It("should call readMetadata SOAP method", func() {
			ReadMetadata(&stub, "ReportFolder", []string{"Test"})
			Expect(stub.action).To(Equal("readMetadata"))
		})
		It("should read metadata for the passed type", func() {
			ReadMetadata(&stub, "ReportFolder", []string{"Test"})
			var query struct {
				MetadataType string `xml:"metadataType"`
			}
			xml.Unmarshal([]byte(stub.query), &query)
			Expect(query.MetadataType).To(Equal("ReportFolder"))
		})
		It("should read metadata for the objects", func() {
			ReadMetadata(&stub, "ReportFolder", []string{"Test", "Test2"})
			var query struct {
				Objects []string `xml:"fullNames"`
			}
			xml.Unmarshal([]byte(stub.query), &query)
			Expect(query.Objects).To(HaveLen(2))
			Expect(query.Objects).To(ContainElement("Test"))
			Expect(query.Objects).To(ContainElement("Test2"))
		})
		It("should return SOAP call results", func() {
			method, _ := ReadMetadata(&stub, "ReportFolder", []string{"Test"})
			Expect(string(method)).To(Equal("readMetadata"))
		})
	})

})
