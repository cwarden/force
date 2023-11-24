package metadata

import fmd "github.com/ForceCLI/force-md/general"

type MergeableMetadata interface {
	DeployableMetadata
	ParentPath() string
	AddTo(fmd.Metadata) (fmd.Metadata, error)
}
