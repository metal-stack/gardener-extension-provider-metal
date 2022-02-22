package infrastructure

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

func (a *actuator) Migrate(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	// during shoot migration, the cloud provider secret for the MCM is not being deleted preventing the migration to succeed
	// because the secret is blocking the shoot namespace deletion.
	//
	// this is probably due to the reason that our MCM is provided OOT.

	var (
		deleteFinalizerName = "machine.sapcloud.io/machine-controller"
	)

	a.logger.Info("debugmigrate")

	var secrets corev1.SecretList
	err := a.client.List(ctx, &secrets, client.MatchingLabels{
		"garden.sapcloud.io/purpose": "machineclass",
	}, client.InNamespace(cluster.ObjectMeta.Name))
	if err != nil {
		return err
	}

	a.logger.Info("debugmigrate", "secrets", len(secrets.Items))

	for _, secret := range secrets.Items {
		secret := secret.DeepCopy()

		a.logger.Info("debugmigrate", "number", 1)

		if !strings.HasPrefix(secret.Name, "shoot--") {
			continue
		}
		a.logger.Info("debugmigrate", "number", 2)

		if secret.DeletionTimestamp == nil || secret.DeletionTimestamp.IsZero() {
			continue
		}
		a.logger.Info("debugmigrate", "number", 3)

		if !sets.NewString(secret.Finalizers...).Has(deleteFinalizerName) {
			continue
		}
		a.logger.Info("debugmigrate", "number", 4)

		a.logger.Info("removing dangling finalizer from mcm secret", "secret", secret.Name)
		controllerutil.RemoveFinalizer(secret, deleteFinalizerName)

		err = a.client.Update(ctx, secret)
		if err != nil {
			return err
		}
	}

	return nil
}
