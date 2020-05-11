package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Auth is configuration for metal stack specific user authentication in the cluster.
	Auth Auth `json:"auth"`

	// Audit contains the configuration for the auditlogging.
	Audit Audit `json:"audit"`

	// AccountingExporter is the configuration for the accounting exporter.
	AccountingExporter AccountingExporterConfiguration `json:"accountingExporter,omitempty"`
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
}

// Auth contains the configuration for metal stack specific user authentication in the cluster.
type Auth struct {
	// Enabled enables the deployment of metal stack specific cluster authentication when set to true.
	Enabled bool `json:"enabled"`
	// ProviderTenant is the name of the provider tenant who has special privileges.
	ProviderTenant string `json:"providerTenant"`
}

// Audit contains the configuration for the auditlogging.
type Audit struct {
	// Enabled enables the deployment of auditlog webhook when set to true.
	Enabled bool `json:"enabled"`
}

// AccountingExporterConfiguration contains the configuration for the accounting exporter.
type AccountingExporterConfiguration struct {
	// Enabled enables the deployment of the accounting exporter when set to true.
	Enabled bool `json:"enabled"`
	// Client contains the configuration for the accounting exporter client.
	Client AccountingExporterClientConfiguration `json:"clientConfig"`
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
