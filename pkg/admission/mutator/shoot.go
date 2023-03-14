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

// Mutate mutates the given shoot object.
func (s *shoot) Mutate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*gardenv1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if shoot.Spec.CloudProfileName == "" {
		shoot.Spec.CloudProfileName = "metal"
	}

	if shoot.Spec.SecretBindingName == "" {
		shoot.Spec.SecretBindingName = "seed-provider-secret"
	}

	t := true
	if shoot.Spec.Kubernetes.AllowPrivilegedContainers == nil {
		shoot.Spec.Kubernetes.AllowPrivilegedContainers = &t
	}

	nodeCIDRMaskSize := int32(23)
	if shoot.Spec.Kubernetes.KubeControllerManager == nil {
		shoot.Spec.Kubernetes.KubeControllerManager = &gardenv1beta1.KubeControllerManagerConfig{
			NodeCIDRMaskSize: &nodeCIDRMaskSize,
		}
	} else if shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize == nil {
		shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = &nodeCIDRMaskSize
	}

	maxPods := int32(250)
	if shoot.Spec.Kubernetes.Kubelet == nil {
		shoot.Spec.Kubernetes.Kubelet = &gardenv1beta1.KubeletConfig{
			MaxPods: &maxPods,
		}
	} else if shoot.Spec.Kubernetes.Kubelet.MaxPods == nil {
		shoot.Spec.Kubernetes.Kubelet.MaxPods = &maxPods
	}

	infrastructureConfig, err := s.decodeInfrastructureConfig(shoot.Spec.Provider.InfrastructureConfig)
	if err != nil {
		return err
	}

	infrastructureConfig.Firewall, err = s.getFirewall(ctx, shoot.Spec.CloudProfileName, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	shoot.Spec.Provider.InfrastructureConfig = &runtime.RawExtension{
		Object: infrastructureConfig,
	}

	if shoot.Spec.Networking.Type == "" {
		shoot.Spec.Networking.Type = "calico"
	}

	if shoot.Spec.Networking.Type == "calico" {
		shoot.Spec.Kubernetes.KubeProxy = &gardenv1beta1.KubeProxyConfig{
			Enabled: &t,
		}
	}

	if shoot.Spec.Networking.Type == "cilium" {
		f := false
		shoot.Spec.Kubernetes.KubeProxy = &gardenv1beta1.KubeProxyConfig{
			Enabled: &f,
		}
	}

	networkingConfig, err := getNetworkingConfig(shoot.Spec.Networking.Type)
	if err != nil {
		return err
	}

	shoot.Spec.Networking = gardenv1beta1.Networking{
		ProviderConfig: networkingConfig,
	}

	if shoot.Spec.Networking.Pods == nil {
		pods := "10.240.0.0/13"
		shoot.Spec.Networking.Pods = &pods
	}

	if shoot.Spec.Networking.Services == nil {
		services := "10.248.0.0/18"
		shoot.Spec.Networking.Services = &services
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

func encodeProviderConfig(from any) (*runtime.RawExtension, error) {
	encoded, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: encoded,
	}, nil
}

func getNetworkingConfig(t string) (*runtime.RawExtension, error) {
	var networkingConfig *runtime.RawExtension
	var err error

	if t == "calico" {
		backend := calico.None
		networkingConfig, err = encodeProviderConfig(calico.NetworkConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: calico.SchemeGroupVersion.String(),
				Kind:       "NetworkConfig",
			},
			Backend: &backend,
			Typha: &calico.Typha{
				Enabled: false,
			},
		})

		if err != nil {
			return nil, err
		}

		return networkingConfig, nil
	}

	if t == "cilium" {
		t := true
		d := cilium.Disabled
		config := cilium.NetworkConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: cilium.SchemeGroupVersion.String(),
				Kind:       "NetworkConfig",
			},
			Hubble: &cilium.Hubble{
				Enabled: true,
			},
			PSPEnabled: &t,
			TunnelMode: &d,
		}

		networkingConfig, err = encodeProviderConfig(config)
		if err != nil {
			return nil, err
		}

		return networkingConfig, nil
	}

	return networkingConfig, nil
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

	controlPlane, _, err := helper.FindMetalControlPlane(cloudConfig, partition)
	if err != nil {
		return f, err
	}

	latestImage := getLatestImage(controlPlane.FirewallImages)

	f.Image = latestImage
	f.Size = controlPlane.Partitions[partition].FirewallTypes[0]

	return f, nil
}

func getLatestImage(images []string) string {
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
