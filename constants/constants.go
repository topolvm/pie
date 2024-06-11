package constants

const (
	ProvisionProbeNamePrefix = "provision"
	MountProbeNamePrefix     = "mount"
	ProbeContainerName       = "probe"
	PodFinalizerName         = "pie.topolvm.io/pod"
	PVCNamePrefix            = "pie-pvc"

	ProbeNodeLabelKey         = "node"
	ProbeStorageClassLabelKey = "storage-class"
	ProbePieProbeLabelKey     = "pie-probe"
)
