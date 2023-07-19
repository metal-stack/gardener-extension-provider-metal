package mutator

import (
	"errors"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumextensionv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"k8s.io/apimachinery/pkg/runtime"
)

type defaulter struct {
	c       *config
	decoder runtime.Decoder

	controlPlane *metal.MetalControlPlane
	partition    *metal.Partition
}

func (d *defaulter) defaultShoot(shoot *gardenv1beta1.Shoot) error {
	k8version, err := semver.NewVersion(shoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	if shoot.Spec.Kubernetes.KubeControllerManager == nil && k8version.LessThan(semver.MustParse("1.25")) {
		shoot.Spec.Kubernetes.KubeControllerManager = &gardenv1beta1.KubeControllerManagerConfig{}
	}

	if shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize == nil {
		shoot.Spec.Kubernetes.KubeControllerManager.NodeCIDRMaskSize = pointer.Pointer(d.c.nodeCIDRMaskSize())
	}

	if shoot.Spec.Kubernetes.Kubelet == nil {
		shoot.Spec.Kubernetes.Kubelet = &gardenv1beta1.KubeletConfig{}
	}

	if shoot.Spec.Kubernetes.Kubelet.MaxPods == nil {
		shoot.Spec.Kubernetes.Kubelet.MaxPods = pointer.Pointer(d.c.maxPods())
	}

	err = d.defaultInfrastructureConfig(shoot)
	if err != nil {
		return err
	}

	err = d.defaultNetworking(shoot)
	if err != nil {
		return err
	}

	return nil
}

func (d *defaulter) defaultInfrastructureConfig(shoot *gardenv1beta1.Shoot) error {
	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	err := helper.DecodeRawExtension(shoot.Spec.Provider.InfrastructureConfig, infrastructureConfig, d.decoder)
	if err != nil {
		return err
	}

	if infrastructureConfig.Firewall.Image == "" {
		infrastructureConfig.Firewall.Image = getLatestImage(d.controlPlane.FirewallImages)
	}

	if infrastructureConfig.Firewall.Size == "" && len(d.partition.FirewallTypes) > 0 {
		infrastructureConfig.Firewall.Size = d.partition.FirewallTypes[0]
	}

	shoot.Spec.Provider.InfrastructureConfig = &runtime.RawExtension{
		Object: infrastructureConfig,
	}

	return nil
}

func (d *defaulter) defaultNetworking(shoot *gardenv1beta1.Shoot) error {
	if shoot.Spec.Networking.Type == "" {
		shoot.Spec.Networking.Type = d.c.networkType()
	}

	if shoot.Spec.Networking.Pods == nil {
		shoot.Spec.Networking.Pods = pointer.Pointer(d.c.podsCIDR())
	}

	if shoot.Spec.Networking.Services == nil {
		shoot.Spec.Networking.Services = pointer.Pointer(d.c.servicesCIDR())
	}

	// we only default networking config if there is no provider config given
	if shoot.Spec.Networking.ProviderConfig != nil || pointer.SafeDeref(shoot.Spec.Networking.ProviderConfig).Raw != nil {
		return nil
	}

	switch shoot.Spec.Networking.Type {
	case "calico":
		err := d.defaultCalicoConfig(shoot)
		if err != nil {
			return err
		}
	case "cilium":
		err := d.defaultCiliumConfig(shoot)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *defaulter) defaultCalicoConfig(shoot *gardenv1beta1.Shoot) error {
	networkConfig := &calicoextensionv1alpha1.NetworkConfig{}
	err := helper.DecodeRawExtension(shoot.Spec.Networking.ProviderConfig, networkConfig, d.decoder)
	if err != nil {
		return err
	}

	if shoot.Spec.Kubernetes.KubeProxy == nil {
		shoot.Spec.Kubernetes.KubeProxy = &gardenv1beta1.KubeProxyConfig{}
	}

	if shoot.Spec.Kubernetes.KubeProxy.Enabled == nil {
		shoot.Spec.Kubernetes.KubeProxy.Enabled = pointer.Pointer(d.c.calicoKubeProxyEnabled())
	}

	if networkConfig.Backend == nil {
		networkConfig.Backend = pointer.Pointer(d.c.calicoBackend())
	}

	if networkConfig.IPv4 == nil {
		networkConfig.IPv4 = &calicoextensionv1alpha1.IPv4{}
	}

	if networkConfig.IPv4.Mode == nil {
		networkConfig.IPv4.Mode = pointer.Pointer(d.c.calicoPoolMode())
	}

	if networkConfig.Typha == nil {
		networkConfig.Typha = &calicoextensionv1alpha1.Typha{
			Enabled: d.c.calicoTyphaEnabled(),
		}
	}

	shoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
		Object: networkConfig,
	}

	return nil
}

func (d *defaulter) defaultCiliumConfig(shoot *gardenv1beta1.Shoot) error {
	networkConfig := &ciliumextensionv1alpha1.NetworkConfig{}
	err := helper.DecodeRawExtension(shoot.Spec.Networking.ProviderConfig, networkConfig, d.decoder)
	if err != nil {
		return err
	}

	if shoot.Spec.Kubernetes.KubeProxy == nil {
		shoot.Spec.Kubernetes.KubeProxy = &gardenv1beta1.KubeProxyConfig{}
	}

	if shoot.Spec.Kubernetes.KubeProxy.Enabled == nil {
		shoot.Spec.Kubernetes.KubeProxy.Enabled = pointer.Pointer(d.c.ciliumKubeProxyEnabled())
	}

	if networkConfig.Hubble == nil {
		networkConfig.Hubble = &ciliumextensionv1alpha1.Hubble{
			Enabled: d.c.ciliumHubbleEnabled(),
		}
	}

	if networkConfig.PSPEnabled == nil {
		networkConfig.PSPEnabled = pointer.Pointer(d.c.ciliumPSPEnabled())
	}

	if networkConfig.TunnelMode == nil {
		networkConfig.TunnelMode = pointer.Pointer(d.c.ciliumTunnel())
	}

	if networkConfig.Devices == nil {
		networkConfig.Devices = d.c.ciliumDevices()
	}

	if networkConfig.IPv4NativeRoutingCIDREnabled == nil {
		networkConfig.IPv4NativeRoutingCIDREnabled = pointer.Pointer(d.c.ciliumIPv4NativeRoutingCIDREnabled())
	}

	if networkConfig.LoadBalancingMode == nil {
		networkConfig.LoadBalancingMode = pointer.Pointer(d.c.ciliumLoadBalancingMode())
	}

	if networkConfig.MTU == nil {
		networkConfig.MTU = pointer.Pointer(d.c.ciliumMTU())
	}

	shoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
		Object: networkConfig,
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
