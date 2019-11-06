// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/gardener/gardener/pkg/apis/garden/v1beta1/helper"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"path/filepath"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane/genericactuator"
	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	metalclient "github.com/metal-pod/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-pod/metal-go"

	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
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
	groupRolebindingControllerName       = "group-rolebinding-controller"
	limitValidatingWebhookDeploymentName = "limit-validating-webhook"
	limitValidatingWebhookServerName     = "limit-validating-webhook-server"
	accountingExporterName               = "accounting-exporter"
	authNWebhookDeploymentName           = "kube-jwt-authn-webhook"
	authNWebhookServerName               = "kube-jwt-authn-webhook-server"
	droptailerNamespace                  = "firewall"
	droptailerClientSecretName           = "droptailer-client"
	droptailerServerSecretName           = "droptailer-server"
)

var controlPlaneSecrets = &secrets.Secrets{
	CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
		gardencorev1alpha1.SecretNameCACluster: {
			Name:       gardencorev1alpha1.SecretNameCACluster,
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
					SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: gardencorev1alpha1.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:         groupRolebindingControllerName,
					CommonName:   "system:group-rolebinding-controller",
					Organization: []string{user.SystemPrivilegedGroup},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: gardencorev1alpha1.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       authNWebhookServerName,
					CommonName: authNWebhookDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(authNWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[gardencorev1alpha1.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       limitValidatingWebhookServerName,
					CommonName: limitValidatingWebhookDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(limitValidatingWebhookDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[gardencorev1alpha1.SecretNameCACluster],
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       accountingExporterName,
					CommonName: "system:accounting-exporter",
					// Groupname of user
					Organization: []string{accountingExporterName},
					CertType:     secrets.ClientCert,
					SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
				},
				KubeConfigRequest: &secrets.KubeConfigRequest{
					ClusterName:  clusterName,
					APIServerURL: gardencorev1alpha1.DeploymentNameKubeAPIServer,
				},
			},
			&secrets.ControlPlaneSecretConfig{
				CertificateSecretConfig: &secrets.CertificateSecretConfig{
					Name:       cloudControllerManagerServerName,
					CommonName: cloudControllerManagerDeploymentName,
					DNSNames:   controlplane.DNSNamesForService(cloudControllerManagerDeploymentName, clusterName),
					CertType:   secrets.ServerCert,
					SigningCA:  cas[gardencorev1alpha1.SecretNameCACluster],
				},
			},
		}
	},
}

var configChart = &chart.Chart{
	Name:   "config",
	Path:   filepath.Join(metal.InternalChartsPath, "cloud-provider-config"),
	Images: []string{},
	Objects: []*chart.Object{
		// this config is mounted by the shoot-kube-apiserver at startup and should therefore be deployed before the controlplane
		{Type: &corev1.ConfigMap{}, Name: "authn-webhook-config"},
	},
}

var controlPlaneChart = &chart.Chart{
	Name:   "control-plane",
	Path:   filepath.Join(metal.InternalChartsPath, "control-plane"),
	Images: []string{metal.CCMImageName, metal.AuthNWebhookImageName, metal.AccountingExporterImageName, metal.GroupRolebindingControllerImageName, metal.DroptailerImageName, metal.LimitValidatingWebhookImageName},
	Objects: []*chart.Object{
		{Type: &corev1.Service{}, Name: "cloud-controller-manager"},
		{Type: &appsv1.Deployment{}, Name: "cloud-controller-manager"},

		{Type: &appsv1.Deployment{}, Name: "kube-jwt-authn-webhook"},
		{Type: &corev1.Service{}, Name: "kube-jwt-authn-webhook"},

		{Type: &appsv1.Deployment{}, Name: "limit-validating-webhook"},
		{Type: &corev1.Service{}, Name: "limit-validating-webhook"},

		{Type: &corev1.ServiceAccount{}, Name: "group-rolebinding-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "group-rolebinding-controller"},
		{Type: &appsv1.Deployment{}, Name: "group-rolebinding-controller"},

		{Type: &appsv1.Deployment{}, Name: "accounting-exporter"},
	},
}

var cpShootChart = &chart.Chart{
	Name: "shoot-control-plane",
	Path: filepath.Join(metal.InternalChartsPath, "shoot-control-plane"),
	Objects: []*chart.Object{
		{Type: &rbacv1.ClusterRole{}, Name: "system:controller:cloud-node-controller"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:controller:cloud-node-controller"},

		{Type: &admissionv1beta1.ValidatingWebhookConfiguration{}, Name: "limit-validating-webhook"},

		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:group-rolebinding-controller"},

		{Type: &rbacv1.ClusterRole{}, Name: "system:accounting-exporter"},
		{Type: &rbacv1.ClusterRoleBinding{}, Name: "system:accounting-exporter"},

		{Type: &appsv1.Deployment{}, Name: "droptailer"},
	},
}

var storageClassChart = &chart.Chart{
	Name: "shoot-storageclasses",
	Path: filepath.Join(metal.InternalChartsPath, "shoot-storageclasses"),
}

// NewValuesProvider creates a new ValuesProvider for the generic actuator.
func NewValuesProvider(mgr manager.Manager, logger logr.Logger, config AccountingConfig) genericactuator.ValuesProvider {
	return &valuesProvider{
		mgr:              mgr,
		logger:           logger.WithName("metal-values-provider"),
		accountingConfig: config,
	}
}

// valuesProvider is a ValuesProvider that provides AWS-specific values for the 2 charts applied by the generic actuator.
type valuesProvider struct {
	decoder          runtime.Decoder
	restConfig       *rest.Config
	client           client.Client
	logger           logr.Logger
	accountingConfig AccountingConfig
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

	values, err := vp.getAuthNConfigValues(ctx, cp, cluster)

	return values, err
}

func (vp *valuesProvider) getAuthNConfigValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {

	namespace := cluster.Shoot.Status.TechnicalID

	// this should work as the kube-apiserver is a pod in the same cluster as the kube-jwt-authn-webhook
	// example https://kube-jwt-authn-webhook.shoot--local--myshootname.svc.cluster.local/authenticate
	url := fmt.Sprintf("https://%s.%s.svc.cluster.local/authenticate", authNWebhookDeploymentName, namespace)

	values := map[string]interface{}{
		"authnWebhook_url": url,
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
	// Decode providerConfig
	cpConfig := &apismetal.ControlPlaneConfig{}
	if _, _, err := vp.decoder.Decode(cp.Spec.ProviderConfig.Raw, nil, cpConfig); err != nil {
		return nil, errors.Wrapf(err, "could not decode providerConfig of controlplane '%s'", util.ObjectName(cp))
	}

	mclient, err := metalclient.NewClient(ctx, vp.client, &cp.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	// Get CCM chart values
	chartValues, err := getCCMChartValues(cpConfig, cp, cluster, checksums, scaledDown, mclient)
	if err != nil {
		return nil, err
	}

	authValues, err := getAuthNGroupRoleChartValues(cp, cluster)
	if err != nil {
		return nil, err
	}

	accValues, err := getAccountingExporterChartValues(vp.accountingConfig, cluster, mclient)
	if err != nil {
		return nil, err
	}

	lvwValues, err := getLimitValidationWebhookControlPlaneChartValues(cluster)
	if err != nil {
		return nil, err
	}

	merge(chartValues, authValues, accValues, lvwValues)

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
func (vp *valuesProvider) GetControlPlaneShootChartValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {

	vp.logger.Info("GetControlPlaneShootChartValues")

	values, err := vp.getControlPlaneShootLimitValidationWebhookChartValues(ctx, cp, cluster)
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

// GetLimitValidationWebhookChartValues returns the values for the LimitValidationWebhook.
func (vp *valuesProvider) getControlPlaneShootLimitValidationWebhookChartValues(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {
	secretName := limitValidatingWebhookServerName
	namespace := cluster.Shoot.Status.TechnicalID

	secret, err := vp.getSecret(ctx, namespace, secretName)
	if err != nil {
		return nil, err
	}

	// CA-Cert for TLS
	caBundle := base64.StdEncoding.EncodeToString(secret.Data[secrets.DataKeyCertificateCA])

	// this should work as the kube-apiserver is a pod in the same cluster as the limit-validating-webhook
	// example https://limit-validating-webhook.shoot--local--myshootname.svc.cluster.local/validate
	url := fmt.Sprintf("https://%s.%s.svc.cluster.local/validate", limitValidatingWebhookDeploymentName, namespace)

	values := map[string]interface{}{
		"limitValidatingWebhook_url":      url,
		"limitValidatingWebhook_caBundle": caBundle,
	}

	return values, nil
}

func (vp *valuesProvider) deployControlPlaneShootDroptailerCerts(ctx context.Context, cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) error {
	// TODO: There is actually no nice way to deploy the certs into the shoot when we want to use
	// the certificate helper functions from Gardener itself...
	// Maybe we can find a better solution? This is actually only for chart values...

	wanted := &secrets.Secrets{
		CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
			gardencorev1alpha1.SecretNameCACluster: {
				Name:       gardencorev1alpha1.SecretNameCACluster,
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
						SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
					},
				},
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         droptailerServerSecretName,
						CommonName:   "droptailer",
						Organization: []string{"droptailer-server"},
						CertType:     secrets.ServerCert,
						SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
					},
				},
			}
		},
	}

	shootConfig, _, err := util.NewClientForShoot(ctx, vp.client, cluster.Shoot.Status.TechnicalID, client.Options{})
	if err != nil {
		return errors.Wrapf(err, "could not create shoot client")
	}
	cs, err := kubernetes.NewForConfig(shootConfig)
	if err != nil {
		return errors.Wrap(err, "could not create shoot kubernetes client")
	}
	gcs, err := gardenerkubernetes.NewForConfig(shootConfig, client.Options{})
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
			return errors.Wrap(err, "could not search for existance of droptailer namespace")
		}
	}

	_, err = wanted.Deploy(ctx, cs, gcs, droptailerNamespace)
	if err != nil {
		return fmt.Errorf("could not deploy droptailer secrets to shoot cluster; err: %w", err)
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
	cp *extensionsv1alpha1.ControlPlane,
	cluster *extensionscontroller.Cluster,
	checksums map[string]string,
	scaledDown bool,
	mclient *metalgo.Driver,
) (map[string]interface{}, error) {
	projectID := cluster.Shoot.Spec.Cloud.Metal.ProjectID
	nodeCIDR := cluster.Shoot.Spec.Cloud.Metal.Networks.Nodes

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, nodeCIDR)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"replicas":          extensionscontroller.GetControlPlaneReplicas(cluster.Shoot, scaledDown, 1),
		"projectID":         projectID,
		"clusterID":         cluster.Shoot.ObjectMeta.UID,
		"partitionID":       cluster.Shoot.Spec.Cloud.Metal.Zones[0],
		"networkID":         *privateNetwork.ID,
		"kubernetesVersion": cluster.Shoot.Spec.Kubernetes.Version,
		"podNetwork":        extensionscontroller.GetPodNetwork(cluster.Shoot),
		"podAnnotations": map[string]interface{}{
			"checksum/secret-cloud-controller-manager":        checksums[cloudControllerManagerDeploymentName],
			"checksum/secret-cloud-controller-manager-server": checksums[cloudControllerManagerServerName],
			// TODO Use constant from github.com/gardener/gardener/pkg/apis/core/v1alpha1 when available
			// See https://github.com/gardener/gardener/pull/930
			"checksum/secret-cloudprovider":            checksums[gardencorev1alpha1.SecretNameCloudProvider],
			"checksum/configmap-cloud-provider-config": checksums[metal.CloudProviderConfigName],
		},
	}

	if cpConfig.CloudControllerManager != nil {
		values["featureGates"] = cpConfig.CloudControllerManager.FeatureGates
	}

	return values, nil
}

// returns values for "authn-webhook" and "group-rolebinding-controller" that are thematically related
func getAuthNGroupRoleChartValues(cp *extensionsv1alpha1.ControlPlane, cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {

	annotations := cluster.Shoot.GetAnnotations()
	clusterName := annotations[metal.ShootAnnotationClusterName]
	tenant := annotations[metal.ShootAnnotationTenant]

	var tokenIssuerPC *gardencorev1alpha1.ProviderConfig
	for _, ext := range cluster.Shoot.Spec.Extensions {

		if ext.Type == metal.ShootExtensionTypeTokenIssuer {
			tokenIssuerPC = ext.ProviderConfig
			break
		}
	}

	if tokenIssuerPC == nil {
		return nil, errors.New("tokenissuer-Extension not found")
	}

	ti := &TokenIssuer{}
	err := json.Unmarshal(tokenIssuerPC.Raw, ti)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"authn_tenant":             tenant,
		"authn_clustername":        clusterName,
		"authn_oidcIssuerUrl":      ti.IssuerUrl,
		"authn_oidcIssuerClientId": ti.ClientId,
		"authn_debug":              "true",

		"grprb_clustername": clusterName,
	}

	return values, nil
}

func getAccountingExporterChartValues(accountingConfig AccountingConfig, cluster *extensionscontroller.Cluster, mclient *metalgo.Driver) (map[string]interface{}, error) {
	annotations := cluster.Shoot.GetAnnotations()
	partitionID := cluster.Shoot.Spec.Cloud.Metal.Zones[0]
	projectID := cluster.Shoot.Spec.Cloud.Metal.ProjectID
	clusterID := cluster.Shoot.ObjectMeta.UID
	clusterName := annotations[metal.ShootAnnotationClusterName]
	tenant := annotations[metal.ShootAnnotationTenant]

	project, err := metalclient.GetProjectByID(mclient, projectID)
	if err != nil {
		return nil, err
	}

	values := map[string]interface{}{
		"accex_partitionID": partitionID,
		"accex_tenant":      tenant,
		"accex_projectname": project.Name,
		"accex_projectID":   projectID,

		"accex_clustername": clusterName,
		"accex_clusterID":   clusterID,

		"accex_accountingsink_url":  accountingConfig.AccountingSinkUrl,
		"accex_accountingsink_HMAC": accountingConfig.AccountingSinkHmac,
	}

	return values, nil
}

func getLimitValidationWebhookControlPlaneChartValues(cluster *extensionscontroller.Cluster) (map[string]interface{}, error) {

	shootedSeed, err := helper.ReadShootedSeed(cluster.Shoot)
	isNormalShoot := shootedSeed == nil || err != nil

	values := map[string]interface{}{
		"lvw_validate": isNormalShoot,
	}

	return values, nil
}

// Data for configuration of AuthNWebhook
type TokenIssuer struct {
	IssuerUrl string `json:"issuerUrl" optional:"false"`
	ClientId  string `json:"clientId" optional:"false"`
}

// Data for configuration of IDM-API WebHook (deployment to be done!)
type UserDirectory struct {
	IdmApi           string `json:"idmApi" optional:"false"`
	IdmApiUser       string `json:"idmApiUser" optional:"false"`
	IdmApiPassword   string `json:"idmApiPassword" optional:"false"`
	TargetSystemId   string `json:"targetSystemId" optional:"false"`
	TargetSystemType string `json:"targetSystemType" optional:"false"`
	AccessCode       string `json:"accessCode" optional:"false"`
	CustomerId       string `json:"cstomerId" optional:"false"`
}
