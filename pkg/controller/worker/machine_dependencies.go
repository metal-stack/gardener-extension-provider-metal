package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	retryutils "github.com/gardener/gardener/pkg/utils/retry"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
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
	if w.worker.DeletionTimestamp == nil {
		return nil
	}

	return w.ensureFirewallDeploymentDeleted(ctx)
}

func (w *workerDelegate) ensureFirewallDeploymentDeleted(ctx context.Context) error {
	w.logger.Info("ensuring firewall deployment gets deleted")

	return retryutils.UntilTimeout(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (bool, error) {
		deploy := &fcmv2.FirewallDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      metal.FirewallDeploymentName,
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
			w.logger.Info("deleting firewall deployment")
			err = w.client.Delete(ctx, deploy)
			if err != nil {
				return retryutils.SevereError(fmt.Errorf("error deleting firewall deployment: %w", err))
			}

			return retryutils.MinorError(errors.New("firewall deployment is still ongoing"))
		}

		return retryutils.MinorError(errors.New("firewall deployment is still ongoing"))
	})
}
