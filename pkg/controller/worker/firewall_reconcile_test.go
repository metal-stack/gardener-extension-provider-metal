package worker

import (
	"github.com/google/go-cmp/cmp"
	"github.com/metal-stack/metal-lib/pkg/testcommon"

	"testing"
)

func Test_patchUpdate(t *testing.T) {
	tests := []struct {
		name    string
		old     string
		new     string
		want    bool
		wantErr error
	}{
		{
			name:    "no update",
			old:     "firewall-ubuntu-3.0",
			new:     "firewall-ubuntu-3.0",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "no update fully qualified",
			old:     "firewall-ubuntu-3.0.20240101",
			new:     "firewall-ubuntu-3.0.20240101",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "patch update",
			old:     "firewall-ubuntu-3.0.20240101",
			new:     "firewall-ubuntu-3.0.20240201",
			want:    true,
			wantErr: nil,
		},
		{
			name:    "minor update",
			old:     "firewall-ubuntu-3.0.20240101",
			new:     "firewall-ubuntu-3.1.20240101",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "major update",
			old:     "firewall-ubuntu-3.0.20240101",
			new:     "firewall-ubuntu-4.0.20240101",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "os update",
			old:     "firewall-ubuntu-3.0.20240101",
			new:     "firewall-debian-3.0.20240101",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "update to fully qualified",
			old:     "firewall-ubuntu-3.0",
			new:     "firewall-ubuntu-3.0.20240101",
			want:    true,
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchUpdate(tt.old, tt.new)
			if diff := cmp.Diff(tt.wantErr, err, testcommon.ErrorStringComparer()); diff != "" {
				t.Errorf("error diff (+got -want):\n %s", diff)
			}

			if diff := cmp.Diff(got, tt.want, testcommon.StrFmtDateComparer()); diff != "" {
				t.Errorf("diff (+got -want):\n %s", diff)
			}
		})
	}
}
