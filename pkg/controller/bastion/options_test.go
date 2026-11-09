package bastion

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"testing"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_generateBastionBaseResourceName(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		bastionName string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple",
			clusterName: "shoot--foo--bar",
			bastionName: "my-bastion",
			want:        fmt.Sprintf("%s-bastion-%s", "shoot--foo--bar-my-bastion", shortHash("shoot--foo--bar-my-bastion")),
		},
		{
			name:        "long name is truncated",
			clusterName: "shoot--foo--a-really-really-really-long-cluster-namespace",
			bastionName: "my-bastion",
			want: fmt.Sprintf("%s-bastion-%s",
				"shoot--foo--a-really-really-really-long-cluster-namespace-my-bastion"[:maxLengthForBaseName],
				shortHash("shoot--foo--a-really-really-really-long-cluster-namespace-my-bastion")),
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			bastionName: "my-bastion",
			wantErr:     true,
		},
		{
			name:        "empty bastion name",
			clusterName: "shoot--foo--bar",
			bastionName: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateBastionBaseResourceName(tt.clusterName, tt.bastionName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("generateBastionBaseResourceName() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("generateBastionBaseResourceName() = %q, want %q", got, tt.want)
			}
			if !regexp.MustCompile(`-bastion-[0-9a-f]{5}$`).MatchString(got) {
				t.Errorf("generated name %q does not match expected format", got)
			}
		})
	}
}

func shortHash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))[:5]
}

func Test_providerStatusRoundTrip(t *testing.T) {
	status := &providerStatusRaw{
		MachineName: "shoot--foo--bar-bastion-bastion-abcde",
		MachineID:   "machine-id-123",
	}

	raw, err := marshalProviderStatus(status)
	if err != nil {
		t.Fatal(err)
	}

	bastion := &extensionsv1alpha1.Bastion{
		Status: extensionsv1alpha1.BastionStatus{
			DefaultStatus: extensionsv1alpha1.DefaultStatus{
				ProviderStatus: &runtime.RawExtension{Raw: raw},
			},
		},
	}

	got, err := getProviderStatus(bastion)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected provider status, got nil")
	}
	if got.MachineName != status.MachineName || got.MachineID != status.MachineID {
		t.Errorf("unexpected provider status: %+v", got)
	}
}

func Test_getProviderStatus_empty(t *testing.T) {
	bastion := &extensionsv1alpha1.Bastion{
		ObjectMeta: metav1.ObjectMeta{Name: "bastion"},
	}

	got, err := getProviderStatus(bastion)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil provider status, got %+v", got)
	}
}
