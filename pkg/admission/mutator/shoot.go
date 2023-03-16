package mutator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"

	"github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico"
	c "github.com/gardener/gardener-extension-networking-calico/pkg/calico"
	"github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium"
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
	defaultCloudProfileName    = "metal"
	defaultNetworkType         = c.ReleaseName
	defaultSecretBindingName   = "seed-provider-secret"
	defaultCalicoTyphaEnabled  = false
	defaultCiliumHubbleEnabled = false
)

var (
	defaultCalicoBackend               = calico.None
	defaultMaxPods                     = int32(250)
	defaultNodeCIDRMaskSize            = int32(23)
	defaultAllowedPrivilegedContainers = true
	defaultCalicoKubeProxyEnabled      = true
	defaultCiliumKubeProxyEnabled      = false
	defaultCiliumPSPEnabled            = true
	defaultCiliumTunnel                = cilium.Disabled
	defaultPodsCIDR                    = "10.240.0.0/13"
	defaultServicesCIDR                = "10.248.0.0/18"
)

// NewShootMutator returns a new instance of a shoot mutator.
func NewShootMutator() extensionswebhook.Mutator {
	return &shoot{}
}

type shoot struct {
	client  client.Client
	decoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (s *shoot) InjectScheme(scheme *runtime.Scheme) error {
	s.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// InjectClient injects the given client into the mutator.
func (s *shoot) InjectClient(client client.Client) error {
	s.client = client
	return nil
}

// Mutate mutates the given shoot object.
func (s *shoot) Mutate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*gardenv1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if shoot.Spec.CloudProfileName == "" {
		shoot.Spec.CloudProfileName = defaultCloudProfileName
	}

	if shoot.Spec.SecretBindingName == "" {
		shoot.Spec.SecretBindingName = defaultSecretBindingName
	}

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

	infrastructureConfig, err := s.decodeInfrastructureConfig(shoot.Spec.Provider.InfrastructureConfig)
	if err != nil {
		return err
	}

	infrastructureConfig.Firewall, err = s.getFirewall(ctx, shoot.Spec.CloudProfileName, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	encodedInfrastructureConfig, err := encodeRawExtension(infrastructureConfig)
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
		updatedConfig, err := s.getCalicoConfig(shoot.Spec.Kubernetes.KubeProxy, shoot.Spec.Networking.ProviderConfig)
		if err != nil {
			return err
		}
		shoot.Spec.Networking.ProviderConfig = updatedConfig
	case "cilium":
		updatedConfig, err := s.getCiliumConfig(shoot.Spec.Kubernetes.KubeProxy, shoot.Spec.Networking.ProviderConfig)
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

func (s *shoot) decodeInfrastructureConfig(config *runtime.RawExtension) (*metalv1alpha1.InfrastructureConfig, error) {
	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	if config != nil && config.Raw != nil {
		if _, _, err := s.decoder.Decode(config.Raw, nil, infrastructureConfig); err != nil {
			return nil, err
		}
	}

	return infrastructureConfig, nil
}

func (s *shoot) decodeProviderConfig(providerConfig *runtime.RawExtension) (*metal.CloudProfileConfig, error) {
	cp := &metal.CloudProfileConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ControlPlaneConfig",
		},
	}
	if providerConfig != nil && providerConfig.Raw != nil {
		if _, _, err := s.decoder.Decode(providerConfig.Raw, nil, cp); err != nil {
			return nil, err
		}
	}
	return cp, nil
}

func (s *shoot) decodeCalicoNetworkConfig(providerConfig *runtime.RawExtension) (*calico.NetworkConfig, error) {
	nc := &calico.NetworkConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: calico.SchemeGroupVersion.String(),
			Kind:       "NetworkConfig",
		},
	}
	if providerConfig != nil && providerConfig.Raw != nil {
		if _, _, err := s.decoder.Decode(providerConfig.Raw, nil, nc); err != nil {
			return nil, err
		}
	}
	return nc, nil
}

func (s *shoot) decodeCiliumNetworkConfig(providerConfig *runtime.RawExtension) (*cilium.NetworkConfig, error) {
	nc := &cilium.NetworkConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: cilium.SchemeGroupVersion.String(),
			Kind:       "NetworkConfig",
		},
	}
	if providerConfig != nil && providerConfig.Raw != nil {
		if _, _, err := s.decoder.Decode(providerConfig.Raw, nil, nc); err != nil {
			return nil, err
		}
	}
	return nc, nil
}

func encodeRawExtension(from any) (*runtime.RawExtension, error) {
	encoded, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: encoded,
	}, nil
}

func (s *shoot) getCalicoConfig(kubeProxy *gardenv1beta1.KubeProxyConfig, providerConfig *runtime.RawExtension) (*runtime.RawExtension, error) {
	if kubeProxy.Enabled == nil {
		kubeProxy.Enabled = &defaultCalicoKubeProxyEnabled
	}

	networkConfig, err := s.decodeCalicoNetworkConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	if networkConfig.Backend == nil {
		networkConfig.Backend = &defaultCalicoBackend
	}

	if networkConfig.Typha == nil {
		networkConfig.Typha = &calico.Typha{
			Enabled: defaultCalicoTyphaEnabled,
		}
	}

	return encodeRawExtension(networkConfig)
}

func (s *shoot) getCiliumConfig(kubeProxy *gardenv1beta1.KubeProxyConfig, providerConfig *runtime.RawExtension) (*runtime.RawExtension, error) {
	if kubeProxy.Enabled == nil {
		kubeProxy.Enabled = &defaultCiliumKubeProxyEnabled
	}

	networkConfig, err := s.decodeCiliumNetworkConfig(providerConfig)
	if err != nil {
		return nil, err
	}

	if networkConfig.Hubble == nil {
		networkConfig.Hubble = &cilium.Hubble{
			Enabled: defaultCiliumHubbleEnabled,
		}
	}

	if networkConfig.PSPEnabled == nil {
		networkConfig.PSPEnabled = &defaultCiliumPSPEnabled
	}

	if networkConfig.TunnelMode == nil {
		networkConfig.TunnelMode = &defaultCiliumTunnel
	}

	return encodeRawExtension(networkConfig)
}

func (s *shoot) getFirewall(ctx context.Context, profileName string, partition string) (metalv1alpha1.Firewall, error) {
	f := metalv1alpha1.Firewall{}

	profile := &gardenv1beta1.CloudProfile{}
	if err := s.client.Get(ctx, kutil.Key(profileName), profile); err != nil {
		return f, err
	}

	cloudConfig, err := s.decodeProviderConfig(profile.Spec.ProviderConfig)
	if err != nil {
		return f, err
	}

	controlPlane, p, err := helper.FindMetalControlPlane(cloudConfig, partition)
	if err != nil {
		return f, err
	}

	latestImage := getLatestImage(controlPlane.FirewallImages)
	f.Image = latestImage

	if len(p.FirewallTypes) < 1 {
		return f, nil
	}
	f.Size = p.FirewallTypes[0]

	return f, nil
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
