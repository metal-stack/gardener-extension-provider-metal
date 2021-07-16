package validation

import (
	"reflect"
	"testing"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

func Test_getLatestFirewallControllerVersion(t *testing.T) {
	tests := []struct {
		name              string
		availableVersions []apismetal.FirewallControllerVersion
		want              *apismetal.FirewallControllerVersion
		wantErr           bool
	}{
		{
			name:              "simple",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "v1.0.1"}, {Version: "v1.0.2"}, {Version: "v1.0.3"}},
			want:              &apismetal.FirewallControllerVersion{Version: "v1.0.3"},
			wantErr:           false,
		},
		{
			name:              "even more simple",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "v1.0.1"}, {Version: "v0.0.2"}, {Version: "v2.0.3"}, {Version: "v0.0.3"}},
			want:              &apismetal.FirewallControllerVersion{Version: "v2.0.3"},
			wantErr:           false,
		},
		{
			name:              "empty list",
			availableVersions: []apismetal.FirewallControllerVersion{},
			want:              nil,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLatestFirewallControllerVersion(tt.availableVersions)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLatestFirewallControllerVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLatestFirewallControllerVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
