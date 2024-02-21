package controlplane

import (
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/google/go-cmp/cmp"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
)

func Test_firewallCompareFunc(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		fws  []*models.V1FirewallResponse
		want []*models.V1FirewallResponse
	}{
		{
			name: "single entry",
			fws: []*models.V1FirewallResponse{
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now))},
				},
			},
			want: []*models.V1FirewallResponse{
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now))},
				},
			},
		},
		{
			name: "three entries",
			fws: []*models.V1FirewallResponse{
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now.Add(1)))},
				},
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now.Add(-1)))},
				},
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now))},
				},
			},
			want: []*models.V1FirewallResponse{
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now.Add(1)))},
				},
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now))},
				},
				{
					Allocation: &models.V1MachineAllocation{Created: pointer.Pointer(strfmt.DateTime(now.Add(-1)))},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slices.SortFunc(tt.fws, firewallCompareFunc)

			if diff := cmp.Diff(tt.fws, tt.want, testcommon.StrFmtDateComparer()); diff != "" {
				t.Errorf("firewallLessFunc() = %s", diff)
			}

		})
	}
}

func Test_registryMirrorToValueMap(t *testing.T) {
	type args struct {
		r apismetal.RegistryMirror
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]any
		wantErr bool
	}{
		{
			name: "regular values",
			args: args{
				r: apismetal.RegistryMirror{
					Name:     "registry.example.com",
					Endpoint: "https://registry.example.host.com",
					IP:       "1.2.3.4",
					Port:     443,
					MirrorOf: []string{"test1", "test2"},
				},
			},
			want: map[string]any{
				"name":     "registry.example.com",
				"endpoint": "https://registry.example.host.com",
				"cidr":     "1.2.3.4/32",
				"port":     int32(443),
			},
			wantErr: false,
		},
		{
			name: "illegal IP",
			args: args{
				r: apismetal.RegistryMirror{
					Name:     "registry.example.com",
					Endpoint: "https://registry.example.host.com",
					IP:       "1.2.3.4.5",
					Port:     443,
					MirrorOf: []string{"test1", "test2"},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := registryMirrorToValueMap(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("registryMirrorToValueMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("registryMirrorToValueMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getDefaultExternalNetwork(t *testing.T) {
	var (
		internetFirewall = &apismetal.InfrastructureConfig{
			PartitionID: "a",
			ProjectID:   "own-project",
			Firewall: apismetal.Firewall{
				Networks: []string{
					"mpls-network",
					"own-external-network",
					"internet",
				},
			},
		}

		dmzFirewall = &apismetal.InfrastructureConfig{
			PartitionID: "a",
			ProjectID:   "own-project",
			Firewall: apismetal.Firewall{
				Networks: []string{
					"dmz-network",
				},
			},
		}

		nws = networkMap{
			"own-external-network": &models.V1NetworkResponse{
				ID:              pointer.Pointer("own-external-network"),
				Parentnetworkid: "",
				Projectid:       "own-project",
			},
			"somebody-external-network": &models.V1NetworkResponse{
				ID:              pointer.Pointer("somebody-external-network"),
				Parentnetworkid: "",
				Projectid:       "another-project",
			},
			"internet": &models.V1NetworkResponse{
				ID:              pointer.Pointer("internet"),
				Parentnetworkid: "",
				Labels: map[string]string{
					tag.NetworkDefaultExternal: "",
					tag.NetworkDefault:         "",
				},
			},
			"mpls-network": &models.V1NetworkResponse{
				ID:              pointer.Pointer("mpls-network"),
				Parentnetworkid: "",
				Labels: map[string]string{
					tag.NetworkDefaultExternal: "",
				},
			},
			"dmz-network": &models.V1NetworkResponse{
				ID:              pointer.Pointer("dmz-network"),
				Parentnetworkid: "super-network",
				Projectid:       "own-project",
				Shared:          true,
				Labels: map[string]string{
					tag.NetworkDefaultExternal: "",
				},
			},
			"super-network": &models.V1NetworkResponse{
				ID:           pointer.Pointer("super-network"),
				Projectid:    "",
				Privatesuper: pointer.Pointer(true),
			},
		}
	)

	tests := []struct {
		name                 string
		nws                  networkMap
		cpConfig             *apismetal.ControlPlaneConfig
		infrastructureConfig *apismetal.InfrastructureConfig
		want                 string
		wantErr              error
	}{
		{
			name:                 "specific default external network as specified by user",
			nws:                  nws,
			infrastructureConfig: internetFirewall,
			cpConfig: &apismetal.ControlPlaneConfig{
				CloudControllerManager: &apismetal.CloudControllerManagerConfig{
					DefaultExternalNetwork: pointer.Pointer("own-external-network"),
				},
			},
			want: "own-external-network",
		},
		{
			name:                 "cannot specify external network of somebody else",
			nws:                  nws,
			infrastructureConfig: internetFirewall,
			cpConfig: &apismetal.ControlPlaneConfig{
				CloudControllerManager: &apismetal.CloudControllerManagerConfig{
					DefaultExternalNetwork: pointer.Pointer("somebody-external-network"),
				},
			},
			wantErr: fmt.Errorf("given default external network not contained in firewall networks"),
		},
		{
			name:                 "use internet as default external network",
			nws:                  nws,
			infrastructureConfig: internetFirewall,
			cpConfig:             &apismetal.ControlPlaneConfig{},
			want:                 "internet",
		},
		{
			name:                 "fallback to dmz network",
			nws:                  nws,
			infrastructureConfig: dmzFirewall,
			cpConfig:             &apismetal.ControlPlaneConfig{},
			want:                 "dmz-network",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getDefaultExternalNetwork(tt.nws, tt.cpConfig, tt.infrastructureConfig)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(got, tt.want, testcommon.StrFmtDateComparer()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
