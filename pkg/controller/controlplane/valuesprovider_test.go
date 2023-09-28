package controlplane

import (
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/testcommon"
	"golang.org/x/exp/slices"
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
