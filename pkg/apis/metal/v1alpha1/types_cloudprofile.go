package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta

	// MetalControlPlanes is a map of a control plane name to control plane configuration
	MetalControlPlanes map[string]MetalControlPlane `json:"metalControlPlanes"`
}

// MetalControlPlane contains configuration specific for this metal stack control plane
type MetalControlPlane struct {
	// Endpoint is the endpoint to the metal-api of the control plane
	Endpoint string `json:"endpoint"`
	// Partitions is a map of a region name from the regions defined in the cloud profile to region-specific control plane settings
	Partitions map[string]Partition `json:"partitions"`
	// FirewallImages is a list of available firewall images in this control plane. When empty, allows all values.
	FirewallImages []string `json:"firewallImages,omitempty"`
	// FirewallControllerVersions is a list of available firewall controller binary versions
	FirewallControllerVersions []FirewallControllerVersion `json:"firewallControllerVersions,omitempty"`
	// NftablesExporter is the nftables exporter which will be reconciled by the firewall controller
	NftablesExporter NftablesExporter `json:"nftablesExporter,omitempty"`
}

// FirewallControllerVersion describes the version of the firewall controller binary
// version must not be semver compatible, the version of the created PR binary is also valid
// but for the calculation of the most recent version, only semver compatible versions are considered.
// Version 2fb7fd7 URL: https://images.metal-stack.io/firewall-controller/pull-requests/101-upload-to-gcp/firewall-controller
// Version a273591 URL: https://images.metal-stack.io/firewall-controller/pull-requests/102-dns-cwnp/firewall-controller
// Version v1.0.10 URL: https://images.metal-stack.io/firewall-controller/v1.0.10/firewall-controller
// Version v1.0.11 URL: https://images.metal-stack.io/firewall-controller/v1.0.11/firewall-controller
type FirewallControllerVersion struct {
	// Version is the version name of the firewall controller
	Version string `json:"version"`
	// URL points to the downloadable binary artifact of the firewall controller
	URL string `json:"url"`
	// Classification defines the state of a version (preview, supported, deprecated)
	Classification *VersionClassification `json:"classification,omitempty"`
}

// NftablesExporter describes the version of the nftables exporter binary
type NftablesExporter struct {
	// Version is the version name of the nftables exporter
	Version string `json:"version"`
	// URL points to the downloadable binary artifact of the nftables exporter
	URL string `json:"url"`
}

// VersionClassification is the logical state of a version according to https://github.com/gardener/gardener/blob/master/docs/operations/versioning.md
type VersionClassification string

const (
	// ClassificationPreview indicates that a version has recently been added and not promoted to "Supported" yet.
	// ClassificationPreview versions will not be considered for automatic firewallcontroller version updates.
	ClassificationPreview VersionClassification = "preview"
	// ClassificationSupported indicates that a patch version is the recommended version for a shoot.
	// Supported versions are eligible for the automated firewallcontroller version update.
	ClassificationSupported VersionClassification = "supported"
	// ClassificationDeprecated indicates that a patch version should not be used anymore, should be updated to a new version
	// and will eventually expire.
	ClassificationDeprecated VersionClassification = "deprecated"
)

// Partition contains configuration specific for this metal stack control plane partition
type Partition struct {
	// FirewallTypes is a list of available firewall machine types in this partition. When empty, allows all values.
	FirewallTypes []string `json:"firewallTypes"`

	// NetworkIsolation if given allows the creation of shoot clusters which have network restrictions activated.
	NetworkIsolation *NetworkIsolation `json:"networkIsolation,omitempty"`
}

type NetworkIsolation struct {
	// AllowedNetworks is a list of networks which are allowed to connect in restricted or forbidden NetworkIsolated clusters.
	AllowedNetworks []string `json:"allowedNetworks,omitempty"`
	// DNSServers
	DNSServers []string `json:"dnsServers,omitempty"`
	// NTPServers
	NTPServers []string `json:"ntpServers,omitempty"`
	// The registry which serves the images required to create a shoot.
	Registry NetworkServer `json:"registry,omitempty"`
}

type NetworkServer struct {
	// Name describes this server
	Name string `json:"name,omitempty"`
	// Hostname is typically the dns name of this server
	Hostname string `json:"hostname,omitempty"`
	// IP is the ipv4 or ipv6 address of this server
	IP string `json:"ip,omitempty"`
	// IPFamily defines the family of the ip
	IPFamily corev1.IPFamily `json:"ipfamily,omitempty"`
	// Port at which port the service is reachable
	Port int32 `json:"port,omitempty"`
	// Proto the network protocol to reach the service
	Proto corev1.Protocol `json:"proto,omitempty"`
}
