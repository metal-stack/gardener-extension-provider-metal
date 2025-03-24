package controlplane

import (
	"testing"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/google/go-cmp/cmp"
	apimetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func Test_ensureContainerdRegistries(t *testing.T) {
	tests := []struct {
		name    string
		mirrors []apimetal.RegistryMirror
		configs []extensionsv1alpha1.RegistryConfig
		want    []extensionsv1alpha1.RegistryConfig
	}{
		{
			name:    "maintain existing config",
			mirrors: nil,
			configs: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "eu.gcr.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-b",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability},
						},
					},
				},
			},
			want: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "eu.gcr.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-b",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability},
						},
					},
				},
			},
		},
		{
			name: "add new config",
			mirrors: []apimetal.RegistryMirror{
				{
					Name:     "test registry",
					Endpoint: "https://registry-a",
					IP:       "1.1.1.1",
					Port:     443,
					MirrorOf: []string{"quay.io"},
				},
			},
			configs: nil,
			want: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "quay.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-a",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
						},
					},
					ReadinessProbe: pointer.Pointer(false),
				},
			},
		},
		{
			name: "update outdated config",
			mirrors: []apimetal.RegistryMirror{
				{
					Name:     "test registry",
					Endpoint: "https://registry-b",
					IP:       "1.1.1.1",
					Port:     443,
					MirrorOf: []string{"quay.io"},
				},
			},
			configs: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "quay.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-a",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
						},
					},
					ReadinessProbe: pointer.Pointer(false),
				},
			},
			want: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "quay.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-b",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
						},
					},
					ReadinessProbe: pointer.Pointer(false),
				},
			},
		},
		{
			name: "append to existing config",
			mirrors: []apimetal.RegistryMirror{
				{
					Name:     "test registry",
					Endpoint: "https://registry-a",
					IP:       "1.1.1.1",
					Port:     443,
					MirrorOf: []string{"quay.io"},
				},
			},
			configs: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "eu.gcr.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-a",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability},
						},
					},
				},
			},
			want: []extensionsv1alpha1.RegistryConfig{
				{
					Upstream: "quay.io",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "https://registry-a",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability},
						},
					},
				},
				{
					Upstream: "registry-a",
					Hosts: []extensionsv1alpha1.RegistryHost{
						{
							URL:          "quay.io",
							Capabilities: []extensionsv1alpha1.RegistryCapability{extensionsv1alpha1.PullCapability, extensionsv1alpha1.ResolveCapability},
						},
					},
					ReadinessProbe: pointer.Pointer(false),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureContainerdRegistries(tt.mirrors, tt.configs)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("diff = %s", diff)
			}
		})
	}
}
