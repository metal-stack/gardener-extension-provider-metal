package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControlPlaneConfig contains configuration settings for the control plane.
type ControlPlaneConfig struct {
	metav1.TypeMeta `json:",inline"`

	// CloudControllerManager contains configuration settings for the cloud-controller-manager.
	// +optional
	CloudControllerManager *CloudControllerManagerConfig `json:"cloudControllerManager,omitempty"`

	// IAMConfig contains the config for all AuthN/AuthZ related components
	IAMConfig *IAMConfig `json:"iamconfig" optional:"false"`

	// FeatureGates contains feature gates for the control plane.
	FeatureGates ControlPlaneFeatures `json:"featureGates,omitempty"`
}

// ControlPlaneFeatures contains feature gates for the control plane.
type ControlPlaneFeatures struct {
	// MachineControllerManagerOOT enables the deployment of the out-of-tree machine controller manager.
	// Once enabled this cannot be taken back.
	// This will become default at some point in the future.
	// +optional
	MachineControllerManagerOOT *bool `json:"machineControllerManagerOOT,omitempty"`
}

// CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
type CloudControllerManagerConfig struct {
	// FeatureGates contains information about enabled feature gates.
	// +optional
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
	// DefaultExternalNetwork explicitly defines the network from which the CCM allocates IPs for services of type load balancer
	// If not defined, it will use the last network with the default external network tag from the infrastructure firewall networks
	// Networks not derived from a private super network have precedence.
	// +optional
	DefaultExternalNetwork *string `json:"defaultExternalNetwork" optional:"true"`
}
