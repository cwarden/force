package main_test

import (
	"encoding/xml"
	. "github.com/heroku/force"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

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

	Describe("ReadMetadataResponseToFolderMetadata", func() {
		var response []byte
		BeforeEach(func() {
			response = []byte(`<?xml version="1.0" encoding="UTF-8"?>
				<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://soap.sforce.com/2006/04/metadata" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
					<soapenv:Body>
						<readMetadataResponse>
							<result>
								<records xsi:type="ReportFolder">
									<fullName>My_Test</fullName>
									<folderShares>
										<accessLevel>View</accessLevel>
										<sharedTo>AllInternalUsers</sharedTo>
										<sharedToType>Organization</sharedToType>
									</folderShares>
									<name>My Test</name>
								</records>
							</result>
						</readMetadataResponse>
					</soapenv:Body>
				</soapenv:Envelope>`)
		})
		It("should extract folder metadata from response", func() {
			records, err := ReadMetadataResponseToFolderMetadata(response)
			Expect(err).ToNot(HaveOccurred())
			Expect(len(records)).To(Equal(1))
			record := records[0]
			Expect(record.FullName).To(Equal("My_Test"))
			Expect(record.Name).To(Equal("My Test"))
		})
		It("should extract folder share metadata from response", func() {
			records, _ := ReadMetadataResponseToFolderMetadata(response)
			record := records[0]
			Expect(len(record.FolderShares)).To(Equal(1))
			share := record.FolderShares[0]
			Expect(share.AccessLevel).To(Equal("View"))
			Expect(share.SharedTo).To(Equal("AllInternalUsers"))
			Expect(share.SharedToType).To(Equal("Organization"))
		})
		Describe("when an invalid folder is requested", func() {
			It("should fail", func() {
				badResponse := []byte(`<?xml version="1.0" encoding="UTF-8"?>
					<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns="http://soap.sforce.com/2006/04/metadata" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
						<soapenv:Body>
							<readMetadataResponse>
								<result>
									<records xsi:nil="true"/>
								</result>
							</readMetadataResponse>
						</soapenv:Body>
					</soapenv:Envelope>`)
				_, err := ReadMetadataResponseToFolderMetadata(badResponse)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("FolderMetadataFiles", func() {
	})

	Describe("RetrieveFolders", func() {
	})

})
