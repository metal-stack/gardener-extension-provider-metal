package worker

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils/reconciler"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *actuator) firewallRestore(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	var (
		namespace = cluster.ObjectMeta.Name
	)

	fcm := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "firewall-controller-manager",
			Namespace: namespace,
		},
	}
	err := a.client.Get(ctx, client.ObjectKeyFromObject(fcm), fcm)
	if err != nil {
		return &reconciler.RequeueAfterError{
			Cause:        fmt.Errorf("unable to get firewall-controller-manager deployment: %w", err),
			RequeueAfter: 10 * time.Second,
		}
	}

	if fcm.Status.Replicas != fcm.Status.ReadyReplicas {
		return &reconciler.RequeueAfterError{
			Cause:        fmt.Errorf("firewall-controller-manager deployment is not yet ready, waiting..."),
			RequeueAfter: 10 * time.Second,
		}
	}

	a.logger.Info("restoring firewalls deployment")

	err = a.ensureFirewallDeployment(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return nil
}
