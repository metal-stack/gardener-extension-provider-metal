package controlplane

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/google/go-cmp/cmp"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/pointer"
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
