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

	// FeatureGates contains feature gates for the control plane.
	FeatureGates ControlPlaneFeatures `json:"featureGates,omitempty"`

	// CustomDefaultStorageClass
	CustomDefaultStorageClass *CustomDefaultStorageClass `json:"customDefaultStorageClass,omitempty"`

	// NetworkAccessType defines how the cluster can reach external networks.
	// +optional
	NetworkAccessType *NetworkAccessType `json:"networkAccessType,omitempty"`
}

// CustomDefaultStorageClass defines the custom storageclass which should be set as default
// This applies only to storageClasses managed by metal-stack.
// If set to nil, our default storageClass (e.g. csi-lvm) is set as default
type CustomDefaultStorageClass struct {
	// ClassName name of the storageclass to be set as default
	// If you want to have your own SC be set as default, set classname to ""
	ClassName string `json:"className"`
}

// ControlPlaneFeatures contains feature gates for the control plane.
type ControlPlaneFeatures struct {
	// MachineControllerManagerOOT enables the deployment of the out-of-tree machine controller manager.
	// Once enabled this cannot be taken back.
	// Deprecated: This is now default and always on. Toggle does not have an effect anymore.
	// +optional
	MachineControllerManagerOOT *bool `json:"machineControllerManagerOOT,omitempty"`
	// ClusterAudit enables the deployment of a non-null audit policy to the apiserver and the forwarding
	// of the audit events into the cluster where they appear as container log of an audittailer pod, where they
	// can be picked up by any of the available Kubernetes logging solutions.
	// +optional
	ClusterAudit *bool `json:"clusterAudit,omitempty"`
	// AuditToSplunk enables the forwarding of the apiserver auditlog to a defined splunk instance in addition to
	// forwarding it into the cluster. Needs the clusterAudit featureGate to be active.
	// +optional
	AuditToSplunk *bool `json:"auditToSplunk,omitempty"`
	// DurosStorageEncryption enables the deployment of configured encrypted storage classes for the duros-controller.
	// +optional
	DurosStorageEncryption *bool `json:"durosStorageEncryption,omitempty"`
	// RestrictEgress limits the cluster egress to the API server and necessary external dependencies (like container registries)
	// by using DNS egress policies.
	// Requires firewall-controller >= 1.2.0.
	// +optional
	RestrictEgress *bool `json:"restrictEgress,omitempty"`
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
type (
	// NetworkAccessType defines how a cluster is capable of accessing external networks
	NetworkAccessType string
)

const (
	// NetworkAccessBaseline allows the cluster to access external networks in a baseline manner
	NetworkAccessBaseline = NetworkAccessType("baseline")
	// NetworkAccessRestricted access to external networks is by default restricted to registries, dns and ntp to partition only destinations.
	// Therefor registries, dns and ntp destinations must be specified in the cloud-profile accordingly-
	// If this is not the case, restricting the access must not be possible.
	// Image overrides for all images which are required to create such a shoot, must be specified. No other images are provided in the given registry.
	// customers can define own rules to access external networks as in the baseline.
	// Service type loadbalancers are also not restricted.
	NetworkAccessRestricted = NetworkAccessType("restricted")
	// NetworkAccessForbidden in this configuration a customer can no longer create rules to access external networks.
	// which are outside of a given list of allowed networks. This is enforced by the firewall.
	// Service type loadbalancers are also not possible to open a service ip which is not in the list of allowed networks.
	// This is also enforced by the firewall.
	NetworkAccessForbidden = NetworkAccessType("baseline")
)
