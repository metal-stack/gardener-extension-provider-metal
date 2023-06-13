package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
)

// Restore takes the infrastructure state and deploys it as terraform state ConfigMap before calling the terraformer
func (a *actuator) Restore(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, _ *extensionscontroller.Cluster) error {
	infraState := &InfrastructureState{}
	err := json.Unmarshal(infrastructure.Status.State.Raw, infraState)
	if err != nil {
		return fmt.Errorf("unable to decode infrastructure status: %w", err)
	}

	a.logger.Info("restoring firewalls and service accounts", "firewalls", len(infraState.Firewalls), "service-accounts", len(infraState.SeedAccess))

	for _, raw := range infraState.Firewalls {
		raw := raw

		fw := &fcmv2.Firewall{}
		err := yaml.Unmarshal(raw, fw)
		if err != nil {
			return err
		}

		err = a.client.Create(ctx, fw)
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("unable restoring firewall resource: %w", err)
		}
	}

	for _, seedAccess := range infraState.SeedAccess {

		sa := &corev1.ServiceAccount{}
		err := yaml.Unmarshal(seedAccess.ServiceAccount, sa)
		if err != nil {
			return err
		}

		err = a.client.Create(ctx, sa)
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("unable restoring service account: %w", err)
		}

		for _, raw := range seedAccess.ServiceAccountSecrets {
			raw := raw

			secret := &corev1.Secret{}
			err := yaml.Unmarshal(raw, secret)
			if err != nil {
				return err
			}

			err = a.client.Create(ctx, secret)
			if client.IgnoreNotFound(err) != nil {
				return fmt.Errorf("unable restoring service account secret %q: %w", secret.Name, err)
			}
		}
	}

	a.logger.Info("successfully restored infrastructure")

	return nil
}
