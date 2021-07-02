package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	healthcheckconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config/v1alpha1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the metal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta `json:",inline"`

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to metal-specific identifiers, i.e. AMIs.
	MachineImages []MachineImage `json:"machineImages,omitempty"`

	// ETCD is the etcd configuration.
	ETCD ETCD `json:"etcd"`

	// ClusterAudit is the configuration for cluster auditing.
	ClusterAudit ClusterAudit `json:"clusterAudit"`

	// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
	AuditToSplunk AuditToSplunk `json:"auditToSplunk"`

	// Auth is the configuration for metal stack specific user authentication in the cluster.
	Auth Auth `json:"auth"`

	// AccountingExporter is the configuration for the accounting exporter.
	AccountingExporter AccountingExporterConfiguration `json:"accountingExporter,omitempty"`

	// HealthCheckConfig is the config for the health check controller
	// +optional
	HealthCheckConfig *healthcheckconfigv1alpha1.HealthCheckConfig `json:"healthCheckConfig,omitempty"`

	// Storage is the configuration for storage.
	Storage StorageConfiguration `json:"storage,omitempty"`

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret `json:"imagePullSecret,omitempty"`
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
	// Enabled enables collecting of the kube-apiserver auditlog.
	Enabled    bool   `json:"enabled"`
	HECToken   string `json:"hecToken"`
	Index      string `json:"index"`
	HECHost    string `json:"hecHost"`
	HECPort    int    `json:"hecPort"`
	TLSEnabled bool   `json:"tlsEnabled"`
	HECCAFile  string `json:"hecCAFile"`
}

// Auth contains the configuration for metal stack specific user authentication in the cluster.
type Auth struct {
	// Enabled enables the deployment of metal stack specific cluster authentication when set to true.
	Enabled bool `json:"enabled"`
	// ProviderTenant is the name of the provider tenant who has special privileges.
	ProviderTenant string `json:"providerTenant"`
}

// AccountingExporterConfiguration contains the configuration for the accounting exporter.
type AccountingExporterConfiguration struct {
	// Enabled enables the deployment of the accounting exporter when set to true.
	Enabled bool `json:"enabled"`
	// NetworkTraffic contains the configuration for accounting network traffic
	NetworkTraffic AccountingExporterNetworkTrafficConfiguration `json:"networkTraffic"`
	// Client contains the configuration for the accounting exporter client.
	Client AccountingExporterClientConfiguration `json:"clientConfig"`
}

// AccountingExporterClientConfiguration contains the configuration for the network traffic accounting.
type AccountingExporterNetworkTrafficConfiguration struct {
	// Enabled enables network traffic accounting of the accounting exporter when set to true.
	Enabled bool `json:"enabled"`
	// InternalNetworks defines the networks for the firewall that are considered internal (which can be accounted differently)
	InternalNetworks []string `json:"internalNetworks"`
}

// AccountingExporterClientConfiguration contains the configuration for the accounting exporter client.
type AccountingExporterClientConfiguration struct {
	// Hostname is the hostname of the accounting api.
	Hostname string `json:"hostname"`
	// Port is the port of the accounting api.
	Port int `json:"port"`
	// CA is the ca certificate used for communicating with the accounting api.
	CA string `json:"ca"`
	// Cert is the client certificate used for communicating with the accounting api.
	Cert string `json:"cert"`
	// CertKey is the client certificate key used for communicating with the accounting api.
	CertKey string `json:"certKey"`
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

	// SeedConfig is a map of a seed name to the duros seed configuration
	SeedConfig map[string]DurosSeedConfiguration `json:"seedConfig"`
}

// DurosSeedConfiguration is the configuration for duros for a particular seed
type DurosSeedConfiguration struct {
	// Endpoints is the list of endpoints of the duros API
	Endpoints []string `json:"endpoints"`
	// AdminKey is the key used for generating storage credentials
	AdminKey string `json:"adminKey"`
	// AdminToken is the token used by the duros-controller to authenticate against the duros API
	AdminToken string `json:"adminToken"`
	// StorageClasses contain information on the storage classes that the duros-controller creates in the shoot cluster
	StorageClasses []DurosSeedStorageClass `json:"storageClasses"`
}

type DurosSeedStorageClass struct {
	// Name is the name of the storage class
	Name string `json:"name"`
	// ReplicaCount is the amount of replicas in the storage backend for this storage class
	ReplicaCount int `json:"replicaCount"`
	// Compression enables compression for this storage class
	Compression bool `json:"compression"`
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string `json:"encodedDockerConfigJSON"`
}
