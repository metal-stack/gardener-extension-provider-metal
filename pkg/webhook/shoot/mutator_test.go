package shoot

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_extractShootNameFromSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  *corev1.Secret
		want    string
		wantErr bool
	}{
		{
			name: "a simple test",
			secret: &corev1.Secret{
				ObjectMeta: v1.ObjectMeta{
					Annotations: map[string]string{
						"resources.gardener.cloud/origin": "shoot--test--fra-equ01-8fef639c-bbe4-4c6f-9656-617dc4a4efd8-gardener-soil-test:shoot--pjb9j2--forbidden/shoot-cloud-config-execution",
					},
				},
			},
			want: "shoot--pjb9j2--forbidden",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractShootNameFromSecret(tt.secret)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractShootNameFromSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractShootNameFromSecret() = %v, want %v", got, tt.want)
			}
		})
	}
}
