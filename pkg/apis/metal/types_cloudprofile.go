package metal

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
	MetalControlPlanes map[string]MetalControlPlane
}

// MetalControlPlane contains configuration specific for this metal stack control plane
type MetalControlPlane struct {
	// Endpoint is the endpoint to the metal-api of the control plane
	Endpoint string
	// Partitions is a map of a region name from the regions defined in the cloud profile to region-specific control plane settings
	Partitions map[string]Partition
	// FirewallImages is a list of available firewall images in this control plane. When empty, allows all values.
	FirewallImages []string
	// FirewallControllerVersions is a list of available firewall controller binary versions
	FirewallControllerVersions []FirewallControllerVersion
	// NftablesExporter is the nftables exporter which will be reconciled by the firewall controller
	NftablesExporter NftablesExporter
}

// FirewallControllerVersion describes the version of the firewall controller binary
type FirewallControllerVersion struct {
	// Version is the version name of the firewall controller
	Version string
	// URL points to the downloadable binary artifact of the firewall controller
	URL string
	// Classification defines the state of a version (preview, supported, deprecated)
	Classification *VersionClassification
}

// NftablesExporter describes the version of the nftables exporter binary
type NftablesExporter struct {
	// Version is the version name of the nftables exporter
	Version string
	// URL points to the downloadable binary artifact of the nftables exporter
	URL string
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
	FirewallTypes []string

	// NetworkIsolation if given allows the creation of shoot clusters which have network restrictions activated.
	// Will be taken into account if NetworkAccessRestricted or NetworkAccessForbidden is defined
	// +optional
	NetworkIsolation *NetworkIsolation
}

type NetworkIsolation struct {
	// AllowedNetworks is a list of networks which are allowed to connect in restricted or forbidden NetworkIsolated clusters.
	AllowedNetworks []string
	// DNSServers
	DNSServers []string
	// NTPServers
	NTPServers []string
	// The registry which serves the images required to create a shoot.
	Registry NetworkServer
}

type NetworkServer struct {
	// Name describes this server
	Name string
	// Hostname is typically the dns name of this server
	Hostname string
	// IP is the ipv4 or ipv6 address of this server
	IP string
	// IPFamily defines the family of the ip
	IPFamily corev1.IPFamily
	// Port at which port the service is reachable
	Port int32
	// Proto the network protocol to reach the service
	Proto corev1.Protocol
}
