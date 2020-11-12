package metal

import "path/filepath"

const (
	// Name is the name of the Metal provider.
	Name = "provider-metal"

	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// MCMProviderMetalImageName is the name of the metal provider plugin image.
	MCMProviderMetalImageName = "machine-controller-manager-provider-metal"
	// CCMImageName is the name of the cloud controller manager image.
	CCMImageName = "metalccm"
	// GroupRolebindingControllerImageName is the name of the GroupRolebindingController image
	GroupRolebindingControllerImageName = "group-rolebinding-controller"
	// AccountingExporterImageName is the name of the accounting exporter image
	AccountingExporterImageName = "accounting-exporter"
	// AuthNWebhookImageName is the name of the AuthN Webhook configured with the shoot kube-apiserver
	AuthNWebhookImageName = "authn-webhook"
	// SplunkAuditWebhookImageName is the name of the splunk audit Webhook configured with the shoot kube-apiserver
	SplunkAuditWebhookImageName = "splunk-audit-webhook"
	// DroptailerImageName is the name of the Droptailer to deploy to the shoot.
	DroptailerImageName = "droptailer"
	// MetallbSpeakerImageName is the name of the metallb speaker to deploy to the shoot.
	MetallbSpeakerImageName = "metallb-speaker"
	// MetallbControllerImageName is the name of the metallb controller to deploy to the shoot.
	MetallbControllerImageName = "metallb-controller"
	// CSIControllerImageName is the name of the csi lvm controller to deploy to the seed's shoot namespace.
	CSIControllerImageName = "csi-lvm-controller"
	// CSIProvisionerImageName is the name of the csi lvm provisioner to deploy to the seed's shoot namespace.
	CSIProvisionerImageName = "csi-lvm-provisioner"

	// APIKey is a constant for the key in a cloud provider secret.
	APIKey = "metalAPIKey"
	// APIHMac is a constant for the hmac in a cloud provider secret.
	APIHMac = "metalAPIHMac"

	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"
	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"

	AuthNWebHookConfigName               = "authn-webhook-config"
	AuthNWebHookCertName                 = "authn-webhook-cert"
	ShootExtensionTypeTokenIssuer        = "tokenissuer"
	DroptailerNamespace                  = "firewall"
	DroptailerClientSecretName           = "droptailer-client"
	DroptailerServerSecretName           = "droptailer-server"
	CloudControllerManagerDeploymentName = "cloud-controller-manager"
	CloudControllerManagerServerName     = "cloud-controller-manager-server"
	GroupRolebindingControllerName       = "group-rolebinding-controller"
	AccountingExporterName               = "accounting-exporter"
	AuthNWebhookDeploymentName           = "kube-jwt-authn-webhook"
	AuthNWebhookServerName               = "kube-jwt-authn-webhook-server"
	SplunkAuditWebhookDeploymentName     = "splunk-audit-webhook"
	SplunkAuditWebhookServerName         = "splunk-audit-webhook-server"
	SplunkAuditWebHookConfigName         = "splunk-audit-webhook-config"
	SplunkAuditWebHookCertName           = "splunk-audit-webhook-cert"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("controllers", Name, "charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
)

// Credentials stores Metal credentials.
type Credentials struct {
	MetalAPIKey  string
	MetalAPIHMac string
}
