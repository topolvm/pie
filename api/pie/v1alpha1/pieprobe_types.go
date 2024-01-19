package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PieProbeSpec defines the desired state of PieProbe
type PieProbeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of PieProbe. Edit pieprobe_types.go to remove/update
	Foo string `json:"foo,omitempty"`
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
