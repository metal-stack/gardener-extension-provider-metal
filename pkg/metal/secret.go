package metal

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// ReadCredentialsSecret reads a secret containing credentials.
func ReadCredentialsSecret(secret *corev1.Secret) (*Credentials, error) {
	if secret.Data == nil {
		return nil, fmt.Errorf("secret does not contain any data")
	}

	return &Credentials{
		MetalAPIHMac: string(secret.Data[APIHMac]),
		MetalAPIKey:  string(secret.Data[APIKey]),
	}, nil
}
