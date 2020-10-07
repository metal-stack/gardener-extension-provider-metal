package v1alpha1

import (
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkerConfig contains configuration settings for the worker nodes.
type WorkerConfig struct {
	metav1.TypeMeta `json:",inline"`

	// FeatureGates contains feature gates for metal stack worker machines.
	FeatureGates WorkerFeatures `json:"featureGates,omitempty"`
}

// WorkerFeatures contains feature gates for metal stack worker machines.
type WorkerFeatures struct {
	// MachineControllerManagerOOT enables the deployment of the out-of-tree machine controller manager.
	// Once enabled this cannot be taken back.
	// This will become default at some point in the future.
	// +optional
	MachineControllerManagerOOT *bool `json:"machineControllerManagerOOT,omitempty"`
}

// WorkerStatus contains information about created worker resources.
type WorkerStatus struct {
	metav1.TypeMeta `json:",inline"`

	// MachineImages is a list of machine images that have been used in this worker. Usually, the extension controller
	// gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if
	// a version that is still in use gets removed from this componentconfig it cannot reconcile anymore existing `Worker`
	// resources that are still using this version. Hence, it stores the used versions in the provider status to ensure
	// reconciliation is possible.
	MachineImages []config.MachineImage `json:"machineImages,omitempty"`
}
