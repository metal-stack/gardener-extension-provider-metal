package infrastructure

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
)

func (a *actuator) Migrate(ctx context.Context, logger logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	// during shoot migration, the cloud provider secret for the MCM is not being deleted preventing the migration to succeed
	// because the secret is blocking the shoot namespace deletion.
	//
	// this is probably due to the reason that our MCM is provided OOT.

	var (
		deleteFinalizerName = "machine.sapcloud.io/machine-controller"
	)

	var secrets corev1.SecretList
	err := a.client.List(ctx, &secrets, client.MatchingLabels{
		"garden.sapcloud.io/purpose": "machineclass",
	}, client.InNamespace(cluster.ObjectMeta.Name))
	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		secret := secret.DeepCopy()

		if !strings.HasPrefix(secret.Name, "shoot--") {
			continue
		}

		if controllerutil.ContainsFinalizer(secret, deleteFinalizerName) {
			a.logger.Info("removing dangling finalizer from mcm secret", "secret", secret.Name)

			controllerutil.RemoveFinalizer(secret, deleteFinalizerName)

			err = a.client.Update(ctx, secret)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
