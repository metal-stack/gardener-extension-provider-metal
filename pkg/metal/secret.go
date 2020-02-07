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
		MetalAPIURL:  string(secret.Data[APIURL]),
		MetalAPIHMac: string(secret.Data[APIHMac]),
		MetalAPIKey:  string(secret.Data[APIKey]),
		CloudAPIURL:  string(secret.Data[CloudAPIURL]),
		CloudAPIHMac: string(secret.Data[CloudAPIHMac]),
		CloudAPIKey:  string(secret.Data[CloudAPIKey]),
	}, nil
}
