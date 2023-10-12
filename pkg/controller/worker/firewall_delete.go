package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	retryutils "github.com/gardener/gardener/pkg/utils/retry"
	"github.com/go-logr/logr"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *actuator) firewallDelete(ctx context.Context, log logr.Logger, cluster *extensionscontroller.Cluster) error {
	log.Info("ensuring firewall deployment gets deleted")

	return retryutils.UntilTimeout(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (bool, error) {
		deploy := &fcmv2.FirewallDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      metal.FirewallDeploymentName,
				Namespace: cluster.ObjectMeta.Name,
			},
		}

		err := a.client.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Info("firewall deployment deletion succeeded")
				return retryutils.Ok()
			}
			return retryutils.SevereError(fmt.Errorf("error getting firewall deployment: %w", err))
		}

		if deploy.DeletionTimestamp == nil {
			log.Info("deleting firewall deployment")
			err = a.client.Delete(ctx, deploy)
			if err != nil {
				return retryutils.SevereError(fmt.Errorf("error deleting firewall deployment: %w", err))
			}

			return retryutils.MinorError(errors.New("firewall deployment is still ongoing"))
		}

		return retryutils.MinorError(errors.New("firewall deployment is still ongoing"))
	})
}
