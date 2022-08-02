package validation

import (
	"reflect"
	"testing"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

func Test_getLatestFirewallControllerVersion(t *testing.T) {
	preview := apismetal.ClassificationPreview
	supported := apismetal.ClassificationSupported

	tests := []struct {
		name              string
		availableVersions []apismetal.FirewallControllerVersion
		want              *apismetal.FirewallControllerVersion
		wantErr           bool
	}{
		{
			name:              "simple",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "v1.0.1", Classification: &supported}, {Version: "v1.0.2", Classification: &supported}, {Version: "v1.0.3", Classification: &supported}},
			want:              &apismetal.FirewallControllerVersion{Version: "v1.0.3", Classification: &supported},
			wantErr:           false,
		},
		{
			name:              "even more simple",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "v1.0.1", Classification: &preview}, {Version: "v0.0.2", Classification: &supported}, {Version: "v2.0.3", Classification: &supported}, {Version: "v0.0.3", Classification: &supported}},
			want:              &apismetal.FirewallControllerVersion{Version: "v2.0.3", Classification: &supported},
			wantErr:           false,
		},
		{
			name:              "one version is specified with git sha",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "v1.0.1", Classification: &supported}, {Version: "2fb7fd7", Classification: &preview}, {Version: "v2.0.3", Classification: &supported}, {Version: "v0.0.3", Classification: &supported}},
			want:              &apismetal.FirewallControllerVersion{Version: "v2.0.3", Classification: &supported},
			wantErr:           false,
		},
		{
			name:              "only one version is specified semver compatible",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "1fb7fd7"}, {Version: "2fb7fd7", Classification: &preview}, {Version: "v2.0.3", Classification: &supported}, {Version: "4fb7fd7", Classification: &supported}},
			want:              &apismetal.FirewallControllerVersion{Version: "v2.0.3", Classification: &supported},
			wantErr:           false,
		},
		{
			name:              "latest version is preview",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "1fb7fd7"}, {Version: "2fb7fd7", Classification: &preview}, {Version: "v2.0.3", Classification: &supported}, {Version: "v2.1.0", Classification: &preview}},
			want:              &apismetal.FirewallControllerVersion{Version: "v2.0.3", Classification: &supported},
			wantErr:           false,
		},
		{
			name:              "no version is specified semver compatible",
			availableVersions: []apismetal.FirewallControllerVersion{{Version: "1fb7fd7", Classification: &preview}, {Version: "2fb7fd7", Classification: &preview}, {Version: "4fb7fd7", Classification: &preview}},
			want:              nil,
			wantErr:           true,
		},
		{
			name:              "empty list",
			availableVersions: []apismetal.FirewallControllerVersion{},
			want:              nil,
			wantErr:           true,
		},
	}
	for _, tt := range tests {
		tt := tt
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
