package mutator

import (
	"testing"

	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumextensionv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/install"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func Test_defaulter_defaultShoot(t *testing.T) {
	scheme := runtime.NewScheme()
	install.Install(scheme)

	var (
		decoder          = serializer.NewCodecFactory(scheme).UniversalDecoder()
		examplePartition = &metal.Partition{
			FirewallTypes: []string{
				"n1-medium-x86",
				"foo",
			},
		}
		exampleControlPlane = &metal.MetalControlPlane{
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
				"muc": *examplePartition,
			},
		}
		completeInfrastructureConfig = &metalv1alpha1.InfrastructureConfig{
			Firewall: metalv1alpha1.Firewall{
				Image: "firewall-ubuntu-2.0.19700101",
				Size:  "n1-medium-x86",
			},
		}
		completeCiliumSpec = &ciliumextensionv1alpha1.NetworkConfig{
			Debug:      pointer.Pointer(true),
			PSPEnabled: pointer.Pointer(true),
			KubeProxy: &ciliumextensionv1alpha1.KubeProxy{
				ServiceHost: pointer.Pointer("service-host"),
				ServicePort: pointer.Pointer(int32(1)),
			},
			Hubble: &ciliumextensionv1alpha1.Hubble{
				Enabled: false,
			},
			TunnelMode: pointer.Pointer(ciliumextensionv1alpha1.Geneve),
			Store:      pointer.Pointer(ciliumextensionv1alpha1.Kubernetes),
			IPv6: &ciliumextensionv1alpha1.IPv6{
				Enabled: true,
			},
		}
		completeShootSpec = &gardenv1beta1.Shoot{
			Spec: gardenv1beta1.ShootSpec{
				Kubernetes: gardenv1beta1.Kubernetes{
					Version:                   "1.24.0",
					AllowPrivilegedContainers: pointer.Pointer(false),
					KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
						NodeCIDRMaskSize: pointer.Pointer(int32(24)),
					},
					Kubelet: &gardenv1beta1.KubeletConfig{
						MaxPods: pointer.Pointer(int32(200)),
					},
					KubeProxy: &gardenv1beta1.KubeProxyConfig{
						Enabled: pointer.Pointer(true),
					},
				},
				Networking: &gardenv1beta1.Networking{
					Type:           pointer.Pointer("cilium"),
					ProviderConfig: mustEncode(t, completeCiliumSpec),
					Pods:           pointer.Pointer("10.240.0.0/14"),
					Services:       pointer.Pointer("10.248.0.0/19"),
				},
				Provider: gardenv1beta1.Provider{
					InfrastructureConfig: mustEncode(t, completeInfrastructureConfig),
				},
			},
		}
	)

	tests := []struct {
		name  string
		shoot *gardenv1beta1.Shoot
		want  *gardenv1beta1.Shoot
	}{
		{
			name: "empty spec",
			shoot: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version: "1.24.0",
					},
				},
			},
			want: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version: "1.24.0",
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(23)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(250)),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "complete spec does not change",
			shoot: completeShootSpec.DeepCopy(),
			want:  completeShootSpec.DeepCopy(),
		},
		{
			name: "if networking config is present, provider config stays untouched",
			shoot: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
					},
					Networking: &gardenv1beta1.Networking{
						Type: pointer.Pointer("calico"),
						ProviderConfig: &runtime.RawExtension{
							Object: &calicoextensionv1alpha1.NetworkConfig{
								Backend: pointer.Pointer(calicoextensionv1alpha1.Bird),
							},
						},
						Pods:     pointer.Pointer("10.240.0.0/14"),
						Services: pointer.Pointer("10.248.0.0/19"),
					},
				},
			},
			want: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
					},
					Networking: &gardenv1beta1.Networking{
						Type:     pointer.Pointer("calico"),
						Pods:     pointer.Pointer("10.240.0.0/14"),
						Services: pointer.Pointer("10.248.0.0/19"),
						ProviderConfig: &runtime.RawExtension{
							Object: &calicoextensionv1alpha1.NetworkConfig{
								Backend: pointer.Pointer(calicoextensionv1alpha1.Bird),
							},
						},
					},
				},
			},
		},
		{
			name: "empty provider config will be defaulted",
			shoot: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
						Workers: []gardenv1beta1.Worker{
							{},
						},
					},
				},
			},
			want: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: pointer.Pointer(true),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
						Workers: []gardenv1beta1.Worker{
							{},
						},
					},
					Networking: &gardenv1beta1.Networking{
						Type:     pointer.Pointer("calico"),
						Pods:     pointer.Pointer("10.240.0.0/13"),
						Services: pointer.Pointer("10.248.0.0/18"),
						ProviderConfig: &runtime.RawExtension{
							Object: &calicoextensionv1alpha1.NetworkConfig{
								Backend: pointer.Pointer(calicoextensionv1alpha1.None),
								IPv4: &calicoextensionv1alpha1.IPv4{
									Mode: pointer.Pointer(calicoextensionv1alpha1.Never),
								},
								Typha: &calicoextensionv1alpha1.Typha{
									Enabled: false,
								},
							},
						},
					},
				},
			},
		},
		{
			name: "cilium networking provider config will be defaulted",
			shoot: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
						Workers: []gardenv1beta1.Worker{
							{},
						},
					},
					Networking: &gardenv1beta1.Networking{
						Type: pointer.Pointer("cilium"),
					},
				},
			},
			want: &gardenv1beta1.Shoot{
				Spec: gardenv1beta1.ShootSpec{
					Kubernetes: gardenv1beta1.Kubernetes{
						Version:                   "1.24.0",
						AllowPrivilegedContainers: pointer.Pointer(false),
						KubeControllerManager: &gardenv1beta1.KubeControllerManagerConfig{
							NodeCIDRMaskSize: pointer.Pointer(int32(24)),
						},
						Kubelet: &gardenv1beta1.KubeletConfig{
							MaxPods: pointer.Pointer(int32(200)),
						},
						KubeProxy: &gardenv1beta1.KubeProxyConfig{
							Enabled: pointer.Pointer(false),
						},
					},
					Provider: gardenv1beta1.Provider{
						InfrastructureConfig: &runtime.RawExtension{
							Object: &metalv1alpha1.InfrastructureConfig{
								Firewall: metalv1alpha1.Firewall{
									Image: "firewall-2.0.20210207",
									Size:  "n1-medium-x86",
								},
							},
						},
						Workers: []gardenv1beta1.Worker{
							{},
						},
					},
					Networking: &gardenv1beta1.Networking{
						Type:     pointer.Pointer("cilium"),
						Pods:     pointer.Pointer("10.240.0.0/13"),
						Services: pointer.Pointer("10.248.0.0/18"),
						ProviderConfig: &runtime.RawExtension{
							Object: &ciliumextensionv1alpha1.NetworkConfig{
								PSPEnabled: pointer.Pointer(true),
								Hubble: &ciliumextensionv1alpha1.Hubble{
									Enabled: true,
								},
								TunnelMode:                   pointer.Pointer(ciliumextensionv1alpha1.Disabled),
								MTU:                          pointer.Pointer(1440),
								Devices:                      []string{"lan+"},
								LoadBalancingMode:            pointer.Pointer(ciliumextensionv1alpha1.DSR),
								IPv4NativeRoutingCIDREnabled: pointer.Pointer(true),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &defaulter{
				decoder:      decoder,
				controlPlane: exampleControlPlane,
				partition:    examplePartition,
			}

			got := tt.shoot.DeepCopy()

			err := d.defaultShoot(got)

			require.NoError(t, err)

			if diff := cmp.Diff(tt.want, got, cmpopts.IgnoreFields(runtime.RawExtension{}, "Raw")); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}

func mustEncode(t *testing.T, from runtime.Object) *runtime.RawExtension {
	enc, err := helper.EncodeRawExtension(from)
	require.NoError(t, err)
	enc.Object = from
	return enc
}
