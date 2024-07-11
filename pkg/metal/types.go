package metal

const (
	// Name is the name of the Metal provider.
	Name = "provider-metal"

	// MCMProviderMetalImageName is the name of the metal provider plugin image.
	MCMProviderMetalImageName = "machine-controller-manager-provider-metal"
	// CCMImageName is the name of the cloud controller manager image.
	CCMImageName = "metalccm"
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
	// DurosControllerImageName is the name of the duros controller to deploy to the seed's shoot namespace.
	DurosControllerImageName = "duros-controller"
	// DurosResourceName is the name of the duros resource to deploy to the seed's shoot namespace.
	DurosResourceName = "shoot-default-storage"
	// NodeInitImageName is the name of the node-init to deploy to the shoot.
	NodeInitImageName = "node-init"
	// MetallbHealthSidecarImageName is the name of the metallb health sidecar image used for metallb health checking to deploy to the shoot.
	MetallbHealthSidecarImageName = "metallb-health-sidecar"

	// APIKey is a constant for the key in a cloud provider secret.
	APIKey = "metalAPIKey"
	// APIHMac is a constant for the hmac in a cloud provider secret.
	APIHMac = "metalAPIHMac"

	// CloudProviderConfigName is the name of the configmap containing the cloud provider config.
	CloudProviderConfigName = "cloud-provider-config"

	// ShootExtensionTypeTokenIssuer appears unused? CHECKME
	ShootExtensionTypeTokenIssuer = "tokenissuer"
	// DroptailerNamespace is the namespace where the firewall droptailer will get deployed.
	DroptailerNamespace = "firewall"
	// DroptailerClientSecretName is the name of the secret containing the certificates for the droptailer client.
	DroptailerClientSecretName = "droptailer-client" // nolint:gosec
	// DroptailerServerSecretName is the name of the secret containing the certificates for the droptailer server.
	DroptailerServerSecretName = "droptailer-server" // nolint:gosec
	// CloudControllerManagerDeploymentName is the name of the deployment for the cloud controller manager.
	CloudControllerManagerDeploymentName = "cloud-controller-manager"
	// CloudControllerManagerServerName is the name of the secret containing the certificates for the cloud controller manager server.
	CloudControllerManagerServerName = "cloud-controller-manager-server"
	// DurosControllerDeploymentName is the name of the deployment for the duros-controller.
	DurosControllerDeploymentName = "duros-controller"
	// FirewallControllerManagerDeploymentName is the name of the deployment for the firewall controller manager
	FirewallControllerManagerDeploymentName = "firewall-controller-manager"
	// FirewallDeploymentName is the name of the firewall deployment deployed to the seed cluster to get managed by the FCM.
	FirewallDeploymentName = "shoot-firewall"
	// ManagerIdentity is put as a label to every secret managed by the gepm and secretsmanager to make searching easier
	ManagerIdentity = "provider-" + Type + "-controlplane"
)

// Credentials stores Metal credentials.
type Credentials struct {
	MetalAPIKey  string
	MetalAPIHMac string
}
