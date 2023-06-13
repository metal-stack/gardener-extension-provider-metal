package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Restore takes the infrastructure state and deploys it as terraform state ConfigMap before calling the terraformer
func (a *actuator) Restore(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, _ *extensionscontroller.Cluster) error {
	infraState := &InfrastructureState{}
	err := json.Unmarshal(infrastructure.Status.State.Raw, infraState)
	if err != nil {
		return fmt.Errorf("unable to decode infrastructure status: %w", err)
	}

	a.logger.Info("restoring firewalls and service accounts", "firewalls", len(infraState.Firewalls), "service-accounts", len(infraState.SeedAccessTokens))

	for _, fw := range infraState.Firewalls {
		fw := fw
		err = a.client.Create(ctx, &fw)
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("unable restoring firewall resource: %w", err)
		}
	}

	for _, seedAccess := range infraState.SeedAccessTokens {
		err = a.client.Create(ctx, &seedAccess.ServiceAccount)
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("unable restoring service account %q: %w", seedAccess.ServiceAccount.Name, err)
		}

		for _, secret := range seedAccess.ServiceAccountSecrets {
			secret := secret
			err = a.client.Create(ctx, &secret)
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("unable restoring service account secret %q: %w", secret.Name, err)
			}
		}
	}

	a.logger.Info("successfully restored infrastructure")

	return nil
}
