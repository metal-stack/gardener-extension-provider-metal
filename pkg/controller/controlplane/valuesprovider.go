package controlplane

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"path/filepath"

	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/go-logr/logr"

	"github.com/pkg/errors"

	admissionv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Object names
const (
	cloudControllerManagerDeploymentName = "cloud-controller-manager"
	cloudControllerManagerServerName     = "cloud-controller-manager-server"
	groupManagerName                     = "group-manager"
	limitValidatingWebhookDeploymentName = "limit-validating-webhook"
	limitValidatingWebhookServerName     = "limit-validating-webhook-server"
	accountingExporterName               = "accounting-exporter"
	authNWebhookDeploymentName           = "kube-jwt-authn-webhook"
	authNWebhookServerName               = "kube-jwt-authn-webhook-server"
	splunkAuditWebhookDeploymentName     = "splunk-audit-webhook"
	splunkAuditWebhookServerName         = "splunk-audit-webhook-server"
	droptailerNamespace                  = "firewall"
	droptailerClientSecretName           = "droptailer-client"
	droptailerServerSecretName           = "droptailer-server"
)

var controlPlaneSecrets = &secrets.Secrets{
	CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
		v1alpha1constants.SecretNameCACluster: {
			Name:       v1alpha1constants.SecretNameCACluster,
			CommonName: "kubernetes",
			CertType:   secrets.CACert,
		},
	},
	SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
		return []secrets.ConfigInterface{
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         cloudControllerManagerDeploymentName,
					CommonName:   "system:cloud-controller-manager",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1alpha1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         groupManagerName,
					CommonName:   "system:group-manager",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1alpha1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       authNWebhookServerName,
					CommonName: authNWebhookDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(authNWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1alpha1constants.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       splunkAuditWebhookServerName,
					CommonName: splunkAuditWebhookDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(splunkAuditWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1alpha1constants.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       limitValidatingWebhookServerName,
					CommonName: limitValidatingWebhookDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(limitValidatingWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1alpha1constants.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       accountingExporterName,
					CommonName: "system:accounting-exporter",
					// Groupname of user
					Organization: []string{accountingExporterName},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: v1alpha1constants.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       cloudControllerManagerServerName,
					CommonName: cloudControllerManagerDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(cloudControllerManagerDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1alpha1constants.SecretNameCACluster],
				},
			},
		}
	},
}

var configChart = &chart.Chart{
	Name:    "config",
	Path:    filepath.Join(metal.InternalChartsPath, "cloud-provider-config"),
	Images:  []string{},
	Objects: []*chart.Object{},
}

var controlPlaneChart = &chart.Chart{
	Name:   "control-plane",
	Path:   filepath.Join(metal.InternalChartsPath, "control-plane"),
	Images: []string{metal.CCMImageName},
	Objects: []*chart.Object{
		// cloud controller manager
		{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
		{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},

		// network policies
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-dns"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-any"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-http"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-https"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-ntp"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-vpn"},
	},
}

var cpShootChart = &chart.Chart{
	Name:   "shoot-control-plane",
	Path:   filepath.Join(metal.InternalChartsPath, "shoot-control-plane"),
	Images: []string{metal.DroptailerImageName, metal.MetallbSpeakerImageName, metal.MetallbControllerImageName},
	Objects: []*chart.Object{
		// metallb
		{Type: &corev1.Namespace{}, Name: "metallb-system"},
		{Type: &policyv1beta1.PodSecurityPolicy{}, Name: "speaker"},
		{Type: &corev1.ServiceAccount{}, Name: "controller"},
		{Type: &corev1.ServiceAccount{}, Name: "speaker"},
		{Type: &rbacv1.ClusterRole{}, Name: "metallb-system:controller"},
		{Type: &rbacv1.ClusterRole{}, Name: "metallb-system:speaker"},
		{Type: &rbacv1.Role{}, Name: "config-watcher"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "metallb-system:controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "metallb-system:speaker"},
		{Type: &rbacv1.RoleBinding{}, Name: "config-watcher"},
		{Type: &appsv1.DaemonSet{}, Name: "speaker"},
		{Type: &appsv1.Deployment{}, Name: "controller"},

		// network policies
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-dns"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-any"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-http"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-https"},
		{Type: &networkingv1.NetworkPolicy{}, Name: "egress-allow-ntp"},

		// cluster wide network policies
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-http"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-https"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-dns"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-ntp"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-vpn"},

		// firewall controller
		{Type: &rbacv1.ClusterRole{}, Name: "system:firewall-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:firewall-controller"},
		{Type: &firewallv1.Firewall{}, Name: "firewall"},

		// firewall policy controller TODO can be removed in a future version
		{Type: &rbacv1.ClusterRole{}, Name: "system:firewall-policy-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:firewall-policy-controller"},

		// droptailer
		{Type: &appsv1.Deployment{}, Name: "droptailer"},

		// ccm
		{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},
		{Type: &rbacv1.ClusterRole{}, Name: "cloud-controller-manager"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "cloud-controller-manager"},
	},
}

var storageClassChart = &chart.Chart{
	Name:   "shoot-storageclasses",
	Path:   filepath.Join(metal.InternalChartsPath, "shoot-storageclasses"),
	Images: []string{metal.CSIControllerImageName, metal.CSIProvisionerImageName},
	Objects: []*chart.Object{
		{Type: &corev1.Namespace{}, Name: "csi-lvm"},
		{Type: &storagev1.StorageClass{}, Name: "csi-lvm"},
		{Type: &corev1.ServiceAccount{}, Name: "csi-lvm-controller"},
		{Type: &rbacv1.ClusterRole{}, Name: "csi-lvm-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "csi-lvm-controller"},
		{Type: &appsv1.Deployment{}, Name: "csi-lvm-controller"},
		{Type: &corev1.ServiceAccount{}, Name: "csi-lvm-reviver"},
		{Type: &rbacv1.Role{}, Name: "csi-lvm-reviver"},
		{Type: &rbacv1.RoleBinding{}, Name: "csi-lvm-reviver"},
		{Type: &policyv1beta1.PodSecurityPolicy{}, Name: "csi-lvm-reviver-psp"},
		{Type: &rbacv1.Role{}, Name: "csi-lvm-reviver-psp"},
		{Type: &rbacv1.RoleBinding{}, Name: "csi-lvm-reviver-psp"},
		{Type: &appsv1.DaemonSet{}, Name: "csi-lvm-reviver"},
	},
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(mgr manager.Manager, logger logr.Logger, controllerConfig config.ControllerConfiguration) genericactuator.ValuesProvider {
	if controllerConfig.Auth.Enabled {
		configChart.Objects = append(configChart.Objects, []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: "authn-webhook-config"},
		}...)
		controlPlaneChart.Images = append(controlPlaneChart.Images, []string{
			metal.AuthNWebhookImageName,
			metal.GroupRolebindingControllerImageName,
			metal.GroupManagerImageName,
			metal.LimitValidatingWebhookImageName,
		}...)
		controlPlaneChart.Objects = append(controlPlaneChart.Objects, []*chart.Object{
			// authn webhook
			{Type: &appsv1.Deployment{}, Name: "kube-jwt-authn-webhook"},
			{Type: &corev1.Service{}, Name: "kube-jwt-authn-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kubeapi2kube-jwt-authn-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kube-jwt-authn-webhook-allow-namespace"},

			// group manager
			{Type: &appsv1.Deployment{}, Name: "group-manager"},

			// limit validation webhook
			{Type: &appsv1.Deployment{}, Name: "limit-validating-webhook"},
			{Type: &corev1.Service{}, Name: "limit-validating-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "limit-validating-webhook-allow-namespace"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kubeapi2limit-validating-webhook"},
		}...)
		cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
			// limit validating webhook
			{Type: &admissionv1beta1.ValidatingWebhookConfiguration{}, Name: "limit-validating-webhook"},
			// group manager
			{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:group-manager"},
		}...)
	}
	if controllerConfig.AccountingExporter.Enabled {
		controlPlaneChart.Images = append(controlPlaneChart.Images, []string{metal.AccountingExporterImageName}...)
		controlPlaneChart.Objects = append(controlPlaneChart.Objects, []*chart.Object{
			// accounting exporter
			{Type: &corev1.Secret{}, Name: "accounting-exporter-tls"},
			{Type: &appsv1.Deployment{}, Name: "accounting-exporter"},
		}...)
		cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
			// accounting controller
			{Type: &rbacv1.ClusterRole{}, Name: "system:accounting-exporter"},
			{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:accounting-exporter"},
		}...)

	}
	if controllerConfig.SplunkAudit.Enabled {
		controlPlaneChart.Images = append(controlPlaneChart.Images, []string{metal.SplunkAuditWebhookImageName}...)
		controlPlaneChart.Objects = append(controlPlaneChart.Objects, []*chart.Object{
			// splunk audit webhook
			{Type: &appsv1.Deployment{}, Name: "splunk-audit-webhook"},
			{Type: &corev1.Service{}, Name: "splunk-audit-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "splunk-audit-webhook-allow-apiserver"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kubeapi2splunk-audit-webhook"},
		}...)
	}

	return &valuesProvider{
		mgr:              mgr,
		logger:           logger.WithName("metal-values-provider"),
		controllerConfig: controllerConfig,
	}
}

// valuesProvider is a ValuesProvider that provides metal-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	decoder          runtime.Decoder
	restConfig       *rest.Config
	client           client.Client
	logger           logr.Logger
	controllerConfig config.ControllerConfiguration
	mgr              manager.Manager
}

// InjectScheme injects the given scheme into the valuesProvider.
func (vp *valuesProvider) InjectScheme(scheme *runtime.Scheme) error {
	vp.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

func (vp *valuesProvider) InjectConfig(restConfig *rest.Config) error {
	vp.restConfig = restConfig
	return nil
}

func (vp *valuesProvider) InjectClient(client client.Client) error {
	vp.client = client
	return nil
}

// GetConfigChartValues returns the values for the config chart applied by the generic actuator.
func (vp *valuesProvider) GetConfigChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {

	authValues, err := vp.getAuthNConfigValues(ctx, cp, cluster)
	if err != nil {
		return nil, err
	}

	return authValues, err
}

func (vp *valuesProvider) getAuthNConfigValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	namespace := cluster.ObjectMeta.Name

	// this should work as the kube-apiserver is a pod in the same cluster as the kube-jwt-authn-webhook
	// example https://kube-jwt-authn-webhook.shoot--local--myshootname.svc.cluster.local/authenticate
	url := fmt.Sprintf("https://%s.%s.svc.cluster.local/authenticate", authNWebhookDeploymentName, namespace)

	values := map[string]interface{}{
		"authnWebhook": map[string]interface{}{
			"url":     url,
			"enabled": vp.controllerConfig.Auth.Enabled,
		},
	}

	return values, nil
}

// GetControlPlaneChartValues returns the values for the control plane chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
) (map[string]interface{}, error) {
	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := vp.decoder.Decode(cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, errors.Wrapf(err, "could not decode providerConfig of infrastructure")
	}

	cpConfig, err := helper.ControlPlaneConfigFromControlPlane(cp)
	if err != nil {
		return nil, err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return nil, err
	}

	cpConfig.IAMConfig, err = helper.MergeIAMConfig(metalControlPlane.IAMConfig, cpConfig.IAMConfig)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClient(ctx, vp.client, metalControlPlane.Endpoint, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	// TODO: this is a workaround to speed things for the time being...
	// the infrastructure controller writes the nodes cidr back into the infrastructure status, but the cluster resource does not contain it immediately
	// it would need the start of another reconcilation until the node cidr can be picked up from the cluster resource
	// therefore, we read it directly from the infrastructure status
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := vp.client.Get(ctx, kutil.Key(cp.Namespace, cp.Name), infrastructure); err != nil {
		return nil, err
	}

	// Get CCM chart values
	chartValues, err := getCCMChartValues(cpConfig, infrastructureConfig, infrastructure, cp, cluster, checksums, scaledDown, mclient, metalControlPlane)
	if err != nil {
		return nil, err
	}

	authValues, err := getAuthNGroupRoleChartValues(cpConfig, cluster, vp.controllerConfig.Auth)
	if err != nil {
		return nil, err
	}

	splunkAuditValues, err := getSplunkAuditChartValues(cpConfig, cluster, vp.controllerConfig.SplunkAudit)
	if err != nil {
		return nil, err
	}

	accValues, err := getAccountingExporterChartValues(vp.controllerConfig.AccountingExporter, cluster, infrastructureConfig, mclient)
	if err != nil {
		return nil, err
	}

	lvwValues, err := getLimitValidationWebhookControlPlaneChartValues(cluster)
	if err != nil {
		return nil, err
	}

	merge(chartValues, authValues, splunkAuditValues, accValues, lvwValues)

	return chartValues, nil
}

// merge all source maps in the target map
// hint: prevent overwriting of values due to duplicate keys by the use of prefixes
func merge(target map[string]interface{}, sources ...map[string]interface{}) {
	for sIndex := range sources {
		for k, v := range sources[sIndex] {
			target[k] = v
		}
	}
}

// GetControlPlaneExposureChartValues returns the values for the control plane exposure chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneExposureChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster, m map[string]string) (map[string]interface{}, error) {
	return nil, nil
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster, checksums map[string]string) (map[string]interface{}, error) {
	vp.logger.Info("GetControlPlaneShootChartValues")

	values, err := vp.getControlPlaneShootChartValues(ctx, cp, cluster)
	if err != nil {
		vp.logger.Error(err, "Error getting LimitValidationWebhookChartValues")
		return nil, err
	}

	err = vp.deployControlPlaneShootDroptailerCerts(ctx, cp, cluster)
	if err != nil {
		vp.logger.Error(err, "error deploying droptailer certs")
	}

	return values, nil
}

// getControlPlaneShootChartValues returns the values for the shoot control plane chart.
func (vp *valuesProvider) getControlPlaneShootChartValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	namespace := cluster.ObjectMeta.Name

	secret, err := vp.getSecret(ctx, namespace, limitValidatingWebhookServerName)
	if err != nil {
		return nil, err
	}
	limitCABundle := base64.StdEncoding.EncodeToString(secret.Data[secrets.DataKeyCertificateCA])

	secret, err = vp.getSecret(ctx, namespace, splunkAuditWebhookServerName)
	if err != nil {
		return nil, err
	}
	splunkCABundle := base64.StdEncoding.EncodeToString(secret.Data[secrets.DataKeyCertificateCA])

	// this should work as the kube-apiserver is a pod in the same cluster as the limit-validating-webhook
	// example https://limit-validating-webhook.shoot--local--myshootname.svc.cluster.local/validate
	limitURL := fmt.Sprintf("https://%s.%s.svc.cluster.local/validate", limitValidatingWebhookDeploymentName, namespace)
	splunkURL := fmt.Sprintf("https://%s.%s.svc.cluster.local/audit", splunkAuditWebhookDeploymentName, namespace)

	internalPrefixes := []string{}
	if vp.controllerConfig.AccountingExporter.Enabled && vp.controllerConfig.AccountingExporter.NetworkTraffic.Enabled {
		internalPrefixes = vp.controllerConfig.AccountingExporter.NetworkTraffic.InternalNetworks
	}

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := vp.decoder.Decode(cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, errors.Wrapf(err, "could not decode providerConfig of infrastructure")
	}

	cpConfig, err := helper.ControlPlaneConfigFromControlPlane(cp)
	if err != nil {
		return nil, err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return nil, err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return nil, err
	}

	cpConfig.IAMConfig, err = helper.MergeIAMConfig(metalControlPlane.IAMConfig, cpConfig.IAMConfig)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClient(ctx, vp.client, metalControlPlane.Endpoint, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	rateLimits := []map[string]interface{}{}
	for net, rate := range infrastructureConfig.Firewall.RateLimits {
		resp, err := mclient.NetworkFind(&metalgo.NetworkFindRequest{
			ID: &net,
		})
		if err != nil {
			vp.logger.Info("could not find network by id for rate limit", "id", &net)
			continue
		}
		if len(resp.Networks) != 1 {
			vp.logger.Info("network id is ambiguous", "id", &net)
			continue
		}
		n := resp.Networks[0]
		iface := fmt.Sprintf("vrf%d", n.Vrf)
		rl := map[string]interface{}{
			"interface": iface,
			"rate":      rate,
		}
		rateLimits = append(rateLimits, rl)
	}

	values := map[string]interface{}{
		"firewall": map[string]interface{}{
			"internalPrefixes": internalPrefixes,
			"rateLimits":       rateLimits,
		},
		"limitValidatingWebhook": map[string]interface{}{
			"enabled": vp.controllerConfig.Auth.Enabled,
			"url":     limitURL,
			"ca":      limitCABundle,
		},
		"groupManager": map[string]interface{}{
			"enabled": vp.controllerConfig.Auth.Enabled,
		},
		"accountingExporter": map[string]interface{}{
			"enabled": vp.controllerConfig.AccountingExporter.Enabled,
		},
		"splunkAuditWebhook": map[string]interface{}{
			"enabled": vp.controllerConfig.SplunkAudit.Enabled,
			"url":     splunkURL,
			"ca":      splunkCABundle,
		},
	}

	return values, nil
}

func (vp *valuesProvider) deployControlPlaneShootDroptailerCerts(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) error {
	// TODO: There is actually no nice way to deploy the certs into the shoot when we want to use
	// the certificate helper functions from Gardener itself...
	// Maybe we can find a better solution? This is actually only for chart values...

	wanted := &secrets.Secrets{
		CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
			v1alpha1constants.SecretNameCACluster: {
				Name:       v1alpha1constants.SecretNameCACluster,
				CommonName: "kubernetes",
				CertType:   secrets.CACert,
			},
		},
		SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
			return []secrets.ConfigInterface{
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         droptailerClientSecretName,
						CommonName:   "droptailer",
						Organization: []string{"droptailer-client"},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         droptailerServerSecretName,
						CommonName:   "droptailer",
						Organization: []string{"droptailer-server"},
						CertType:     secrets.ServerCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
			}
		},
	}

	shootConfig, _, err := util.NewClientForShoot(ctx, vp.client, cluster.ObjectMeta.Name, client.Options{})
	if err != nil {
		return errors.Wrap(err, "could not create shoot client")
	}

	cs, err := kubernetes.NewForConfig(shootConfig)
	if err != nil {
		return errors.Wrap(err, "could not create shoot kubernetes client")
	}
	gcs, err := gardenerkubernetes.NewWithConfig(gardenerkubernetes.WithRESTConfig(shootConfig))
	if err != nil {
		return errors.Wrap(err, "could not create shoot Gardener client")
	}

	_, err = cs.CoreV1().Namespaces().Get(droptailerNamespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: droptailerNamespace,
				},
			}
			_, err := cs.CoreV1().Namespaces().Create(ns)
			if err != nil {
				return errors.Wrap(err, "could not create droptailer namespace")
			}
		} else {
			return errors.Wrap(err, "could not search for existence of droptailer namespace")
		}
	}

	_, err = wanted.Deploy(ctx, cs, gcs, droptailerNamespace)
	if err != nil {
		return errors.Wrap(err, "could not deploy droptailer secrets to shoot cluster")
	}

	return nil
}

// getSecret returns the secret with the given namespace/secretName
func (vp *valuesProvider) getSecret(ctx context.Context, namespace string, secretName string) (*corev1.Secret, error) {
	key := kutil.Key(namespace, secretName)
	vp.logger.Info("GetSecret", "key", key)
	secret := &corev1.Secret{}
	err := vp.mgr.GetClient().Get(ctx, key, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			vp.logger.Error(err, "error getting chart secret - not found")
			return nil, err
		}
		vp.logger.Error(err, "error getting chart secret")
		return nil, err
	}
	return secret, nil
}

// GetStorageClassesChartValues returns the values for the storage classes chart applied by the generic actuator.
func (vp *valuesProvider) GetStorageClassesChartValues(context.Context, *extensionsv1alpha1.ControlPlane, *extensionscontroller.Cluster) (map[string]interface{}, error) {
	return nil, nil
}

// getCCMChartValues collects and returns the CCM chart values.
func getCCMChartValues(
	cpConfig *apismetal.ControlPlaneConfig,
	infrastructureConfig *apismetal.InfrastructureConfig,
	infrastructure *extensionsv1alpha1.Infrastructure,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
	mclient *metalgo.Driver,
	mcp *apismetal.MetalControlPlane,
) (map[string]interface{}, error) {
	projectID := infrastructureConfig.ProjectID
	nodeCIDR := infrastructure.Status.NodesCIDR

	if nodeCIDR == nil {
		return nil, fmt.Errorf("nodeCIDR was not yet set by infrastructure controller")
	}

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, *nodeCIDR)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"cloudControllerManager": map[string]interface{}{
			"replicas":    extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"projectID":   projectID,
			"clusterID":   cluster.Shoot.ObjectMeta.UID,
			"partitionID": infrastructureConfig.PartitionID,
			"networkID":   *privateNetwork.ID,
			"podNetwork":  extensionscontroller.GetPodNetwork(cluster),
			"metal": map[string]interface{}{
				"endpoint": mcp.Endpoint,
			},
			"podAnnotations": map[string]interface{}{
				"checksum/secret-cloud-controller-manager":        checksums[cloudControllerManagerDeploymentName],
				"checksum/secret-cloud-controller-manager-server": checksums[cloudControllerManagerServerName],
				"checksum/secret-cloudprovider":                   checksums[v1alpha1constants.SecretNameCloudProvider],
				"checksum/configmap-cloud-provider-config":        checksums[metal.CloudProviderConfigName],
			},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	return values, nil
}

// returns values for "authn-webhook" and "group-manager" that are thematically related
func getAuthNGroupRoleChartValues(cpConfig *apismetal.ControlPlaneConfig, cluster *extensionscontroller.Cluster, config config.Auth) (map[string]interface{}, error) {
	annotations := cluster.Shoot.GetAnnotations()
	clusterName := annotations[tag.ClusterName]
	tenant := annotations[tag.ClusterTenant]

	ti := cpConfig.IAMConfig.IssuerConfig

	values := map[string]interface{}{
		"authnWebhook": map[string]interface{}{
			"enabled":        config.Enabled,
			"tenant":         tenant,
			"providerTenant": config.ProviderTenant,
			"clusterName":    clusterName,
			"oidc": map[string]interface{}{
				"issuerUrl":      ti.Url,
				"issuerClientId": ti.ClientId,
			},
		},

		"groupManager": map[string]interface{}{
			"enabled":          config.Enabled,
			"clusterName":      clusterName,
			"providerOperated": true, // TODO
			"providerTenant":   config.ProviderTenant,
			"idmPassword":      "", // TODO
			"idmUser":          "", // TODO
			"tls": map[string]interface{}{
				"ca":      "", // TODO
				"cert":    "", // TODO
				"certKey": "", // TODO
			},
			"idmProviderSettings": map[string]interface{}{
				"apiURL":            "", // TODO
				"owner":             "", // TODO
				"jobInfo":           "", // TODO
				"domainName":        "", // TODO
				"targetSystemID":    "", // TODO
				"type":              "", // TODO
				"requestSystem":     "", // TODO
				"requestEMail":      "", // TODO
				"accessCode":        "", // TODO
				"customerID":        "", // TODO
				"group":             "", // TODO
				"groupNameTemplate": "", // TODO
			},
			"idmTenantSettings": map[string]interface{}{
				"apiURL":            "", // TODO
				"owner":             "", // TODO (contained in garden.sapcloud.io/owner annotation)
				"jobInfo":           "", // TODO
				"domainName":        "", // TODO
				"targetSystemID":    "", // TODO
				"type":              "", // TODO
				"requestSystem":     "", // TODO
				"requestEMail":      "", // TODO (contained in garden.sapcloud.io/owner annotation)
				"accessCode":        "", // TODO
				"customerID":        "", // TODO
				"group":             "", // TODO
				"groupNameTemplate": "", // TODO
				"tenantPrefix":      tenant,
			},
		},
	}

	return values, nil
}

// returns values for "splunk-audit-webhook"
func getSplunkAuditChartValues(cpConfig *apismetal.ControlPlaneConfig, cluster *extensionscontroller.Cluster, config config.SplunkAudit) (map[string]interface{}, error) {

	values := map[string]interface{}{
		"splunkAuditWebhook": map[string]interface{}{
			"enabled": config.Enabled,
			"hecEndpoint": map[string]interface{}{
				"url":   config.HecURL,
				"token": config.HecToken,
			},
		},
	}

	return values, nil
}

func getAccountingExporterChartValues(accountingConfig config.AccountingExporterConfiguration, cluster *extensionscontroller.Cluster, infrastructure *apismetal.InfrastructureConfig, mclient *metalgo.Driver) (map[string]interface{}, error) {
	annotations := cluster.Shoot.GetAnnotations()
	partitionID := infrastructure.PartitionID
	projectID := infrastructure.ProjectID
	clusterID := cluster.Shoot.ObjectMeta.UID
	clusterName := annotations[tag.ClusterName]
	tenant := annotations[tag.ClusterTenant]

	values := map[string]interface{}{
		"accountingExporter": map[string]interface{}{
			"enabled": accountingConfig.Enabled,
			"networkTraffic": map[string]interface{}{
				"enabled": accountingConfig.NetworkTraffic.Enabled,
			},
			"enrichments": map[string]interface{}{
				"partitionID": partitionID,
				"tenant":      tenant,
				"projectID":   projectID,
				"clusterName": clusterName,
				"clusterID":   clusterID,
			},
			"accountingAPI": map[string]interface{}{
				"hostname": accountingConfig.Client.Hostname,
				"port":     accountingConfig.Client.Port,
				"ca":       accountingConfig.Client.CA,
				"cert":     accountingConfig.Client.Cert,
				"certKey":  accountingConfig.Client.CertKey,
			},
		},
	}

	return values, nil
}

func getLimitValidationWebhookControlPlaneChartValues(cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	values := map[string]interface{}{
		"limitValidatingWebhook": map[string]interface{}{
			"enabled": false, // TODO: Add to opts
		},
	}

	return values, nil
}
