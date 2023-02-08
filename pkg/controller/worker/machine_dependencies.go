package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	retryutils "github.com/gardener/gardener/pkg/utils/retry"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeployMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) DeployMachineDependencies(_ context.Context) error {
	return nil
}

// CleanupMachineDependencies implements genericactuator.WorkerDelegate.
func (w *workerDelegate) CleanupMachineDependencies(ctx context.Context) error {
	worker := w.worker.DeepCopy()
	err := w.client.Get(ctx, client.ObjectKeyFromObject(w.worker), worker)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if worker.DeletionTimestamp == nil {
		return nil
	}

	err = w.deleteFirewallDeployment(ctx)
	if err != nil {
		return err
	}

	return w.waitForFirewallDeploymentDeletion(ctx)
}

func (w *workerDelegate) deleteFirewallDeployment(ctx context.Context) error {
	deploy := &fcmv2.FirewallDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      firewallDeploymentName,
			Namespace: w.cluster.ObjectMeta.Name,
		},
	}

	err := w.client.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("unable to find firewall deployment: %w", err)
	}

	if deploy.DeletionTimestamp != nil {
		w.logger.Info("deletion timestamp on firewall deployment already set")
		return nil
	}

	err = w.client.Delete(ctx, deploy)
	if err != nil {
		return fmt.Errorf("error deleting firewall deployment: %w", err)
	}

	w.logger.Info("firewall deployment deletion started")

	return nil
}

func (w *workerDelegate) waitForFirewallDeploymentDeletion(ctx context.Context) error {
	w.logger.Info("waiting until firewall deployment was deleted")

	return retryutils.UntilTimeout(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (bool, error) {
		deploy := &fcmv2.FirewallDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      firewallDeploymentName,
				Namespace: w.cluster.ObjectMeta.Name,
			},
		}

		err := w.client.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)
		if err != nil {
			if apierrors.IsNotFound(err) {
				w.logger.Info("firewall deployment deletion succeeded")
				return retryutils.Ok()
			}
			return retryutils.SevereError(fmt.Errorf("error getting firewall deployment: %w", err))
		}

		if deploy.DeletionTimestamp == nil {
			return retryutils.SevereError(fmt.Errorf("deletion timestamp not set on firewall deployment"))
		}

		return retryutils.MinorError(errors.New("machine class credentials secret has not yet been acquired or released"))
	})
}
