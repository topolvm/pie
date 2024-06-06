package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PieProbeSpec defines the desired state of PieProbe
type PieProbeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="monitoringStorageClass is immutable"
	MonitoringStorageClass string `json:"monitoringStorageClass"`

	NodeSelector corev1.NodeSelector `json:"nodeSelector"`

	//+kubebuilder:validation:Maximum:=59
	//+kubebuilder:validation:Minimum:=1
	ProbePeriod int `json:"probePeriod"`

	ProbeThreshold metav1.Duration `json:"probeThreshold"`
}

// PieProbeStatus defines the observed state of PieProbe
type PieProbeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PieProbe is the Schema for the pieprobes API
type PieProbe struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PieProbeSpec   `json:"spec,omitempty"`
	Status PieProbeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PieProbeList contains a list of PieProbe
type PieProbeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PieProbe `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PieProbe{}, &PieProbeList{})
}
