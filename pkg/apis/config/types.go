package config

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	componentbaseconfig "k8s.io/component-base/config"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the metal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection *componentbaseconfig.ClientConnectionConfiguration

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to metal-specific identifiers, i.e. AMIs.
	MachineImages []MachineImage

	// FirewallInternalPrefixes is a list of prefixes for the firewall-controller
	// which will be counted as internal network traffic. this is important for accounting
	// networking traffic.
	FirewallInternalPrefixes []string

	// ETCD is the etcd configuration.
	ETCD ETCD

	// ClusterAudit is the configuration for cluster auditing.
	ClusterAudit ClusterAudit

	// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
	AuditToSplunk AuditToSplunk

	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig

	// Storage is the configuration for storage.
	Storage StorageConfiguration

	// ImagePullPolicy defines the pull policy for the components deployed through the control plane controller.
	// Defaults to IfNotPresent if empty or unknown.
	ImagePullPolicy string

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret

	// EgressDestinations is used when the RestrictEgress control plane feature gate is enabled
	// and provides additional egress destinations to the kube-apiserver.
	//
	// It is intended to be configured at least with container registries for the cluster.
	// Deprecated: Will be replaced by NetworkAccessRestricted.
	EgressDestinations []EgressDest
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

// ClusterAudit is the configuration for cluster auditing.
type ClusterAudit struct {
	// Enabled enables collecting of the kube-apiserver auditlog.
	Enabled bool
}

// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
type AuditToSplunk struct {
	// Enabled enables forwarding of the kube-apiserver auditlog to splunk.
	Enabled bool
	// This defines the default splunk endpoint unless otherwise specified by the cluster user
	HECToken   string
	Index      string
	HECHost    string
	HECPort    int
	TLSEnabled bool
	HECCAFile  string
}

// StorageConfiguration contains the configuration for provider specfic storage solutions.
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

	// APIEndpoint is an optional endpoint used for control plane network communication.
	//
	// In certain scenarios the data plane network cannot be reached from the duros-controller in the seed
	// (i.e. only the shoot is able to reach the storage network).
	//
	// In these cases, APIEndpoint can be utilized to point to a gRPC proxy such that the storage
	// integration can be deployed anyway.
	APIEndpoint *string
	// APICA is the ca of the client cert to access the grpc-proxy
	APICA string
	// APICert is the cert of the client cert to access the grpc-proxy
	APICert string
	// APIKey is the key of the client cert to access the grpc-proxy
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

type EgressDest struct {
	// Description is a description for this egress destination.
	Description string
	// MatchPattern is the DNS match pattern for this destination. Use either a pattern or a name.
	MatchPattern string
	// MatchName is the DNS match name for this destination. Use either a pattern or a name.
	MatchName string
	// Protocol is either TCP or UDP.
	Protocol string
	// Port is the port for this destination.
	Port int
}
