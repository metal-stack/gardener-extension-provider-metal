package metal

import "path/filepath"

const (
	// Name is the name of the Metal provider.
	Name = "provider-metal"

	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// CCMImageName is the name of the cloud controller manager image.
	CCMImageName = "metalccm"
	// GroupRolebindingControllerImageName is the name of the GroupRolebindingController image
	GroupRolebindingControllerImageName = "group-rolebinding-controller"
	// AccountingExporterImageName is the name of the accounting exporter image
	AccountingExporterImageName = "accounting-exporter"
	// AuthNWebhookImageName is the name of the AuthN Webhook configured with the shoot kube-apiserver
	AuthNWebhookImageName = "authn-webhook"
	// DroptailerImageName is the name of the Droptailer to deploy to the shoot.
	DroptailerImageName = "droptailer"
	// LimitValidatingWebhookImageName is the name of the limit validating webhook to deploy to the seed's shoot namespace.
	LimitValidatingWebhookImageName = "limit-validating-webhook"
	// CSIControllerImageName is the name of the csi lvm controller to deploy to the seed's shoot namespace.
	CSIControllerImageName = "csi-lvm-controller"
	// CSIProvisionerImageName is the name of the csi lvm provisioner to deploy to the seed's shoot namespace.
	CSIProvisionerImageName = "csi-lvm-provisioner"

	// APIURL is a constant for the url of metal-api.
	APIURL = "metalAPIURL"
	// APIKey is a constant for the key in a cloud provider secret.
	APIKey = "metalAPIKey"
	// APIHMac is a constant for the hmac in a cloud provider secret.
	APIHMac = "metalAPIHMac"

	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"
	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"

	// AuthN Webhook
	AuthNWebHookConfigName        = "authn-webhook-config"
	AuthNWebHookCertName          = "authn-webhook-cert"
	ShootExtensionTypeTokenIssuer = "tokenissuer"

	// FIXME: change to metal-stack
	ShootAnnotationProject     = "cluster.metal-pod.io/project"
	ShootAnnotationDescription = "cluster.metal-pod.io/description"
	ShootAnnotationClusterName = "cluster.metal-pod.io/name"
	ShootAnnotationTenant      = "cluster.metal-pod.io/tenant"
	ShootAnnotationClusterID   = "cluster.metal-pod.io/id"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("controllers", Name, "charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
)

// Credentials stores Metal credentials.
type Credentials struct {
	MetalAPIURL  string
	MetalAPIKey  string
	MetalAPIHMac string
}
