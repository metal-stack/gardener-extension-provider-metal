package config

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the metal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection *configv1alpha1.ClientConnectionConfiguration

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to metal-specific identifiers, i.e. AMIs.
	MachineImages []MachineImage

	// FirewallInternalPrefixes is a list of prefixes for the firewall-controller
	// which will be counted as internal network traffic. this is important for accounting
	// networking traffic.
	FirewallInternalPrefixes []string

	// ETCD is the etcd configuration.
	ETCD ETCD

	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig

	// Storage is the configuration for storage.
	Storage StorageConfiguration

	// ImagePullPolicy defines the pull policy for the components deployed through the control plane controller.
	// Defaults to IfNotPresent if empty or unknown.
	ImagePullPolicy string

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret
}

// MachineImage is a mapping from logical names and versions to GCP-specific identifiers.
type MachineImage struct {
	// Name is the logical name of the machine image.
	Name string
	// Version is the logical version of the machine image.
	Version string
	// Image is the path to the image.
	Image string
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	ClassName *string
	// Capacity is the storage capacity used in etcd-main volume claims.
	Capacity *resource.Quantity
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	Schedule *string
	// DeltaSnapshotPeriod is the time for delta snapshots to be made
	DeltaSnapshotPeriod *string
}

// StorageConfiguration contains the configuration for provider specific storage solutions.
type StorageConfiguration struct {
	// Duros contains the configuration for duros cloud storage
	Duros DurosConfiguration
}

// DurosConfiguration contains the configuration for lightbits duros storage.
type DurosConfiguration struct {
	// Enabled enables duros storage when set to true.
	Enabled bool
	// PartitionConfig is a map of a partition id to the duros partition configuration
	PartitionConfig map[string]DurosPartitionConfiguration
}

// DurosPartitionConfiguration is the configuration for duros for a particular partition
type DurosPartitionConfiguration struct {
	// Endpoints is the list of endpoints for the storage data plane and control plane communication
	Endpoints []string
	// AdminKey is the key used for generating storage credentials
	AdminKey string
	// AdminToken is the token used by the duros-controller to authenticate against the duros API
	AdminToken string
	// StorageClasses contain information on the storage classes that the duros-controller creates in the shoot cluster
	StorageClasses []DurosSeedStorageClass

	// APIEndpoint is the endpoint used for control plane network communication.
	APIEndpoint string
	// APICA is the ca of the client cert to access the api endpoint
	APICA string
	// APICert is the cert of the client cert to access the api endpoint
	APICert string
	// APIKey is the key of the client cert to access the api endpoint
	APIKey string
}

type DurosSeedStorageClass struct {
	// Name is the name of the storage class
	Name string
	// ReplicaCount is the amount of replicas in the storage backend for this storage class
	ReplicaCount int
	// Compression enables compression for this storage class
	Compression bool
	// Encryption defines a SC with client side encryption enabled
	Encryption bool
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string
}
