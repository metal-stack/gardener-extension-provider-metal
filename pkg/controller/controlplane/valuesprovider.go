package controlplane

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	durosv1 "github.com/metal-stack/duros-controller/api/v1"
	firewallv1 "github.com/metal-stack/firewall-controller/api/v1"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/pkg/utils"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	metalgo "github.com/metal-stack/metal-go"

	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	v1alpha1constants "github.com/gardener/gardener/pkg/apis/core/v1alpha1/constants"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/go-logr/logr"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	dnsv1alpha1 "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
					Name:         metal.CloudControllerManagerDeploymentName,
					CommonName:   "system:cloud-controller-manager",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1alpha1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         metal.DurosControllerDeploymentName,
					CommonName:   "system:duros-controller",
					DNSNames:     kutil.DNSNamesForService(metal.DurosControllerDeploymentName, clusterName),
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1alpha1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         metal.AudittailerClientSecretName,
					CommonName:   "audittailer",
					DNSNames:     []string{"audittailer"},
					Organization: []string{"audittailer-client"},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1alpha1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         metal.GroupRolebindingControllerName,
					CommonName:   "system:group-rolebinding-controller",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1alpha1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       metal.AuthNWebhookServerName,
					CommonName: metal.AuthNWebhookDeploymentName,
					DNSNames:   kutil.DNSNamesForService(metal.AuthNWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[v1alpha1constants.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       metal.AccountingExporterName,
					CommonName: "system:accounting-exporter",
					// Groupname of user
					Organization: []string{metal.AccountingExporterName},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
				},
				KubeConfigRequests: []secrets.KubeConfigRequest{
					{
						ClusterName:   clusterName,
						APIServerHost: v1alpha1constants.DeploymentNameKubeAPIServer,
					},
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       metal.CloudControllerManagerServerName,
					CommonName: metal.CloudControllerManagerDeploymentName,
					DNSNames:   kutil.DNSNamesForService(metal.CloudControllerManagerDeploymentName, clusterName),
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

		// node-init
		{Type: &corev1.ServiceAccount{}, Name: "node-init"},
		{Type: &policyv1beta1.PodSecurityPolicy{}, Name: "node-init"},
		{Type: &rbacv1.ClusterRole{}, Name: "kube-system:node-init"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "kube-system:node-init"},
		{Type: &appsv1.DaemonSet{}, Name: "node-init"},
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

type networkMap map[string]*models.V1NetworkResponse

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(mgr manager.Manager, logger logr.Logger, controllerConfig config.ControllerConfiguration) genericactuator.ValuesProvider {
	if controllerConfig.Auth.Enabled {
		configChart.Objects = append(configChart.Objects, []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: "authn-webhook-config"},
		}...)
		controlPlaneChart.Images = append(controlPlaneChart.Images, []string{
			metal.AuthNWebhookImageName,
			metal.GroupRolebindingControllerImageName,
		}...)
		controlPlaneChart.Objects = append(controlPlaneChart.Objects, []*chart.Object{
			// authn webhook
			{Type: &appsv1.Deployment{}, Name: "kube-jwt-authn-webhook"},
			{Type: &corev1.Service{}, Name: "kube-jwt-authn-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kubeapi2kube-jwt-authn-webhook"},
			{Type: &networkingv1.NetworkPolicy{}, Name: "kube-jwt-authn-webhook-allow-namespace"},

			// group rolebinding controller
			{Type: &appsv1.Deployment{}, Name: "group-rolebinding-controller"},
		}...)
		cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
			// group rolebinding controller
			{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:group-rolebinding-controller"},
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
	if controllerConfig.Storage.Duros.Enabled {
		controlPlaneChart.Images = append(controlPlaneChart.Images, []string{metal.DurosControllerImageName}...)
		controlPlaneChart.Objects = append(controlPlaneChart.Objects, []*chart.Object{
			// duros storage
			{Type: &corev1.ServiceAccount{}, Name: "duros-controller"},
			{Type: &rbacv1.Role{}, Name: "duros-controller"},
			{Type: &rbacv1.RoleBinding{}, Name: "duros-controller"},
			{Type: &corev1.Secret{}, Name: "duros-admin"},
			{Type: &appsv1.Deployment{}, Name: "duros-controller"},
			{Type: &durosv1.Duros{}, Name: metal.DurosResourceName},
			{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-storage"},
		}...)
		cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
			{Type: &rbacv1.ClusterRole{}, Name: "system:duros-controller"},
			{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:duros-controller"},
		}...)
	}
	if controllerConfig.ClusterAudit.Enabled {
		configChart.Objects = append(configChart.Objects, []*chart.Object{
			{Type: &corev1.ConfigMap{}, Name: "audit-policy-override"},
		}...)
		cpShootChart.Images = append(cpShootChart.Images, []string{metal.AudittailerImageName}...)
		cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
			// audittailer
			{Type: &corev1.Namespace{}, Name: "audit"},
			{Type: &appsv1.Deployment{}, Name: "audittailer"},
			{Type: &corev1.ConfigMap{}, Name: "audittailer-config"},
			{Type: &corev1.Service{}, Name: "audittailer"},
			{Type: &rbacv1.Role{}, Name: "audittailer"},
			{Type: &rbacv1.RoleBinding{}, Name: "audittailer"},
		}...)
		if controllerConfig.AuditToSplunk.Enabled {
			configChart.Objects = append(configChart.Objects, []*chart.Object{
				{Type: &corev1.Secret{}, Name: "audit-to-splunk-secret"},
				{Type: &corev1.ConfigMap{}, Name: "audit-to-splunk-config"},
			}...)
		}
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

	clusterAuditValues, err := vp.getClusterAuditConfigValues(ctx, cp, cluster)
	if err != nil {
		return nil, err
	}

	merge(authValues, clusterAuditValues)
	return authValues, nil
}

func (vp *valuesProvider) getAuthNConfigValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	namespace := cluster.ObjectMeta.Name

	// this should work as the kube-apiserver is a pod in the same cluster as the kube-jwt-authn-webhook
	// example https://kube-jwt-authn-webhook.shoot--local--myshootname.svc.cluster.local/authenticate
	url := fmt.Sprintf("https://%s.%s.svc.cluster.local/authenticate", metal.AuthNWebhookDeploymentName, namespace)

	values := map[string]interface{}{
		"authnWebhook": map[string]interface{}{
			"url":     url,
			"enabled": vp.controllerConfig.Auth.Enabled,
		},
	}

	return values, nil
}

func (vp *valuesProvider) getClusterAuditConfigValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	cpConfig, err := helper.ControlPlaneConfigFromControlPlane(cp)
	if err != nil {
		return nil, err
	}

	var (
		clusterAuditValues = map[string]interface{}{
			"enabled": false,
		}
		auditToSplunkValues = map[string]interface{}{
			"enabled": false,
		}
		values = map[string]interface{}{
			"clusterAudit":  clusterAuditValues,
			"auditToSplunk": auditToSplunkValues,
		}
	)

	if !validation.ClusterAuditEnabled(&vp.controllerConfig, cpConfig) {
		return values, nil
	}

	clusterAuditValues["enabled"] = true

	if !validation.AuditToSplunkEnabled(&vp.controllerConfig, cpConfig) {
		return values, nil
	}

	auditToSplunkValues["enabled"] = true
	auditToSplunkValues["hecToken"] = vp.controllerConfig.AuditToSplunk.HECToken
	auditToSplunkValues["index"] = vp.controllerConfig.AuditToSplunk.Index
	auditToSplunkValues["hecHost"] = vp.controllerConfig.AuditToSplunk.HECHost
	auditToSplunkValues["hecPort"] = vp.controllerConfig.AuditToSplunk.HECPort
	auditToSplunkValues["tlsEnabled"] = vp.controllerConfig.AuditToSplunk.TLSEnabled
	auditToSplunkValues["hecCAFile"] = vp.controllerConfig.AuditToSplunk.HECCAFile
	auditToSplunkValues["clusterName"] = cluster.ObjectMeta.Name

	if !extensionscontroller.IsHibernated(cluster) {
		values["auditToSplunk"], err = vp.getCustomSplunkValues(ctx, cluster.ObjectMeta.Name, auditToSplunkValues)
		if err != nil {
			vp.logger.Error(err, "could not read custom splunk values")
		}
	}

	return values, nil
}

func (vp *valuesProvider) getCustomSplunkValues(ctx context.Context, clusterName string, auditToSplunkValues map[string]interface{}) (map[string]interface{}, error) {
	shootConfig, _, err := util.NewClientForShoot(ctx, vp.client, clusterName, client.Options{})
	if err != nil {
		return auditToSplunkValues, err
	}

	cs, err := kubernetes.NewForConfig(shootConfig)
	if err != nil {
		return auditToSplunkValues, err
	}

	splunkConfigSecret, err := cs.CoreV1().Secrets("kube-system").Get(ctx, "splunk-config", metav1.GetOptions{})
	if err != nil {
		return auditToSplunkValues, nil
	}

	if splunkConfigSecret.Data == nil {
		vp.logger.Error(errors.New("secret is empty"), "custom splunk config secret contains no data")
		return auditToSplunkValues, nil
	}

	for key, value := range splunkConfigSecret.Data {
		switch key {
		case "hecToken":
			auditToSplunkValues[key] = string(value)
		case "index":
			auditToSplunkValues[key] = string(value)
		case "hecHost":
			auditToSplunkValues[key] = string(value)
		case "hecPort":
			auditToSplunkValues[key] = string(value)
		case "tlsEnabled":
			auditToSplunkValues[key] = string(value)
		case "hecCAFile":
			auditToSplunkValues[key] = string(value)
		}
	}

	return auditToSplunkValues, nil
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
		return nil, fmt.Errorf("could not decode providerConfig of infrastructure %w", err)
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

	metalCredentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, vp.client, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClientFromCredentials(metalControlPlane.Endpoint, metalCredentials)
	if err != nil {
		return nil, err
	}

	resp, err := mclient.NetworkList()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve networks from metal-api %w", err)
	}

	nws := networkMap{}
	for _, n := range resp.Networks {
		n := n
		nws[*n.ID] = n
	}

	// TODO: this is a workaround to speed things for the time being...
	// the infrastructure controller writes the nodes cidr back into the infrastructure status, but the cluster resource does not contain it immediately
	// it would need the start of another reconcilation until the node cidr can be picked up from the cluster resource
	// therefore, we read it directly from the infrastructure status
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := vp.client.Get(ctx, kutil.Key(cp.Namespace, cp.Name), infrastructure); err != nil {
		return nil, err
	}

	p, err := mclient.ProjectGet(infrastructureConfig.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve project from metal-api %w", err)
	}

	chartValues, err := getCCMChartValues(cpConfig, infrastructureConfig, infrastructure, cp, cluster, checksums, scaledDown, mclient, metalControlPlane, nws)
	if err != nil {
		return nil, err
	}

	metalEndpoint := metalControlPlane.Endpoint
	ma := metalAccess{
		url:          metalEndpoint,
		hmac:         metalCredentials.MetalAPIHMac,
		hmacAuthType: "", // currently default is used
		apiToken:     metalCredentials.MetalAPIKey,
	}
	authValues, err := getAuthNGroupRoleChartValues(cpConfig, cluster, vp.controllerConfig.Auth, p.Project, ma)
	if err != nil {
		return nil, err
	}

	accValues, err := getAccountingExporterChartValues(ctx, vp.client, vp.controllerConfig.AccountingExporter, cluster, infrastructureConfig, p.Project)
	if err != nil {
		return nil, err
	}

	storageValues, err := getStorageControlPlaneChartValues(ctx, vp.client, vp.logger, vp.controllerConfig.Storage, cluster, infrastructureConfig, cpConfig, nws)
	if err != nil {
		return nil, err
	}

	merge(chartValues, authValues, accValues, storageValues)

	if vp.controllerConfig.ImagePullSecret != nil {
		chartValues["imagePullSecret"] = vp.controllerConfig.ImagePullSecret.DockerConfigJSON
	}

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
	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := vp.decoder.Decode(cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return nil, fmt.Errorf("could not decode providerConfig of infrastructure %w", err)
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

	mclient, err := metalclient.NewClient(ctx, vp.client, metalControlPlane.Endpoint, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	resp, err := mclient.NetworkList()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve networks from metal-api %w", err)
	}

	nws := networkMap{}
	for _, n := range resp.Networks {
		n := n
		nws[*n.ID] = n
	}

	values, err := vp.getControlPlaneShootChartValues(ctx, metalControlPlane, cp, cpConfig, cluster, nws, infrastructureConfig, mclient)
	if err != nil {
		vp.logger.Error(err, "Error getting shoot control plane chart values")
		return nil, err
	}

	if !extensionscontroller.IsHibernated(cluster) {
		if validation.ClusterAuditEnabled(&vp.controllerConfig, cpConfig) {
			if err := vp.deployControlPlaneShootAudittailerCerts(ctx, cp, cluster); err != nil {
				vp.logger.Error(err, "error deploying audittailer certs")
			}
		}

		if err := vp.deployControlPlaneShootDroptailerCerts(ctx, cp, cluster); err != nil {
			vp.logger.Error(err, "error deploying droptailer certs")
		}
	}

	return values, nil
}

// getControlPlaneShootChartValues returns the values for the shoot control plane chart.
func (vp *valuesProvider) getControlPlaneShootChartValues(ctx context.Context, metalControlPlane *apismetal.MetalControlPlane, cp *extensionsv1alpha1.ControlPlane, cpConfig *apismetal.ControlPlaneConfig, cluster *extensionscontroller.Cluster, nws networkMap, infrastructure *apismetal.InfrastructureConfig, mclient *metalgo.Driver) (map[string]interface{}, error) {
	namespace := cluster.ObjectMeta.Name

	fwSpec, err := vp.getFirewallSpec(ctx, metalControlPlane, infrastructure, cluster, nws, mclient)
	if err != nil {
		return nil, fmt.Errorf("could not assemble firewall values %w", err)
	}

	err = vp.signFirewallValues(ctx, namespace, fwSpec)
	if err != nil {
		return nil, fmt.Errorf("could not sign firewall values %w", err)
	}

	durosValues := map[string]interface{}{
		"enabled": vp.controllerConfig.Storage.Duros.Enabled,
	}

	clusterAuditValues := map[string]interface{}{
		"enabled": false,
	}
	if validation.ClusterAuditEnabled(&vp.controllerConfig, cpConfig) {
		clusterAuditValues["enabled"] = true
	}

	apiserverIPs := []string{}
	if !extensionscontroller.IsHibernated(cluster) {
		// get apiserver ip adresses from external dns entry
		dns := &dnsv1alpha1.DNSEntry{}

		err = vp.client.Get(ctx, types.NamespacedName{Name: "external", Namespace: namespace}, dns)
		if err != nil {
			return nil, fmt.Errorf("failed to get dnsEntry %w", err)
		}
		apiserverIPs = dns.Spec.Targets
	}

	values := map[string]interface{}{
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"apiserverIPs":      apiserverIPs,
		"firewallSpec":      fwSpec,
		"groupRolebindingController": map[string]interface{}{
			"enabled": vp.controllerConfig.Auth.Enabled,
		},
		"accountingExporter": map[string]interface{}{
			"enabled": vp.controllerConfig.AccountingExporter.Enabled,
		},
		"duros":        durosValues,
		"clusterAudit": clusterAuditValues,
	}

	if vp.controllerConfig.Storage.Duros.Enabled {
		partitionConfig, ok := vp.controllerConfig.Storage.Duros.PartitionConfig[infrastructure.PartitionID]

		found, err := hasDurosStorageNetwork(infrastructure, nws)
		if err != nil {
			return nil, fmt.Errorf("unable to determine storage network %w", err)
		}

		if found && ok {
			durosValues["endpoints"] = partitionConfig.Endpoints
		} else {
			durosValues["enabled"] = false
		}
	}

	return values, nil
}

func (vp *valuesProvider) getFirewallSpec(ctx context.Context, metalControlPlane *apismetal.MetalControlPlane, infrastructureConfig *apismetal.InfrastructureConfig, cluster *extensionscontroller.Cluster, nws networkMap, mclient *metalgo.Driver) (*firewallv1.FirewallSpec, error) {
	internalPrefixes := []string{}
	if vp.controllerConfig.AccountingExporter.Enabled && vp.controllerConfig.AccountingExporter.NetworkTraffic.Enabled {
		internalPrefixes = vp.controllerConfig.AccountingExporter.NetworkTraffic.InternalNetworks
	}

	rateLimits := []firewallv1.RateLimit{}
	for _, rateLimit := range infrastructureConfig.Firewall.RateLimits {
		rateLimits = append(rateLimits, firewallv1.RateLimit{
			NetworkID: rateLimit.NetworkID,
			Rate:      rateLimit.RateLimit,
		})
	}

	egressRules := []firewallv1.EgressRuleSNAT{}
	for _, egressRule := range infrastructureConfig.Firewall.EgressRules {
		egressRules = append(egressRules, firewallv1.EgressRuleSNAT{
			NetworkID: egressRule.NetworkID,
			IPs:       egressRule.IPs,
		})
	}

	logAcceptedConnections := false
	if infrastructureConfig.Firewall.LogAcceptedConnections {
		logAcceptedConnections = true
	}

	clusterID := string(cluster.Shoot.GetUID())
	projectID := infrastructureConfig.ProjectID
	firewalls, err := metalclient.FindClusterFirewalls(mclient, clusterTag(clusterID), projectID)
	if err != nil {
		return nil, fmt.Errorf("could not find firewall for cluster %w", err)
	}
	if len(firewalls) != 1 {
		return nil, fmt.Errorf("cluster %s has %d firewalls", clusterID, len(firewalls))
	}

	firewall := *firewalls[0]
	firewallNetworks := []firewallv1.FirewallNetwork{}
	for _, n := range firewall.Allocation.Networks {
		if n.Networkid == nil {
			continue
		}
		n := n

		// prefixes in the firewall machine allocation are just a snapshot when the firewall was created.
		// -> when changing prefixes in the referenced network the firewall does not know about any prefix changes.
		//
		// we replace the prefixes from the snapshot with the actual prefixes that are currently attached to the network.
		// this allows dynamic prefix reconfiguration of the firewall.
		prefixes := n.Prefixes
		networkRef, ok := nws[*n.Networkid]
		if !ok {
			vp.logger.Info("network in firewall allocation does not exist anymore")
		} else {
			prefixes = networkRef.Prefixes
		}

		firewallNetworks = append(firewallNetworks, firewallv1.FirewallNetwork{
			Asn:                 n.Asn,
			Destinationprefixes: n.Destinationprefixes,
			Ips:                 n.Ips,
			Nat:                 n.Nat,
			Networkid:           n.Networkid,
			Networktype:         n.Networktype,
			Prefixes:            prefixes,
			Vrf:                 n.Vrf,
		})
	}

	spec := firewallv1.FirewallSpec{
		Data: firewallv1.Data{
			Interval:         "10s",
			FirewallNetworks: firewallNetworks,
			InternalPrefixes: internalPrefixes,
			RateLimits:       rateLimits,
			EgressRules:      egressRules,
		},
		LogAcceptedConnections: logAcceptedConnections,
	}

	fwcv, err := validation.ValidateFirewallControllerVersion(metalControlPlane.FirewallControllerVersions, infrastructureConfig.Firewall.ControllerVersion)
	if err != nil {
		return nil, err
	}

	spec.ControllerVersion = fwcv.Version
	spec.ControllerURL = fwcv.URL

	return &spec, nil
}

func (vp *valuesProvider) signFirewallValues(ctx context.Context, namespace string, spec *firewallv1.FirewallSpec) error {
	secret, err := vp.getSecret(ctx, namespace, v1alpha1constants.SecretNameCACluster)
	if err != nil {
		return fmt.Errorf("could not find ca secret for signing firewall values %w", err)
	}

	privateKey, err := utils.DecodePrivateKey(secret.Data[secrets.DataKeyPrivateKeyCA])
	if err != nil {
		return fmt.Errorf("could not decode private key from ca secret for signing firewall values %w", err)
	}

	vp.logger.Info("signing firewall", "data", spec.Data)
	signature, err := spec.Data.Sign(privateKey)
	if err != nil {
		return fmt.Errorf("could not sign firewall values %w", err)
	}

	spec.Signature = signature
	return nil
}

func (vp *valuesProvider) deployControlPlaneShootAudittailerCerts(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) error {
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
						Name:         metal.AudittailerClientSecretName,
						CommonName:   "audittailer",
						DNSNames:     []string{"audittailer"},
						Organization: []string{"audittailer-client"},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         metal.AudittailerServerSecretName,
						CommonName:   "audittailer",
						DNSNames:     []string{"audittailer"},
						Organization: []string{"audittailer-server"},
						CertType:     secrets.ServerCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
			}
		},
	}

	return vp.deploySecretsToShoot(ctx, cluster, metal.AudittailerNamespace, wanted)
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
						Name:         metal.DroptailerClientSecretName,
						CommonName:   "droptailer",
						DNSNames:     []string{"droptailer"},
						Organization: []string{"droptailer-client"},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         metal.DroptailerServerSecretName,
						CommonName:   "droptailer",
						DNSNames:     []string{"droptailer"},
						Organization: []string{"droptailer-server"},
						CertType:     secrets.ServerCert,
						SigningCA:    cas[v1alpha1constants.SecretNameCACluster],
					},
				},
			}
		},
	}

	return vp.deploySecretsToShoot(ctx, cluster, metal.DroptailerNamespace, wanted)
}

func (vp *valuesProvider) deploySecretsToShoot(ctx context.Context, cluster *extensionscontroller.Cluster, namespace string, wanted *secrets.Secrets) error {
	shootConfig, _, err := util.NewClientForShoot(ctx, vp.client, cluster.ObjectMeta.Name, client.Options{})
	if err != nil {
		return fmt.Errorf("could not create shoot client %w", err)
	}

	cs, err := kubernetes.NewForConfig(shootConfig)
	if err != nil {
		return fmt.Errorf("could not create shoot kubernetes client %w", err)
	}

	gcs, err := gardenerkubernetes.NewWithConfig(gardenerkubernetes.WithRESTConfig(shootConfig))
	if err != nil {
		return fmt.Errorf("could not create shoot Gardener client %w", err)
	}

	_, err = cs.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			}
			_, err := cs.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("could not create namespace %w", err)
			}
		} else {
			return fmt.Errorf("could not search for existence of namespace %w", err)
		}
	}

	_, err = wanted.Deploy(ctx, cs, gcs, namespace)
	if err != nil {
		return fmt.Errorf("could not deploy secrets to shoot cluster %w", err)
	}

	return nil
}

// getSecret returns the secret with the given namespace/secretName
func (vp *valuesProvider) getSecret(ctx context.Context, namespace string, secretName string) (*corev1.Secret, error) {
	key := kutil.Key(namespace, secretName)
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
func (vp *valuesProvider) GetStorageClassesChartValues(_ context.Context, controlPlane *extensionsv1alpha1.ControlPlane, _ *extensionscontroller.Cluster) (map[string]interface{}, error) {
	cp, err := helper.ControlPlaneConfigFromControlPlane(controlPlane)
	if err != nil {
		return nil, err
	}

	isDefaultSC := true
	if cp.CustomDefaultStorageClass != nil && cp.CustomDefaultStorageClass.ClassName != "csi-lvm" {
		isDefaultSC = false
	}

	values := map[string]interface{}{
		"isDefaultStorageClass": isDefaultSC,
	}

	return values, nil
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
	nws networkMap,
) (map[string]interface{}, error) {
	projectID := infrastructureConfig.ProjectID
	nodeCIDR := infrastructure.Status.NodesCIDR

	if nodeCIDR == nil {
		if cluster.Shoot.Spec.Networking.Nodes == nil {
			return nil, fmt.Errorf("nodeCIDR was not yet set by infrastructure controller")
		}
		nodeCIDR = cluster.Shoot.Spec.Networking.Nodes
	}

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, *nodeCIDR)
	if err != nil {
		return nil, err
	}

	var defaultExternalNetwork string
	if cpConfig.CloudControllerManager != nil && cpConfig.CloudControllerManager.DefaultExternalNetwork != nil {
		defaultExternalNetwork = *cpConfig.CloudControllerManager.DefaultExternalNetwork
		resp, err := mclient.NetworkGet(defaultExternalNetwork)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve user-given default external network: %s %w", defaultExternalNetwork, err)
		}

		if resp.Network.Shared && resp.Network.Partitionid != infrastructureConfig.PartitionID {
			return nil, fmt.Errorf("shared external network must be in same partition as shoot")
		}

		if resp.Network.Projectid != "" && resp.Network.Projectid != infrastructureConfig.ProjectID && !resp.Network.Shared {
			return nil, fmt.Errorf("cannot define default external unshared network of another project")
		}

		if (resp.Network.Underlay != nil && *resp.Network.Underlay) || (resp.Network.Privatesuper != nil && *resp.Network.Privatesuper) {
			return nil, fmt.Errorf("cannot declare underlay or private super networks as default external network")
		}
	} else {
		var dmzNetwork string
		for _, networkID := range infrastructureConfig.Firewall.Networks {
			nw, ok := nws[networkID]
			if !ok {
				return nil, fmt.Errorf("network defined in firewall networks does not exist in metal-api")
			}
			for k := range nw.Labels {
				if k == tag.NetworkDefaultExternal {
					if nw.Parentnetworkid != "" {
						pn, ok := nws[nw.Parentnetworkid]
						if !ok {
							return nil, fmt.Errorf("network defined in firewall networks specified a parent network that does not exist in metal-api")
						}
						if *pn.Privatesuper {
							dmzNetwork = networkID
						}
					} else {
						defaultExternalNetwork = networkID
					}
					break
				}
			}
		}
		// fallback to a dmz network with the NetworkDefaultExternal tag
		if defaultExternalNetwork == "" && dmzNetwork != "" {
			defaultExternalNetwork = dmzNetwork
		}
		if defaultExternalNetwork == "" {
			return nil, fmt.Errorf("unable to find a default external network for metal-ccm deployment")
		}
	}

	values := map[string]interface{}{
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"cloudControllerManager": map[string]interface{}{
			"replicas":               extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"projectID":              projectID,
			"clusterID":              cluster.Shoot.ObjectMeta.UID,
			"partitionID":            infrastructureConfig.PartitionID,
			"networkID":              *privateNetwork.ID,
			"podNetwork":             extensionscontroller.GetPodNetwork(cluster),
			"defaultExternalNetwork": defaultExternalNetwork,
			"additionalNetworks":     strings.Join(infrastructureConfig.Firewall.Networks, ","),
			"metal": map[string]interface{}{
				"endpoint": mcp.Endpoint,
			},
			"podAnnotations": map[string]interface{}{
				"checksum/secret-cloud-controller-manager":        checksums[metal.CloudControllerManagerDeploymentName],
				"checksum/secret-cloud-controller-manager-server": checksums[metal.CloudControllerManagerServerName],
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

type metalAccess struct {
	url          string
	hmac         string
	hmacAuthType string
	apiToken     string
}

// returns values for "authn-webhook" and "group-rolebinding-controller" that are thematically related
func getAuthNGroupRoleChartValues(cpConfig *apismetal.ControlPlaneConfig, cluster *extensionscontroller.Cluster, config config.Auth, p *models.V1ProjectResponse, metalAccess metalAccess) (map[string]interface{}, error) {

	annotations := cluster.Shoot.GetAnnotations()
	clusterName := annotations[tag.ClusterName]

	ti := cpConfig.IAMConfig.IssuerConfig

	values := map[string]interface{}{
		"authnWebhook": map[string]interface{}{
			"enabled":        config.Enabled,
			"replicas":       extensionscontroller.GetReplicas(cluster, 1),
			"tenant":         p.TenantID,
			"providerTenant": config.ProviderTenant,
			"clusterName":    clusterName,
			"oidc": map[string]interface{}{
				"issuerUrl":      ti.Url,
				"issuerClientId": ti.ClientId,
			},
			"metalapi": map[string]interface{}{
				"url":            metalAccess.url,
				"hmac":           metalAccess.hmac,
				"hmac_auth_type": metalAccess.hmacAuthType,
				"apitoken":       metalAccess.apiToken,
			},
		},

		"groupRolebindingController": map[string]interface{}{
			"enabled":     config.Enabled,
			"replicas":    extensionscontroller.GetReplicas(cluster, 1),
			"clusterName": clusterName,
		},
	}

	return values, nil
}

func getAccountingExporterChartValues(ctx context.Context, client client.Client, accountingConfig config.AccountingExporterConfiguration, cluster *extensionscontroller.Cluster, infrastructure *apismetal.InfrastructureConfig, p *models.V1ProjectResponse) (map[string]interface{}, error) {
	annotations := cluster.Shoot.GetAnnotations()
	partitionID := infrastructure.PartitionID
	projectID := infrastructure.ProjectID
	clusterID := cluster.Shoot.ObjectMeta.UID
	clusterName := annotations[tag.ClusterName]

	if accountingConfig.Enabled {
		cp := &firewallv1.ClusterwideNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "egress-allow-accounting-api",
				Namespace: "firewall",
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, client, cp, func() error {
			port9000 := intstr.FromInt(9000)
			tcp := corev1.ProtocolTCP

			cp.Spec.Egress = []firewallv1.EgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &port9000,
							Protocol: &tcp,
						},
					},
					To: []networkingv1.IPBlock{
						{
							CIDR: "0.0.0.0/0",
						},
					},
				},
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("unable to deploy clusterwide network policy for accounting-api into firewall namespace %w", err)
		}
	}

	values := map[string]interface{}{
		"accountingExporter": map[string]interface{}{
			"enabled":  accountingConfig.Enabled,
			"replicas": extensionscontroller.GetReplicas(cluster, 1),
			"networkTraffic": map[string]interface{}{
				"enabled": accountingConfig.NetworkTraffic.Enabled,
			},
			"enrichments": map[string]interface{}{
				"partitionID": partitionID,
				"tenant":      p.TenantID,
				"projectID":   projectID,
				"projectName": p.Name,
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

func getStorageControlPlaneChartValues(ctx context.Context, client client.Client, logger logr.Logger, storageConfig config.StorageConfiguration, cluster *extensionscontroller.Cluster, infrastructure *apismetal.InfrastructureConfig, cp *apismetal.ControlPlaneConfig, nws networkMap) (map[string]interface{}, error) {
	disabledValues := map[string]interface{}{
		"duros": map[string]interface{}{
			"enabled": false,
		},
	}

	partitionConfig, ok := storageConfig.Duros.PartitionConfig[infrastructure.PartitionID]
	if !ok {
		logger.Info("skipping duros storage deployment because no storage configuration found for partition", "partition", infrastructure.PartitionID)
		return disabledValues, nil
	}

	found, err := hasDurosStorageNetwork(infrastructure, nws)
	if err != nil {
		return nil, fmt.Errorf("unable to determine storage network %w", err)
	}

	if !found {
		logger.Info("skipping duros storage deployment because no storage network found for partition")
		return disabledValues, nil
	}

	if storageConfig.Duros.Enabled {
		cp := &firewallv1.ClusterwideNetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "allow-to-storage",
				Namespace: "firewall",
			},
		}

		_, err := controllerutil.CreateOrUpdate(ctx, client, cp, func() error {
			var to []networkingv1.IPBlock
			for _, e := range partitionConfig.Endpoints {
				withoutPort := strings.Split(e, ":")
				to = append(to, networkingv1.IPBlock{
					CIDR: withoutPort[0] + "/32",
				})
			}

			port443 := intstr.FromInt(443)
			port4420 := intstr.FromInt(4420)
			port8009 := intstr.FromInt(8009)
			tcp := corev1.ProtocolTCP

			cp.Spec.Egress = []firewallv1.EgressRule{
				{
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Port:     &port443,
							Protocol: &tcp,
						},
						{
							Port:     &port4420,
							Protocol: &tcp,
						},
						{
							Port:     &port8009,
							Protocol: &tcp,
						},
					},
					To: to,
				},
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("unable to deploy clusterwide network policy for duros storage into firewall namespace %w", err)
		}
	}

	var scs []map[string]interface{}
	for _, sc := range partitionConfig.StorageClasses {

		isDefaultSC := false
		if cp.CustomDefaultStorageClass != nil && cp.CustomDefaultStorageClass.ClassName == sc.Name {
			isDefaultSC = true
		}

		scs = append(scs, map[string]interface{}{
			"name":        sc.Name,
			"replicas":    sc.ReplicaCount,
			"compression": sc.Compression,
			"default":     isDefaultSC,
		})
	}

	controllerValues := map[string]interface{}{
		"endpoints":  partitionConfig.Endpoints,
		"adminKey":   partitionConfig.AdminKey,
		"adminToken": partitionConfig.AdminToken,
	}

	if partitionConfig.APIEndpoint != nil {
		controllerValues["apiEndpoint"] = *partitionConfig.APIEndpoint
		controllerValues["apiCA"] = partitionConfig.APICA
		controllerValues["apiKey"] = partitionConfig.APIKey
		controllerValues["apiCert"] = partitionConfig.APICert
	}

	values := map[string]interface{}{
		"duros": map[string]interface{}{
			"enabled":        storageConfig.Duros.Enabled,
			"replicas":       extensionscontroller.GetReplicas(cluster, 1),
			"storageClasses": scs,
			"projectID":      infrastructure.ProjectID,
			"controller":     controllerValues,
		},
	}

	return values, nil
}

func clusterTag(clusterID string) string {
	return fmt.Sprintf("%s=%s", tag.ClusterID, clusterID)
}

func hasDurosStorageNetwork(infrastructure *apismetal.InfrastructureConfig, nws networkMap) (bool, error) {
	for _, networkID := range infrastructure.Firewall.Networks {
		nw, ok := nws[networkID]
		if !ok {
			return false, fmt.Errorf("network defined in firewall networks does not exist in metal-api")
		}
		if nw.Partitionid != infrastructure.PartitionID {
			continue
		}
		for k := range nw.Labels {
			if k == tag.NetworkPartitionStorage {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetControlPlaneShootCRDsChartValues returns the values for the control plane shoot CRDs chart applied by the generic actuator.
// Currently the provider extension does not specify a control plane shoot CRDs chart. That's why we simply return empty values.
func (vp *valuesProvider) GetControlPlaneShootCRDsChartValues(
	_ context.Context,
	_ *extensionsv1alpha1.ControlPlane,
	_ *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
