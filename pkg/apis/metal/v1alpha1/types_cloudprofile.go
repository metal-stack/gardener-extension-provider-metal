package v1alpha1

import (
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
	// IAMConfig contains the config for all AuthN/AuthZ related components, can be overriden in shoots control plane config
	// +optional
	IAMConfig *IAMConfig `json:"iamconfig,omitempty"`
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
}

// IAMConfig contains the config for all AuthN/AuthZ related components
type IAMConfig struct {
	IssuerConfig *IssuerConfig         `json:"issuerConfig,omitempty"`
	IdmConfig    *IDMConfig            `json:"idmConfig,omitempty"`
	GroupConfig  *NamespaceGroupConfig `json:"groupConfig,omitempty"`
}

// IssuerConfig contains configuration settings for the token issuer.
type IssuerConfig struct {
	Url      string `json:"url,omitempty"`
	ClientId string `json:"clientId,omitempty"`
}

// IDMConfig contains config for the IDM-System that is used as directory for users and groups
type IDMConfig struct {
	Idmtype string `json:"idmtype,omitempty"`

	ConnectorConfig *ConnectorConfig `json:"connectorConfig,omitempty"`
}

// NamespaceGroupConfig for group-rolebinding-controller
type NamespaceGroupConfig struct {
	// no action is taken or any namespace in this list
	// kube-system,kube-public,kube-node-lease,default
	ExcludedNamespaces string `json:"excludedNamespaces,omitempty"`
	// for each element a RoleBinding is created in any Namespace - ClusterRoles are bound with this name
	// admin,edit,view
	ExpectedGroupsList string `json:"expectedGroupsList,omitempty"`
	// Maximum length of namespace-part in clusterGroupname and therefore in the corresponding groupname in the directory.
	// 20 chars für AD, given the FITS-naming-conventions
	NamespaceMaxLength int `json:"namespaceMaxLength,omitempty"`
	// The created RoleBindings will reference this group (from token).
	// oidc:{{ .Namespace }}-{{ .Group }}
	ClusterGroupnameTemplate string `json:"clusterGroupnameTemplate,omitempty"`
	// The RoleBindings will created with this name.
	// oidc-{{ .Namespace }}-{{ .Group }}
	RoleBindingNameTemplate string `json:"roleBindingNameTemplate,omitempty"`
}

// ConnectorConfig optional config for the IDM Webhook - if it should be used to automatically create/delete groups/roles in the tenant IDM
type ConnectorConfig struct {
	IdmApiUrl            string `json:"idmApiUrl,omitempty"`
	IdmApiUser           string `json:"idmApiUser,omitempty"`
	IdmApiPassword       string `json:"idmApiPassword,omitempty"`
	IdmSystemId          string `json:"idmSystemId,omitempty"`
	IdmAccessCode        string `json:"idmAccessCode,omitempty"`
	IdmCustomerId        string `json:"idmCustomerId,omitempty"`
	IdmGroupOU           string `json:"idmGroupOU,omitempty"`
	IdmGroupnameTemplate string `json:"idmGroupnameTemplate,omitempty"`
	IdmDomainName        string `json:"idmDomainName,omitempty"`
	IdmTenantPrefix      string `json:"idmTenantPrefix,omitempty"`
	IdmSubmitter         string `json:"idmSubmitter,omitempty"`
	IdmJobInfo           string `json:"idmJobInfo,omitempty"`
	IdmReqSystem         string `json:"idmReqSystem,omitempty"`
	IdmReqUser           string `json:"idmReqUser,omitempty"`
	IdmReqEMail          string `json:"idmReqEMail,omitempty"`
}
