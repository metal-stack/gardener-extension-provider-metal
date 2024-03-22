package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	durosv1 "github.com/metal-stack/duros-controller/api/v1"
	firewallv1 "github.com/metal-stack/firewall-controller/v2/api/v1"

	extensionsconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1helper "github.com/gardener/gardener/pkg/apis/core/v1beta1/helper"

	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	metalgo "github.com/metal-stack/metal-go"

	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"

	extensionssecretsmanager "github.com/gardener/gardener/extensions/pkg/util/secret/manager"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/chart"
	"github.com/gardener/gardener/pkg/utils/secrets"
	secretsmanager "github.com/gardener/gardener/pkg/utils/secrets/manager"

	"github.com/go-logr/logr"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	caNameControlPlane = "ca-" + metal.Name + "-controlplane"
	droptailerCAName   = "ca-" + metal.Name + "-droptailer"
	auditTailerCAName  = "ca-" + metal.Name + "-audittailer"

	ipv4HostMask = "/32"
	ipv6HostMask = "/128"
	ipv4Any      = "0.0.0.0/0"
)

func secretConfigsFunc(namespace string) []extensionssecretsmanager.SecretConfigWithOptions {
	return []extensionssecretsmanager.SecretConfigWithOptions{
		{
			Config: &secrets.CertificateSecretConfig{
				Name:       caNameControlPlane,
				CommonName: caNameControlPlane,
				CertType:   secrets.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.CloudControllerManagerServerName,
				CommonName:                  metal.CloudControllerManagerDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.CloudControllerManagerDeploymentName, namespace),
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: true,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane, secretsmanager.UseCurrentCA)},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.FirewallControllerManagerDeploymentName,
				CommonName:                  metal.FirewallControllerManagerDeploymentName,
				DNSNames:                    kutil.DNSNamesForService(metal.FirewallControllerManagerDeploymentName, namespace),
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: false,
			},
			// use current CA for signing server cert to prevent mismatches when dropping the old CA from the webhook
			// config in phase Completing
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(caNameControlPlane, secretsmanager.UseCurrentCA)},
		},
		// droptailer
		{
			Config: &secrets.CertificateSecretConfig{
				Name:       droptailerCAName,
				CommonName: droptailerCAName,
				CertType:   secrets.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.DroptailerClientSecretName,
				CommonName:                  "droptailer",
				DNSNames:                    []string{"droptailer"},
				Organization:                []string{"droptailer-client"},
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: false,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(droptailerCAName, secretsmanager.UseCurrentCA)},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.DroptailerServerSecretName,
				CommonName:                  "droptailer",
				DNSNames:                    []string{"droptailer"},
				Organization:                []string{"droptailer-server"},
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: false,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(droptailerCAName, secretsmanager.UseCurrentCA)},
		},
		// audit tailer
		{
			Config: &secrets.CertificateSecretConfig{
				Name:       auditTailerCAName,
				CommonName: auditTailerCAName,
				CertType:   secrets.CACert,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.Persist()},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.AudittailerClientSecretName,
				CommonName:                  "audittailer",
				DNSNames:                    []string{"audittailer"},
				Organization:                []string{"audittailer-client"},
				CertType:                    secrets.ClientCert,
				SkipPublishingCACertificate: false,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(auditTailerCAName, secretsmanager.UseCurrentCA)},
		},
		{
			Config: &secrets.CertificateSecretConfig{
				Name:                        metal.AudittailerServerSecretName,
				CommonName:                  "audittailer",
				DNSNames:                    []string{"audittailer"},
				Organization:                []string{"audittailer-server"},
				CertType:                    secrets.ServerCert,
				SkipPublishingCACertificate: false,
			},
			Options: []secretsmanager.GenerateOption{secretsmanager.SignedByCA(auditTailerCAName, secretsmanager.UseCurrentCA)},
		},
	}
}

func shootAccessSecretsFunc(namespace string) []*gutil.AccessSecret {
	return []*gutil.AccessSecret{
		gutil.NewShootAccessSecret(metal.FirewallControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.CloudControllerManagerDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.DurosControllerDeploymentName, namespace),
		gutil.NewShootAccessSecret(metal.MachineControllerManagerName, namespace),
		gutil.NewShootAccessSecret(metal.AudittailerClientSecretName, namespace),
	}
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
	Images: []string{metal.CCMImageName, metal.FirewallControllerManagerDeploymentName},
	Objects: []*chart.Object{
		// cloud controller manager
		{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
		{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},
	},
}

var cpShootChart = &chart.Chart{
	Name:   "shoot-control-plane",
	Path:   filepath.Join(metal.InternalChartsPath, "shoot-control-plane"),
	Images: []string{metal.DroptailerImageName, metal.MetallbSpeakerImageName, metal.MetallbControllerImageName, metal.NodeInitImageName, metal.MetallbHealthSidecarImageName},
	Objects: []*chart.Object{
		// metallb
		{Type: &corev1.Namespace{}, Name: "metallb-system"},
		{Type: &policyv1beta1.PodSecurityPolicy{}, Name: "speaker"},
		{Type: &corev1.ServiceAccount{}, Name: "controller"},
		{Type: &corev1.ServiceAccount{}, Name: "speaker"},
		{Type: &rbacv1.ClusterRole{}, Name: "metallb-system:controller"},
		{Type: &rbacv1.ClusterRole{}, Name: "metallb-system:speaker"},
		{Type: &rbacv1.Role{}, Name: "pod-lister"},
		{Type: &rbacv1.Role{}, Name: "controller"},
		{Type: &rbacv1.Role{}, Name: "health-monitoring"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "metallb-system:controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "metallb-system:speaker"},
		{Type: &rbacv1.RoleBinding{}, Name: "pod-lister"},
		{Type: &rbacv1.RoleBinding{}, Name: "controller"},
		{Type: &rbacv1.RoleBinding{}, Name: "health-monitoring"},
		{Type: &corev1.ConfigMap{}, Name: "metallb-excludel2"},
		{Type: &corev1.Secret{}, Name: "webhook-server-cert"},
		{Type: &corev1.Service{}, Name: "webhook-service"},
		{Type: &appsv1.DaemonSet{}, Name: "speaker"},
		{Type: &appsv1.Deployment{}, Name: "controller"},

		// cluster wide network policies
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-http"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-https"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-dns"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-ntp"},
		{Type: &firewallv1.ClusterwideNetworkPolicy{}, Name: "allow-to-vpn"},

		// firewall controller
		{Type: &rbacv1.ClusterRole{}, Name: "system:firewall-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:firewall-controller"},

		// firewall controller manager
		{Type: &corev1.ServiceAccount{}, Name: "firewall-controller-manager"},
		{Type: &rbacv1.Role{}, Name: "firewall-controller-manager"},
		{Type: &rbacv1.RoleBinding{}, Name: "firewall-controller-manager"},
		{Type: &appsv1.Deployment{}, Name: "firewall-controller-manager"},
		{Type: &corev1.Service{}, Name: "firewall-controller-manager"},
		{Type: &admissionregistrationv1.MutatingWebhookConfiguration{}, Name: "firewall-controller-manager-namespace"},
		{Type: &admissionregistrationv1.ValidatingWebhookConfiguration{}, Name: "firewall-controller-manager-namespace"},

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
func NewValuesProvider(mgr manager.Manager, controllerConfig config.ControllerConfiguration) genericactuator.ValuesProvider {
	cpShootChart.Objects = append(cpShootChart.Objects, []*chart.Object{
		{Type: &corev1.ConfigMap{}, Name: "shoot-info-node-cidr"},
	}...)

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
			{Type: &networkingv1.NetworkPolicy{}, Name: "egress-from-duros-controller-to-storage"},
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
		controllerConfig: controllerConfig,
		client:           mgr.GetClient(),
		decoder:          serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

// valuesProvider is a ValuesProvider that provides metal-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	genericactuator.NoopValuesProvider
	client           client.Client
	decoder          runtime.Decoder
	logger           logr.Logger
	controllerConfig config.ControllerConfiguration
}

// GetConfigChartValues returns the values for the config chart applied by the generic actuator.
func (vp *valuesProvider) GetConfigChartValues(
	ctx context.Context,
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
) (map[string]interface{}, error) {
	clusterAuditValues, err := vp.getClusterAuditConfigValues(ctx, cp, cluster)
	if err != nil {
		return nil, err
	}

	return clusterAuditValues, nil
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
		customValues, err := vp.getCustomSplunkValues(ctx, cluster.ObjectMeta.Name, auditToSplunkValues)
		if err != nil {
			vp.logger.Error(err, "could not read custom splunk values")
		} else {
			values["auditToSplunk"] = customValues
		}
	}

	return values, nil
}

func (vp *valuesProvider) getCustomSplunkValues(ctx context.Context, clusterName string, auditToSplunkValues map[string]interface{}) (map[string]interface{}, error) {
	shootConfig, _, err := util.NewClientForShoot(ctx, vp.client, clusterName, client.Options{}, extensionsconfig.RESTOptions{})
	if err != nil {
		return auditToSplunkValues, err
	}

	cs, err := kubernetes.NewForConfig(shootConfig)
	if err != nil {
		return auditToSplunkValues, err
	}

	splunkConfigSecret, err := cs.CoreV1().Secrets("kube-system").Get(ctx, "splunk-config", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return auditToSplunkValues, nil
		}
		return nil, err
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
	secretsReader secretsmanager.Reader,
	checksums map[string]string,
	scaledDown bool,
) (map[string]any, error) {
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

	metalCredentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, vp.client, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClientFromCredentials(metalControlPlane.Endpoint, metalCredentials)
	if err != nil {
		return nil, err
	}

	resp, err := mclient.Network().ListNetworks(network.NewListNetworksParams().WithContext(ctx), nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve networks from metal-api %w", err)
	}

	nws := networkMap{}
	for _, n := range resp.Payload {
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

	sshSecret, err := helper.GetLatestSSHSecret(ctx, vp.client, cp.Namespace)
	if err != nil {
		return nil, fmt.Errorf("could not find current ssh secret: %w", err)
	}

	caBundle, err := helper.GetLatestSecret(ctx, vp.client, cp.Namespace, metal.FirewallControllerManagerDeploymentName)
	if err != nil {
		return nil, fmt.Errorf("could not get ca from secret: %w", err)
	}

	ccmValues, err := getCCMChartValues(ctx, sshSecret, cpConfig, infrastructureConfig, infrastructure, cluster, checksums, scaledDown, mclient, metalControlPlane, nws, secretsReader)
	if err != nil {
		return nil, err
	}

	storageValues, err := getStorageControlPlaneChartValues(ctx, vp.client, vp.logger, vp.controllerConfig.Storage, cluster, infrastructureConfig, cpConfig, nws)
	if err != nil {
		return nil, err
	}

	firewallValues, err := vp.getFirewallControllerManagerChartValues(ctx, cluster, metalControlPlane, sshSecret, caBundle, secretsReader)
	if err != nil {
		return nil, err
	}

	values := map[string]any{
		"imagePullPolicy": helper.ImagePullPolicyFromString(vp.controllerConfig.ImagePullPolicy),
		"pspDisabled":     gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-" + metal.FirewallControllerManagerDeploymentName: checksums[metal.FirewallControllerManagerDeploymentName],
			"checksum/secret-cloudprovider":                                    checksums[v1beta1constants.SecretNameCloudProvider],
		},
		"genericTokenKubeconfigSecretName": extensionscontroller.GenericTokenKubeconfigSecretNameFromCluster(cluster),
	}

	merge(values, ccmValues, storageValues, firewallValues)

	if vp.controllerConfig.ImagePullSecret != nil {
		values["imagePullSecret"] = vp.controllerConfig.ImagePullSecret.DockerConfigJSON
	}

	return values, nil
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
	cluster *extensionscontroller.Cluster,
	secretsReader secretsmanager.Reader,
	checksums map[string]string) (map[string]interface{}, error) {
	return nil, nil
}

// GetControlPlaneShootChartValues returns the values for the control plane shoot chart applied by the generic actuator.
func (vp *valuesProvider) GetControlPlaneShootChartValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster, secretsReader secretsmanager.Reader, checksums map[string]string) (map[string]interface{}, error) {
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

	metalControlPlane, partition, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return nil, err
	}

	mclient, err := metalclient.NewClient(ctx, vp.client, metalControlPlane.Endpoint, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	resp, err := mclient.Network().ListNetworks(network.NewListNetworksParams().WithContext(ctx), nil)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve networks from metal-api %w", err)
	}

	nws := networkMap{}
	for _, n := range resp.Payload {
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

	values, err := vp.getControlPlaneShootChartValues(ctx, cpConfig, cluster, partition, nws, infrastructure, infrastructureConfig, secretsReader, checksums)
	if err != nil {
		vp.logger.Error(err, "Error getting shoot control plane chart values")
		return nil, err
	}

	return values, nil
}

// getControlPlaneShootChartValues returns the values for the shoot control plane chart.
func (vp *valuesProvider) getControlPlaneShootChartValues(ctx context.Context, cpConfig *apismetal.ControlPlaneConfig, cluster *extensionscontroller.Cluster, partition *apismetal.Partition, nws networkMap, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *apismetal.InfrastructureConfig, secretsReader secretsmanager.Reader, checksums map[string]string) (map[string]interface{}, error) {
	namespace := cluster.ObjectMeta.Name

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return nil, err
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

	nodeInitValues := map[string]any{
		"enabled": true,
	}
	if pointer.SafeDeref(pointer.SafeDeref(cluster.Shoot.Spec.Networking).Type) == "cilium" {
		nodeInitValues["enabled"] = false
	}

	apiserverIPs := []string{}
	if !extensionscontroller.IsHibernated(cluster) {
		// get apiserver ip adresses from external dns entry
		// DNSEntry was replaced by DNSRecord and will be dropped in a future gardener release
		// We can then remove reading the dns entry resources entirely
		// get apiserver ip adresses from external dns record
		dnsRecord := &extensionsv1alpha1.DNSRecord{}
		err := vp.client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-external", cluster.Shoot.Name), Namespace: namespace}, dnsRecord)
		if err != nil {
			return nil, fmt.Errorf("failed to get dnsRecord %w", err)
		}
		apiserverIPs = dnsRecord.Spec.Values

		if len(apiserverIPs) == 0 {
			return nil, fmt.Errorf("apiserver dns records were not yet reconciled")
		}
	}

	// FIXME remove this block an replace with networkAccessType
	var egressDestinations []map[string]any
	for _, dest := range vp.controllerConfig.EgressDestinations {
		dest := dest
		if dest.MatchPattern == "" && dest.MatchName == "" {
			continue
		}
		if dest.MatchPattern != "" && dest.MatchName != "" {
			dest.MatchName = ""
		}
		if dest.Port == 0 {
			dest.Port = 443
		}
		if dest.Protocol == "" {
			dest.Protocol = "TCP"
		}
		egressDestinations = append(egressDestinations, map[string]any{
			"matchName":    dest.MatchName,
			"matchPattern": dest.MatchPattern,
			"port":         dest.Port,
			"protocol":     dest.Protocol,
		})
	}

	networkAccessType := apismetal.NetworkAccessBaseline
	if cpConfig.NetworkAccessType != nil {
		networkAccessType = *cpConfig.NetworkAccessType
	}
	restrictedOrForbidden := networkAccessType != apismetal.NetworkAccessBaseline
	if restrictedOrForbidden && partition.NetworkIsolation == nil {
		return nil, fmt.Errorf("cluster-isolation is not supported in partition %q", infrastructureConfig.PartitionID)
	}

	var dnsCidrs []string
	if restrictedOrForbidden && partition.NetworkIsolation != nil {
		dnsCidrs = make([]string, len(partition.NetworkIsolation.DNSServers))
		for i, ip := range partition.NetworkIsolation.DNSServers {
			parsedIP, err := netip.ParseAddr(ip)
			if err != nil {
				return nil, fmt.Errorf("unable to parse dns ip:%w", err)
			}
			if parsedIP.Is4() {
				dnsCidrs[i] = ip + ipv4HostMask
			}
			if parsedIP.Is6() {
				dnsCidrs[i] = ip + ipv6HostMask
			}
		}
	}
	if !restrictedOrForbidden {
		dnsCidrs = []string{ipv4Any}
	}
	if len(dnsCidrs) == 0 {
		return nil, fmt.Errorf("no dns configured")
	}

	var ntpCidrs []string
	if restrictedOrForbidden && partition.NetworkIsolation != nil {
		ntpCidrs = make([]string, len(partition.NetworkIsolation.NTPServers))
		for i, ip := range partition.NetworkIsolation.NTPServers {
			parsedIP, err := netip.ParseAddr(ip)
			if err != nil {
				return nil, fmt.Errorf("unable to parse ntp ip:%w", err)
			}
			if parsedIP.Is4() {
				ntpCidrs[i] = ip + ipv4HostMask
			}
			if parsedIP.Is6() {
				ntpCidrs[i] = ip + ipv6HostMask
			}
		}
	}
	if !restrictedOrForbidden {
		ntpCidrs = []string{ipv4Any}
	}
	if len(ntpCidrs) == 0 {
		return nil, fmt.Errorf("no ntp configured")
	}

	var networkAccessMirrors []map[string]any
	if restrictedOrForbidden && partition.NetworkIsolation != nil {
		for _, r := range partition.NetworkIsolation.RegistryMirrors {
			nam, err := registryMirrorToValueMap(r)
			if err != nil {
				return nil, err
			}
			networkAccessMirrors = append(networkAccessMirrors, nam)
		}
	}

	values := map[string]any{
		"imagePullPolicy": helper.ImagePullPolicyFromString(vp.controllerConfig.ImagePullPolicy),
		"pspDisabled":     gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		"apiserverIPs":    apiserverIPs,
		"nodeCIDR":        nodeCIDR,
		"duros":           durosValues,
		"clusterAudit":    clusterAuditValues,
		"nodeInit":        nodeInitValues,
		"restrictEgress": map[string]any{ // FIXME remove
			"enabled":                cpConfig.FeatureGates.RestrictEgress != nil && *cpConfig.FeatureGates.RestrictEgress,
			"apiServerIngressDomain": "api." + *cluster.Shoot.Spec.DNS.Domain,
			"destinations":           egressDestinations,
		},
		"networkAccess": map[string]any{
			"restrictedOrForbidden": restrictedOrForbidden,
			"dnsCidrs":              dnsCidrs,
			"ntpCidrs":              ntpCidrs,
			"registryMirrors":       networkAccessMirrors,
		},
	}

	droptailerServer, serverOK := secretsReader.Get(metal.DroptailerServerSecretName)
	droptailerClient, clientOK := secretsReader.Get(metal.DroptailerClientSecretName)
	if serverOK && clientOK {
		values["droptailer"] = map[string]any{
			"podAnnotations": map[string]interface{}{
				"checksum/secret-droptailer-server": checksums[metal.DroptailerServerSecretName],
				"checksum/secret-droptailer-client": checksums[metal.DroptailerClientSecretName],
			},
			"server": map[string]any{
				"ca":   droptailerServer.Data["ca.crt"],
				"cert": droptailerServer.Data["tls.crt"],
				"key":  droptailerServer.Data["tls.key"],
			},
			"client": map[string]any{
				"ca":   droptailerClient.Data["ca.crt"],
				"cert": droptailerClient.Data["tls.crt"],
				"key":  droptailerClient.Data["tls.key"],
			},
		}
	}

	audittailerServer, serverOK := secretsReader.Get(metal.AudittailerServerSecretName)
	audittailerClient, clientOK := secretsReader.Get(metal.AudittailerClientSecretName)
	if serverOK && clientOK {
		values["audittailer"] = map[string]any{
			"podAnnotations": map[string]interface{}{
				"checksum/secret-audittailer-server": checksums[metal.AudittailerServerSecretName],
				"checksum/secret-audittailer-client": checksums[metal.AudittailerClientSecretName],
			},
			"server": map[string]any{
				"ca":   audittailerServer.Data["ca.crt"],
				"cert": audittailerServer.Data["tls.crt"],
				"key":  audittailerServer.Data["tls.key"],
			},
			"client": map[string]any{
				"ca":   audittailerClient.Data["ca.crt"],
				"cert": audittailerClient.Data["tls.crt"],
				"key":  audittailerClient.Data["tls.key"],
			},
		}
	}

	if vp.controllerConfig.Storage.Duros.Enabled {
		partitionConfig, ok := vp.controllerConfig.Storage.Duros.PartitionConfig[infrastructureConfig.PartitionID]

		found, err := hasDurosStorageNetwork(infrastructureConfig, nws)
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

// getSecret returns the secret with the given namespace/secretName
func (vp *valuesProvider) getSecret(ctx context.Context, namespace string, secretName string) (*corev1.Secret, error) {
	key := kutil.Key(namespace, secretName)
	secret := &corev1.Secret{}
	err := vp.client.Get(ctx, key, secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			vp.logger.Error(err, "error getting secret - not found")
			return nil, err
		}
		vp.logger.Error(err, "error getting secret")
		return nil, err
	}
	return secret, nil
}

// GetStorageClassesChartValues returns the values for the storage classes chart applied by the generic actuator.
func (vp *valuesProvider) GetStorageClassesChartValues(_ context.Context, controlPlane *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	cp, err := helper.ControlPlaneConfigFromControlPlane(controlPlane)
	if err != nil {
		return nil, err
	}

	isDefaultSC := true
	if cp.CustomDefaultStorageClass != nil && cp.CustomDefaultStorageClass.ClassName != "csi-lvm" {
		isDefaultSC = false
	}

	values := map[string]interface{}{
		"pspDisabled":           gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		"isDefaultStorageClass": isDefaultSC,
	}

	return values, nil
}

// getCCMChartValues collects and returns the CCM chart values.
func getCCMChartValues(
	ctx context.Context,
	sshSecret *corev1.Secret,
	cpConfig *apismetal.ControlPlaneConfig,
	infrastructureConfig *apismetal.InfrastructureConfig,
	infrastructure *extensionsv1alpha1.Infrastructure,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
	mclient metalgo.Client,
	mcp *apismetal.MetalControlPlane,
	nws networkMap,
	secretsReader secretsmanager.Reader,
) (map[string]interface{}, error) {
	projectID := infrastructureConfig.ProjectID

	nodeCIDR, err := helper.GetNodeCIDR(infrastructure, cluster)
	if err != nil {
		return nil, err
	}

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(ctx, mclient, projectID, nodeCIDR)
	if err != nil {
		return nil, err
	}

	defaultExternalNetwork, err := getDefaultExternalNetwork(nws, cpConfig, infrastructureConfig)
	if err != nil {
		return nil, err
	}

	serverSecret, found := secretsReader.Get(metal.CloudControllerManagerServerName)
	if !found {
		return nil, fmt.Errorf("secret %q not found", metal.CloudControllerManagerServerName)
	}

	values := map[string]interface{}{
		"pspDisabled": gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		"cloudControllerManager": map[string]interface{}{
			"replicas":               extensionscontroller.GetControlPlaneReplicas(cluster, scaledDown, 1),
			"projectID":              projectID,
			"clusterID":              cluster.Shoot.ObjectMeta.UID,
			"partitionID":            infrastructureConfig.PartitionID,
			"networkID":              *privateNetwork.ID,
			"podNetwork":             extensionscontroller.GetPodNetwork(cluster),
			"defaultExternalNetwork": defaultExternalNetwork,
			"additionalNetworks":     strings.Join(infrastructureConfig.Firewall.Networks, ","),
			"sshPublicKey":           string(sshSecret.Data["id_rsa.pub"]),
			"metal": map[string]interface{}{
				"endpoint": mcp.Endpoint,
			},
			"podAnnotations": map[string]interface{}{
				"checksum/secret-cloud-controller-manager":        checksums[metal.CloudControllerManagerDeploymentName],
				"checksum/secret-cloud-controller-manager-server": checksums[metal.CloudControllerManagerServerName],
				"checksum/secret-cloudprovider":                   checksums[v1beta1constants.SecretNameCloudProvider],
				"checksum/configmap-cloud-provider-config":        checksums[metal.CloudProviderConfigName],
			},
			"secrets": map[string]any{
				"server": serverSecret.Name,
			},
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
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

	var scs []map[string]any
	for _, sc := range partitionConfig.StorageClasses {
		if cp.FeatureGates.DurosStorageEncryption == nil || !*cp.FeatureGates.DurosStorageEncryption {
			if sc.Encryption {
				continue
			}
		}

		isDefaultSC := false
		if cp.CustomDefaultStorageClass != nil && cp.CustomDefaultStorageClass.ClassName == sc.Name {
			isDefaultSC = true
		}

		scs = append(scs, map[string]any{
			"name":        sc.Name,
			"replicas":    sc.ReplicaCount,
			"compression": sc.Compression,
			"encryption":  sc.Encryption,
			"default":     isDefaultSC,
		})
	}

	controllerValues := map[string]any{
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

	values := map[string]any{
		"pspDisabled": gardencorev1beta1helper.IsPSPDisabled(cluster.Shoot),
		"duros": map[string]any{
			"enabled":        storageConfig.Duros.Enabled,
			"replicas":       extensionscontroller.GetReplicas(cluster, 1),
			"storageClasses": scs,
			"projectID":      infrastructure.ProjectID,
			"controller":     controllerValues,
		},
	}

	return values, nil
}

func (vp *valuesProvider) getFirewallControllerManagerChartValues(ctx context.Context, cluster *extensionscontroller.Cluster, metalControlPlane *apismetal.MetalControlPlane, sshSecret, caBundle *corev1.Secret, secretsReader secretsmanager.Reader) (map[string]any, error) {
	if cluster.Shoot.Spec.DNS.Domain == nil {
		return nil, fmt.Errorf("cluster dns domain is not yet set")
	}

	seedApiURL := fmt.Sprintf("https://%s", os.Getenv("KUBERNETES_SERVICE_HOST"))

	// for gardener-managed clusters the KUBERNETES_SERVICE_HOST env variable
	// points to the kube-apiserver hosted in the seed's shoot namespace, which
	// is publically reachable and works just fine.
	//
	// for non-gardener-managed clusters (e.g. shoots running in GKE), the
	// KUBERNETES_SERVICE_HOST environment variable may point to an internal
	// cluster ip, which is not reachable from the internet. the firewall-controller
	// has to reach the kube-apiserver though. in these cases, a config map
	// can be provided in this seed's garden namespace to provide the external
	// ip address of the kube-apiserver.
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "seed-api-server",
			Namespace: "garden",
		},
	}
	isConfigMapConfigured := false
	err := vp.client.Get(ctx, client.ObjectKeyFromObject(cm), cm)
	if err == nil {
		url, ok := cm.Data["url"]
		if ok {
			seedApiURL = url
			isConfigMapConfigured = true
		}
	}
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	// We generally expect to get a DNS name for the seed api url.
	// This is alway true for gardener managed clusters, because the mutating webhook
	// of the api-server-proxy sets the KUBERNETES_SERVICE_HOST env variable.
	// But for Managed Seeds where the control plane resides at GKE, this is always a IP
	// in this case we set the seedAPI URL in a configmap.
	if !isConfigMapConfigured {
		u, err := url.Parse(seedApiURL)
		if err != nil {
			return nil, err
		}

		_, err = netip.ParseAddr(u.Hostname())
		if err == nil {
			// If hostname is a parsable ipaddress we error out because we need a dnsname.
			panic(fmt.Sprintf("seedApiUrl:%q is not a dns entry, exiting", seedApiURL))
		}
	}

	serverSecret, found := secretsReader.Get(metal.FirewallControllerManagerDeploymentName)
	if !found {
		return nil, fmt.Errorf("secret %q not found", metal.FirewallControllerManagerDeploymentName)
	}

	return map[string]any{
		"firewallControllerManager": map[string]any{
			// We want to throw the firewall away once the cluster is hibernated.
			// when woken up, a new firewall is created with new token, ssh key etc.
			// This will break the firewall-only case actually only used in our test env.
			// TODO: deletion of the firewall is not yet implemented.
			"replicas":         extensionscontroller.GetReplicas(cluster, 1),
			"clusterID":        string(cluster.Shoot.GetUID()),
			"seedApiURL":       seedApiURL,
			"shootApiURL":      fmt.Sprintf("https://api.%s", *cluster.Shoot.Spec.DNS.Domain),
			"sshKeySecretName": sshSecret.Name,
			"metalapi": map[string]any{
				"url": metalControlPlane.Endpoint,
			},
			"caBundle": strings.TrimSpace(string(caBundle.Data["ca.crt"])),
			"secrets": map[string]any{
				"server": serverSecret.Name,
			},
		},
	}, nil
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

func firewallCompareFunc(a, b *models.V1FirewallResponse) int {
	if b.Allocation == nil || b.Allocation.Created == nil {
		return 1
	}
	if a.Allocation == nil || a.Allocation.Created == nil {
		return -1
	}

	atime := time.Time(*a.Allocation.Created)
	btime := time.Time(*b.Allocation.Created)

	if atime.Before(btime) {
		return -1
	} else if atime.After(btime) {
		return 1
	} else {
		return 0
	}
}

func registryMirrorToValueMap(r apismetal.RegistryMirror) (map[string]any, error) {
	parsedIP, err := netip.ParseAddr(r.IP)
	if err != nil {
		return nil, fmt.Errorf("unable to parse registry ip:%w", err)
	}
	registryIP := parsedIP.String()
	if parsedIP.Is4() {
		registryIP = registryIP + ipv4HostMask
	}
	if parsedIP.Is6() {
		registryIP = registryIP + ipv6HostMask
	}

	return map[string]any{
		"name":     r.Name,
		"endpoint": r.Endpoint,
		"cidr":     registryIP,
		"port":     r.Port,
	}, nil
}

func getDefaultExternalNetwork(nws networkMap, cpConfig *apismetal.ControlPlaneConfig, infrastructureConfig *apismetal.InfrastructureConfig) (string, error) {
	if cpConfig.CloudControllerManager != nil && cpConfig.CloudControllerManager.DefaultExternalNetwork != nil {
		// user has set a specific default external network, check if it's valid

		networkID := *cpConfig.CloudControllerManager.DefaultExternalNetwork

		if !slices.Contains(infrastructureConfig.Firewall.Networks, networkID) {
			return "", fmt.Errorf("given default external network not contained in firewall networks")
		}

		return networkID, nil
	}

	if pointer.SafeDeref(cpConfig.NetworkAccessType) == apismetal.NetworkAccessForbidden {
		// for isolated clusters with forbidden access type it makes no sense to define a default external network because connections will not be allowed automatically anyway
		return "", nil
	}

	var (
		externalNetworks []*models.V1NetworkResponse
		dmzNetworks      []*models.V1NetworkResponse // dmzNetworks are deprecated, this can be removed after all users had enough time to migrate to isolated clusters
	)

	for _, networkID := range infrastructureConfig.Firewall.Networks {
		nw, ok := nws[networkID]
		if !ok {
			return "", fmt.Errorf("network defined in firewall networks does not exist in metal-api")
		}

		_, ok = nw.Labels[tag.NetworkDefaultExternal]
		if !ok {
			continue
		}

		if nw.Parentnetworkid == "" {
			externalNetworks = append(externalNetworks, nw)
			continue
		}

		pn, ok := nws[nw.Parentnetworkid]
		if !ok {
			return "", fmt.Errorf("network defined in firewall networks specified a parent network that does not exist in metal-api")
		}

		if *pn.Privatesuper {
			dmzNetworks = append(dmzNetworks, nw)
			continue
		}
	}

	// if there is an external network we prefer this over DMZ networks
	// from the external network we prefer the one that is the default
	// if there are multiple external networks it's impossible to distinguish which one to choose, so we use the first one defined in the list
	if len(externalNetworks) != 0 {
		for _, nw := range externalNetworks {
			if _, ok := nw.Labels[tag.NetworkDefault]; ok {
				return *nw.ID, nil
			}
		}

		return *externalNetworks[0].ID, nil
	}

	if len(dmzNetworks) != 0 {
		// if there are multiple dmz networks it's impossible to distinguish which one to choose, so we use the first one defined in the list
		return *dmzNetworks[0].ID, nil
	}

	return "", nil
}
