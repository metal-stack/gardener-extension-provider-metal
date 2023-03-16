package mutator

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"

	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumextensionv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultNetworkType         = "calico"
	defaultSecretBindingName   = "seed-provider-secret" // nolint:gosec
	defaultCalicoTyphaEnabled  = false
	defaultCiliumHubbleEnabled = false
)

var (
	defaultCalicoBackend               = calicoextensionv1alpha1.None
	defaultCalicoPoolMode              = calicoextensionv1alpha1.Never
	defaultMaxPods                     = int32(250)
	defaultNodeCIDRMaskSize            = int32(23)
	defaultAllowedPrivilegedContainers = true
	defaultCalicoKubeProxyEnabled      = true
	defaultCiliumKubeProxyEnabled      = false
	defaultCiliumPSPEnabled            = true
	defaultCiliumTunnel                = ciliumextensionv1alpha1.Disabled
	defaultPodsCIDR                    = "10.240.0.0/13"
	defaultServicesCIDR                = "10.248.0.0/18"
)

// NewShootMutator returns a new instance of a shoot mutator.
func NewShootMutator() extensionswebhook.Mutator {
	return &mutator{}
}

type mutator struct {
	client  client.Client
	decoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (m *mutator) InjectScheme(scheme *runtime.Scheme) error {
	m.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// InjectClient injects the given client into the mutator.
func (s *mutator) InjectClient(client client.Client) error {
	s.client = client
	return nil
}

// Mutate mutates the given shoot object.
func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*gardenv1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	profile := &gardenv1beta1.CloudProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: shoot.Spec.CloudProfileName,
		},
	}
	if err := m.client.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), profile); err != nil {
		return err
	}

	return m.mutate(shoot, *profile)
}

func (m *mutator) mutate(shoot *gardenv1beta1.Shoot, profile gardenv1beta1.CloudProfile) error {
	if shoot.Spec.Kubernetes.AllowPrivilegedContainers == nil {
		shoot.Spec.Kubernetes.AllowPrivilegedContainers = &defaultAllowedPrivilegedContainers
	}

	if shoot.Spec.Kubernetes.KubeControllerManager == nil {
		shoot.Spec.Kubernetes.KubeControllerManager = &gardenv1beta1.KubeControllerManagerConfig{}
	}

	if shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize == nil {
		shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = &defaultNodeCIDRMaskSize
	}

	if shoot.Spec.Kubernetes.Kubelet == nil {
		shoot.Spec.Kubernetes.Kubelet = &gardenv1beta1.KubeletConfig{}
	}

	if shoot.Spec.Kubernetes.Kubelet.MaxPods == nil {
		shoot.Spec.Kubernetes.Kubelet.MaxPods = &defaultMaxPods
	}

	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	err := helper.DecodeRawExtension[*metalv1alpha1.InfrastructureConfig](shoot.Spec.Provider.InfrastructureConfig, infrastructureConfig, m.decoder)
	if err != nil {
		return err
	}

	cloudConfig := &metal.CloudProfileConfig{}
	err = helper.DecodeRawExtension[*metal.CloudProfileConfig](profile.Spec.ProviderConfig, cloudConfig, m.decoder)
	if err != nil {
		return err
	}

	controlPlane, p, err := helper.FindMetalControlPlane(cloudConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	err = m.getFirewall(controlPlane, p, &infrastructureConfig.Firewall)
	if err != nil {
		return err
	}

	encodedInfrastructureConfig, err := helper.EncodeRawExtension(infrastructureConfig)
	if err != nil {
		return err
	}
	shoot.Spec.Provider.InfrastructureConfig = encodedInfrastructureConfig

	if shoot.Spec.Networking.Type == "" {
		shoot.Spec.Networking.Type = defaultNetworkType
	}

	if shoot.Spec.Kubernetes.KubeProxy == nil {
		shoot.Spec.Kubernetes.KubeProxy = &gardenv1beta1.KubeProxyConfig{}
	}

	switch shoot.Spec.Networking.Type {
	case "calico":
		updatedConfig, err := m.getCalicoConfig(shoot.Spec.Kubernetes.KubeProxy, shoot.Spec.Networking.ProviderConfig)
		if err != nil {
			return err
		}
		shoot.Spec.Networking.ProviderConfig = updatedConfig
	case "cilium":
		updatedConfig, err := m.getCiliumConfig(shoot.Spec.Kubernetes.KubeProxy, shoot.Spec.Networking.ProviderConfig)
		if err != nil {
			return err
		}
		shoot.Spec.Networking.ProviderConfig = updatedConfig
	}

	if shoot.Spec.Networking.Pods == nil {
		shoot.Spec.Networking.Pods = &defaultPodsCIDR
	}

	if shoot.Spec.Networking.Services == nil {
		shoot.Spec.Networking.Services = &defaultServicesCIDR
	}

	return nil
}

func (m *mutator) getCalicoConfig(kubeProxy *gardenv1beta1.KubeProxyConfig, providerConfig *runtime.RawExtension) (*runtime.RawExtension, error) {
	if kubeProxy.Enabled == nil {
		kubeProxy.Enabled = &defaultCalicoKubeProxyEnabled
	}

	networkConfig := &calicoextensionv1alpha1.NetworkConfig{}
	err := helper.DecodeRawExtension[*calicoextensionv1alpha1.NetworkConfig](providerConfig, networkConfig, m.decoder)
	if err != nil {
		return nil, err
	}

	if networkConfig.Backend == nil {
		networkConfig.Backend = &defaultCalicoBackend
	}

	if networkConfig.IPv4 == nil {
		networkConfig.IPv4 = &calicoextensionv1alpha1.IPv4{}
	}

	if networkConfig.IPv4.Mode == nil {
		networkConfig.IPv4.Mode = &defaultCalicoPoolMode
	}

	if networkConfig.Typha == nil {
		networkConfig.Typha = &calicoextensionv1alpha1.Typha{
			Enabled: defaultCalicoTyphaEnabled,
		}
	}

	return helper.EncodeRawExtension(networkConfig)
}

func (m *mutator) getCiliumConfig(kubeProxy *gardenv1beta1.KubeProxyConfig, providerConfig *runtime.RawExtension) (*runtime.RawExtension, error) {
	if kubeProxy.Enabled == nil {
		kubeProxy.Enabled = &defaultCiliumKubeProxyEnabled
	}

	networkConfig := &ciliumextensionv1alpha1.NetworkConfig{}
	err := helper.DecodeRawExtension[*ciliumextensionv1alpha1.NetworkConfig](providerConfig, networkConfig, m.decoder)
	if err != nil {
		return nil, err
	}

	if networkConfig.Hubble == nil {
		networkConfig.Hubble = &ciliumextensionv1alpha1.Hubble{
			Enabled: defaultCiliumHubbleEnabled,
		}
	}

	if networkConfig.PSPEnabled == nil {
		networkConfig.PSPEnabled = &defaultCiliumPSPEnabled
	}

	if networkConfig.TunnelMode == nil {
		networkConfig.TunnelMode = &defaultCiliumTunnel
	}

	return helper.EncodeRawExtension(networkConfig)
}

func (m *mutator) getFirewall(controlPlane *metal.MetalControlPlane, partition *metal.Partition, firewall *metalv1alpha1.Firewall) error {
	if firewall.Image == "" {
		firewall.Image = getLatestImage(controlPlane.FirewallImages)
	}

	if firewall.Size == "" && len(partition.FirewallTypes) > 0 {
		firewall.Size = partition.FirewallTypes[0]
	}

	return nil
}

func getLatestImage(images []string) string {
	if len(images) < 1 {
		return ""
	}

	sort.SliceStable(images, func(i, j int) bool {
		osI, vI, err := getOsAndSemverFromImage(images[i])
		if err != nil {
			return false
		}

		osJ, vJ, err := getOsAndSemverFromImage(images[j])
		if err != nil {
			return false
		}

		c := strings.Compare(osI, osJ)
		if c == 0 {
			return vI.GreaterThan(vJ)
		}
		return c <= 0
	})

	return images[0]
}

func getOsAndSemverFromImage(id string) (string, *semver.Version, error) {
	imageParts := strings.Split(id, "-")
	if len(imageParts) < 2 {
		return "", nil, errors.New("image does not contain a version")
	}

	parts := len(imageParts) - 1
	os := strings.Join(imageParts[:parts], "-")
	version := strings.Join(imageParts[parts:], "")
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", nil, err
	}
	return os, v, nil
}
