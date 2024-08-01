package validation

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
)

// ValidateCloudProviderSecret checks whether the given secret contains a valid AWS access keys.
func ValidateCloudProviderSecret(secret *corev1.Secret) error {
	creds, err := metal.ReadCredentialsSecret(secret)
	if err != nil {
		return fmt.Errorf("unable to read credentials secret: %w", err)
	}

	if creds.MetalAPIHMac != "" && creds.MetalAPIKey != "" {
		return fmt.Errorf("either hmac or api key must be set, not both")
	}
	if creds.MetalAPIHMac == "" && creds.MetalAPIKey == "" {
		return fmt.Errorf("either hmac or api key must be set")
	}

	return nil
}
