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
	// +optional
	IAMConfig *IAMConfig `json:"iamconfig,omitempty"`

	// FeatureGates contains feature gates for the control plane.
	FeatureGates ControlPlaneFeatures `json:"featureGates,omitempty"`

	// CustomDefaultStorageClass
	CustomDefaultStorageClass *CustomDefaultStorageClass `json:"customDefaultStorageClass,omitempty"`
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
	// RestrictedEgress limits the cluster egress to the API server and necessary external dependencies (like container registries)
	// by using DNS egress policies.
	// Requires firewall-controller >= 1.2.0.
	// +optional
	RestrictedEgress *bool `json:"restrictedEgress,omitempty"`
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
