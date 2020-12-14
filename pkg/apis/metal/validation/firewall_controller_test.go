package validation

import (
	"testing"

	"github.com/gardener/gardener/pkg/utils/imagevector"
)

func Test_validateFirewallControllerVersionWithoutGithub(t *testing.T) {
	v0_1_0 := "v0.1.0"
	v0_2_0 := "v0.2.0"
	v1_0_0 := "v1.0.0"
	abc := "abc"
	tests := []struct {
		name        string
		iv          imagevector.ImageVector
		specVersion *string
		autoUpdate  bool
		want        *string
		wantErr     error
	}{
		{
			name: "do not modify former shoot spec",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &v0_2_0,
				},
			},
			specVersion: nil,
			autoUpdate:  false,
			want:        nil,
			wantErr:     ErrSpecVersionUndefined,
		},
		{
			name: "update to newer minor version given in image vector",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &v0_2_0,
				},
			},
			specVersion: &v0_1_0,
			autoUpdate:  true,
			want:        &v0_2_0,
		},
		{
			name: "downgrade to older minor version given in image vector",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &v0_1_0,
				},
			},
			specVersion: &v0_2_0,
			autoUpdate:  true,
			want:        &v0_1_0,
		},
		{
			name: "major version updates may contain api changes btw. gepm and firewall-controller and are not supported",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &v1_0_0,
				},
			},
			specVersion: &v0_1_0,
			autoUpdate:  true,
			wantErr:     ErrControllerTooOld,
		},
		{
			name: "spec contains no semver version",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &v0_1_0,
				},
			},
			specVersion: &abc,
			autoUpdate:  true,
			wantErr:     ErrNoSemver,
		},
		{
			name: "image vector contains no semver version",
			iv: imagevector.ImageVector{
				&imagevector.ImageSource{
					Name: "firewall-controller",
					Tag:  &abc,
				},
			},
			specVersion: &v0_1_0,
			autoUpdate:  true,
			wantErr:     ErrNoSemver,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateFirewallControllerVersionWithoutGithub(tt.iv, tt.specVersion, tt.autoUpdate)
			if err != tt.wantErr {
				t.Errorf("validateFirewallControllerVersionWithoutGithub() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.want == nil || got == nil {
				if tt.want != nil || got != nil {
					t.Errorf("error")
				}
			} else if *got != *tt.want {
				t.Errorf("validateFirewallControllerVersionWithoutGithub() = %v, want %v", *got, *tt.want)
			}
		})
	}
}
