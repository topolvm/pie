package constants

const (
	ProbeNamePrefix           = "pie-probe"
	ProvisionProbeNamePrefix  = "provision"
	ProbeContainerName        = "probe"
	PodFinalizerName          = "pie.topolvm.io/pod"
	NodeFinalizerName         = "pie.topolvm.io/node"
	StorageClassFinalizerName = "pie.topolvm.io/storage-class"

	ProbeNodeLabelKey         = "node"
	ProbeStorageClassLabelKey = "storage-class"
	ProbePieProbeLabelKey     = "pie-probe"
)
