package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	componentbaseconfigv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the metal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	// +optional
	ClientConnection *componentbaseconfigv1alpha1.ClientConnectionConfiguration `json:"clientConnection,omitempty"`

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to metal-specific identifiers, i.e. AMIs.
	MachineImages []MachineImage `json:"machineImages,omitempty"`

	// FirewallInternalPrefixes is a list of prefixes for the firewall-controller
	// which will be counted as internal network traffic. this is important for accounting
	// networking traffic.
	FirewallInternalPrefixes []string `json:"firewallInternalPrefixes,omitempty"`

	// ETCD is the etcd configuration.
	ETCD ETCD `json:"etcd"`

	// ClusterAudit is the configuration for cluster auditing.
	ClusterAudit ClusterAudit `json:"clusterAudit"`

	// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
	AuditToSplunk AuditToSplunk `json:"auditToSplunk"`

	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`

	// Storage is the configuration for storage.
	Storage StorageConfiguration `json:"storage,omitempty"`

	// ImagePullPolicy defines the pull policy for the components deployed through the control plane controller.
	// Defaults to IfNotPresent if empty or unknown.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	// +optional
	ImagePullSecret *ImagePullSecret `json:"imagePullSecret,omitempty"`

	// EgressDestinations is used when the RestrictEgress control plane feature gate is enabled
	// and provides additional egress destinations to the kube-apiserver.
	//
	// It is intended to be configured at least with container registries for the cluster.
	// Deprecated: Will be replaced by NetworkAccessRestricted.
	EgressDestinations []EgressDest `json:"egressDestinations,omitempty"`
}

// MachineImage is a mapping from logical names and versions to GCP-specific identifiers.
type MachineImage struct {
	// Name is the logical name of the machine image.
	Name string `json:"name"`
	// Version is the logical version of the machine image.
	Version string `json:"version"`
	// Image is the path to the image.
	Image string `json:"image"`
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage `json:"storage"`
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup `json:"backup"`
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	// +optional
	ClassName *string `json:"className,omitempty"`
	// Capacity is the storage capacity used in etcd-main volume claims.
	// +optional
	Capacity *resource.Quantity `json:"capacity,omitempty"`
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	// +optional
	Schedule *string `json:"schedule,omitempty"`
	// DeltaSnapshotPeriod is the time for delta snapshots to be made
	DeltaSnapshotPeriod *string `json:"deltaSnapshotPeriod,omitempty"`
}

// ClusterAudit is the configuration for cluster auditing.
type ClusterAudit struct {
	// Enabled enables collecting of the kube-apiserver audit log.
	Enabled bool `json:"enabled"`
}

// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
type AuditToSplunk struct {
	// Enabled enables forwarding of the kube-apiserver auditlogto splunk.
	Enabled bool `json:"enabled"`
	// This defines the default splunk endpoint unless otherwise specified by the cluster user
	HECToken   string `json:"hecToken"`
	Index      string `json:"index"`
	HECHost    string `json:"hecHost"`
	HECPort    int    `json:"hecPort"`
	TLSEnabled bool   `json:"tlsEnabled"`
	HECCAFile  string `json:"hecCAFile"`
}

// StorageConfiguration contains the configuration for provider specfic storage solutions.
type StorageConfiguration struct {
	// Duros contains the configuration for duros cloud storage
	Duros DurosConfiguration `json:"duros"`
}

// DurosConfiguration contains the configuration for lightbits duros storage.
type DurosConfiguration struct {
	// Enabled enables duros storage when set to true.
	Enabled bool `json:"enabled"`

	// PartitionConfig is a map of a partition id to the duros partition configuration
	PartitionConfig map[string]DurosPartitionConfiguration `json:"partitionConfig"`
}

// DurosPartitionConfiguration is the configuration for duros for a particular partition
type DurosPartitionConfiguration struct {
	// Endpoints is the list of endpoints for the storage data plane and control plane communication
	Endpoints []string `json:"endpoints"`
	// AdminKey is the key used for generating storage credentials
	AdminKey string `json:"adminKey"`
	// AdminToken is the token used by the duros-controller to authenticate against the duros API
	AdminToken string `json:"adminToken"`
	// StorageClasses contain information on the storage classes that the duros-controller creates in the shoot cluster
	StorageClasses []DurosSeedStorageClass `json:"storageClasses"`

	// APIEndpoint is the endpoint used for control plane network communication.
	APIEndpoint string `json:"apiEndpoint"`
	// APICA is the ca of the client cert to access the api endpoint
	APICA string `json:"apiCA,omitempty"`
	// APICert is the cert of the client cert to access the api endpoint
	APICert string `json:"apiCert,omitempty"`
	// APIKey is the key of the client cert to access the api endpoint
	APIKey string `json:"apiKey,omitempty"`
}

type DurosSeedStorageClass struct {
	// Name is the name of the storage class
	Name string `json:"name"`
	// ReplicaCount is the amount of replicas in the storage backend for this storage class
	ReplicaCount int `json:"replicaCount"`
	// Compression enables compression for this storage class
	Compression bool `json:"compression"`
	// Encryption defines a SC with client side encryption enabled
	Encryption bool `json:"encryption"`
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string `json:"encodedDockerConfigJSON"`
}

type EgressDest struct {
	// Description is a description for this egress destination.
	Description string `json:"description,omitempty"`
	// MatchPattern is the DNS match pattern for this destination.
	MatchPattern string `json:"matchPattern,omitempty"`
	// MatchName is the DNS match name for this destination. Use either a pattern or a name.
	MatchName string `json:"matchName,omitempty"`
	// Protocol is either TCP or UDP.
	Protocol string `json:"protocol,omitempty"`
	// Port is the port for this destination.
	Port int `json:"port,omitempty"`
}
