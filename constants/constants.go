package constants

const (
	ProbeNamePrefix           = "pie-probe"
	ProvisionProbeNamePrefix  = "provision"
	MountProbeNamePrefix      = "mount"
	ProbeContainerName        = "probe"
	PodFinalizerName          = "pie.topolvm.io/pod"
	NodeFinalizerName         = "pie.topolvm.io/node"
	StorageClassFinalizerName = "pie.topolvm.io/storage-class"
	PVCNamePrefix             = "pie-pvc"

	ProbeNodeLabelKey         = "node"
	ProbeStorageClassLabelKey = "storage-class"
	ProbePieProbeLabelKey     = "pie-probe"
)
