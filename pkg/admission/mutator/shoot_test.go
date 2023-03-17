package mutator

import (
	"testing"

	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumextensionv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

type spec struct {
	kubernetes           gardenv1beta1.Kubernetes
	infrastructureConfig metalv1alpha1.InfrastructureConfig
	networkingType       string
	networkingPods       string
	networkingServices   string
}

type want struct {
	spec
	calicoConfig *calicoextensionv1alpha1.NetworkConfig
	ciliumConfig *ciliumextensionv1alpha1.NetworkConfig
}

func Test_mutator_mutate(t *testing.T) {
	tests := []struct {
		name         string
		spec         spec
		calicoConfig *calicoextensionv1alpha1.NetworkConfig
		ciliumConfig *ciliumextensionv1alpha1.NetworkConfig
		wantErr      bool
		want         want
	}{
		{
			name:    "empty spec",
			spec:    spec{},
			wantErr: false,
			want: want{
				spec: spec{
					kubernetes: gardenv1beta1.Kubernetes{
						AllowPrivilegedContainers: &defaultAllowedPrivilegedContainers,
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: &defaultNodeCIDRMaskSize,
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: &defaultMaxPods,
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: &defaultCalicoKubeProxyEnabled,
						},
					},
					networkingType:     defaultNetworkType,
					networkingPods:     defaultPodsCIDR,
					networkingServices: defaultServicesCIDR,
					infrastructureConfig: metalv1alpha1.InfrastructureConfig{
						Firewall: metalv1alpha1.Firewall{
							Image: "firewall-2.0.20210207",
							Size:  "n1-medium-x86",
						},
					},
				},
				calicoConfig: &calicoextensionv1alpha1.NetworkConfig{
					Backend: &defaultCalicoBackend,
					IPv4: &calicoextensionv1alpha1.IPv4{
						Mode: &defaultCalicoPoolMode,
					},
					Typha: &calicoextensionv1alpha1.Typha{
						Enabled: defaultCalicoTyphaEnabled,
					},
				},
			},
		},
		{
			name: "empty spec with cilium",
			spec: spec{
				networkingType: "cilium",
			},
			wantErr: false,
			want: want{
				spec: spec{
					kubernetes: gardenv1beta1.Kubernetes{
						AllowPrivilegedContainers: &defaultAllowedPrivilegedContainers,
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: &defaultNodeCIDRMaskSize,
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: &defaultMaxPods,
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: &defaultCiliumKubeProxyEnabled,
						},
					},
					networkingType:     "cilium",
					networkingPods:     defaultPodsCIDR,
					networkingServices: defaultServicesCIDR,
					infrastructureConfig: metalv1alpha1.InfrastructureConfig{
						Firewall: metalv1alpha1.Firewall{
							Image: "firewall-2.0.20210207",
							Size:  "n1-medium-x86",
						},
					},
				},
				ciliumConfig: &ciliumextensionv1alpha1.NetworkConfig{
					Hubble: &ciliumextensionv1alpha1.Hubble{
						Enabled: defaultCiliumHubbleEnabled,
					},
					PSPEnabled: &defaultCiliumPSPEnabled,
					TunnelMode: &defaultCiliumTunnel,
				},
			},
		},
		{
			name: "calico; no defaults needed",
			spec: spec{
				kubernetes: gardenv1beta1.Kubernetes{
					AllowPrivilegedContainers: pointer.Bool(false),
					KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
						NodeCIDRMaskSize: pointer.Int32(22),
					},
					Kubelet: &gardenv1beta1.KubeletConfig{
						MaxPods: pointer.Int32(200),
					},
					KubeProxy: &gardenv1beta1.KubeProxyConfig{
						Enabled: pointer.Bool(false),
					},
				},
				networkingType:     defaultNetworkType,
				networkingPods:     "10.240.1.0/13",
				networkingServices: "10.248.1.0/18",
				infrastructureConfig: metalv1alpha1.InfrastructureConfig{
					Firewall: metalv1alpha1.Firewall{
						Image: "firewall-ubuntu-2.0.20201214",
						Size:  "n1-medium-x86",
					},
				},
			},
			calicoConfig: &calicoextensionv1alpha1.NetworkConfig{
				Backend: &defaultCalicoBackend,
				IPv4: &calicoextensionv1alpha1.IPv4{
					Mode: &defaultCalicoPoolMode,
				},
				Typha: &calicoextensionv1alpha1.Typha{
					Enabled: *pointer.Bool(true),
				},
			},
			wantErr: false,
			want: want{
				spec: spec{
					kubernetes: gardenv1beta1.Kubernetes{
						AllowPrivilegedContainers: pointer.Bool(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Int32(22),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Int32(200),
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: pointer.Bool(false),
						},
					},
					networkingType:     defaultNetworkType,
					networkingPods:     "10.240.1.0/13",
					networkingServices: "10.248.1.0/18",
					infrastructureConfig: metalv1alpha1.InfrastructureConfig{
						Firewall: metalv1alpha1.Firewall{
							Image: "firewall-ubuntu-2.0.20201214",
							Size:  "n1-medium-x86",
						},
					},
				},
				calicoConfig: &calicoextensionv1alpha1.NetworkConfig{
					Backend: &defaultCalicoBackend,
					IPv4: &calicoextensionv1alpha1.IPv4{
						Mode: &defaultCalicoPoolMode,
					},
					Typha: &calicoextensionv1alpha1.Typha{
						Enabled: *pointer.Bool(true),
					},
				},
			},
		},
		{
			name: "cilium; no defaults needed",
			spec: spec{
				kubernetes: gardenv1beta1.Kubernetes{
					AllowPrivilegedContainers: pointer.Bool(false),
					KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
						NodeCIDRMaskSize: pointer.Int32(22),
					},
					Kubelet: &gardenv1beta1.KubeletConfig{
						MaxPods: pointer.Int32(200),
					},
					KubeProxy: &gardenv1beta1.KubeProxyConfig{
						Enabled: pointer.Bool(true),
					},
				},
				networkingType:     "cilium",
				networkingPods:     "10.240.1.0/13",
				networkingServices: "10.248.1.0/18",
				infrastructureConfig: metalv1alpha1.InfrastructureConfig{
					Firewall: metalv1alpha1.Firewall{
						Image: "firewall-ubuntu-2.0.20201214",
						Size:  "n1-medium-x86",
					},
				},
			},
			ciliumConfig: &ciliumextensionv1alpha1.NetworkConfig{
				Hubble: &ciliumextensionv1alpha1.Hubble{
					Enabled: true,
				},
				PSPEnabled: pointer.Bool(false),
				TunnelMode: &defaultCiliumTunnel,
			},
			wantErr: false,
			want: want{
				spec: spec{
					kubernetes: gardenv1beta1.Kubernetes{
						AllowPrivilegedContainers: pointer.Bool(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Int32(22),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Int32(200),
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: pointer.Bool(true),
						},
					},
					networkingType:     "cilium",
					networkingPods:     "10.240.1.0/13",
					networkingServices: "10.248.1.0/18",
					infrastructureConfig: metalv1alpha1.InfrastructureConfig{
						Firewall: metalv1alpha1.Firewall{
							Image: "firewall-ubuntu-2.0.20201214",
							Size:  "n1-medium-x86",
						},
					},
				},
				ciliumConfig: &ciliumextensionv1alpha1.NetworkConfig{
					Hubble: &ciliumextensionv1alpha1.Hubble{
						Enabled: true,
					},
					PSPEnabled: pointer.Bool(false),
					TunnelMode: &defaultCiliumTunnel,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mutator{}
			err := m.InjectScheme(runtime.NewScheme())
			if err != nil {
				t.Errorf("could not inject scheme, err = %v", err)
			}

			profile, err := getCloudProfile()
			if err != nil {
				t.Errorf("could not get cloud profile, err = %v", err)
			}

			shoot, err := getShoot(tt.spec, tt.calicoConfig, tt.ciliumConfig)
			if err != nil {
				t.Errorf("could not create shoot, err = %v", err)
			}

			if err := m.mutate(shoot, *profile); (err != nil) != tt.wantErr {
				t.Errorf("mutator.mutate() error = %v, wantErr %v", err, tt.wantErr)
			}

			wantShoot, err := getShoot(tt.want.spec, tt.want.calicoConfig, tt.want.ciliumConfig)
			if err != nil {
				t.Errorf("could not get updated shoot, err = %v", err)
			}

			if diff := cmp.Diff(shoot, wantShoot); diff != "" {
				t.Errorf("%v", diff)
			}
		})
	}
}

func getShoot(spec spec, calicoConfig *calicoextensionv1alpha1.NetworkConfig, ciliumConfig *ciliumextensionv1alpha1.NetworkConfig) (*gardenv1beta1.Shoot, error) {
	shoot := &gardenv1beta1.Shoot{}
	spec.infrastructureConfig.PartitionID = "muc"
	shoot.Spec.Kubernetes = spec.kubernetes

	if spec.networkingType != "" {
		shoot.Spec.Networking.Type = spec.networkingType
	}

	if spec.networkingPods != "" {
		shoot.Spec.Networking.Pods = &spec.networkingPods
	}

	if spec.networkingServices != "" {
		shoot.Spec.Networking.Services = &spec.networkingServices
	}

	infrastructureConfig, err := helper.EncodeRawExtension(&spec.infrastructureConfig)
	if err != nil {
		return shoot, err
	}
	shoot.Spec.Provider.InfrastructureConfig = infrastructureConfig

	if calicoConfig != nil {
		config, err := helper.EncodeRawExtension(calicoConfig)
		if err != nil {
			return shoot, err
		}
		shoot.Spec.Networking.ProviderConfig = config
		return shoot, nil
	}

	if ciliumConfig != nil {
		config, err := helper.EncodeRawExtension(ciliumConfig)
		if err != nil {
			return shoot, err
		}
		shoot.Spec.Networking.ProviderConfig = config
	}

	return shoot, nil
}

func getCloudProfile() (*gardenv1beta1.CloudProfile, error) {
	cloudProfileConfig := metal.CloudProfileConfig{
		MetalControlPlanes: map[string]metal.MetalControlPlane{
			"metal": {
				FirewallImages: []string{
					"firewall-2.0.2020121",
					"firewall-2.0.20210207",
					"firewall-2.0dasdf",
					"firewall-ubuntu-2.0.19700101",
					"firewall-ubuntu-2.0.20201126",
					"firewall-ubuntu-2.0.20201023",
					"firewall-ubuntu-2.0.20201214",
					"firewall-ubuntu-2.0.20210304",
					"firewall-ubuntu-2.0.20210131",
					"firewall-ubuntu-2.0.20210216",
					"firewall-ubuntu-2.0.20210207",
					"firewall-ubuntu-2.0.20210316",
				},
				Partitions: map[string]metal.Partition{
					"muc": {
						FirewallTypes: []string{
							"n1-medium-x86",
							"foo",
						},
					},
				},
			},
		},
	}

	providerConfig, err := helper.EncodeRawExtension(&cloudProfileConfig)
	if err != nil {
		return nil, err
	}

	profile := gardenv1beta1.CloudProfile{
		Spec: gardenv1beta1.CloudProfileSpec{
			ProviderConfig: providerConfig,
		},
	}

	return &profile, nil
}
