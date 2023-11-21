package metadata

func init() {
	Registry.Register("ApexClass", createMetadataFunc(NewApexClass))
	Registry.Register("CustomTab", createMetadataFunc(NewCustomTab))
	Registry.Register("ReportFolder", createMetadataFunc(NewReportFolder))
	Registry.Register("Report", createMetadataFunc(NewReport))
	Registry.Register("SharingRules", createMetadataFunc(NewSharingRules))
}

func createMetadataFunc(constructor func(string) Metadata) MetadataCreateFunc {
	return func(path string) Metadata {
		return constructor(path)
	}
}
