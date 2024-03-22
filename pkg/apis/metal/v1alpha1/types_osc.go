package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// ImageProviderConfig is stored in the OSC's provider config RawExtension
type ImageProviderConfig struct {
	// required to convert it to/from RawExtension
	metav1.TypeMeta `json:",inline"`
	// NetworkIsolation defines restricted/forbidden networkaccess for worker nodes
	NetworkIsolation *NetworkIsolation `json:"networkIsolation,omitempty"`
}
