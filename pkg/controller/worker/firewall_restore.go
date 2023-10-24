package worker

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils/reconciler"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *actuator) firewallRestore(ctx context.Context, log logr.Logger, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
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

	log.Info("restoring state from infrastructure status")

	d, err := a.getAdditionalData(ctx, worker, cluster)
	if err != nil {
		return fmt.Errorf("error getting additional data: %w", err)
	}

	err = a.restoreState(ctx, log, d.infrastructure)
	if err != nil {
		return fmt.Errorf("error restoring firewall state: %w", err)
	}

	log.Info("restoring firewalls deployment")

	sshSecret, err := helper.GetLatestSSHSecret(ctx, a.client, worker.Namespace)
	if err != nil {
		return fmt.Errorf("could not find current ssh secret: %w", err)
	}

	err = a.ensureFirewallDeployment(ctx, log, d, cluster, string(sshSecret.Data["id_rsa.pub"]))
	if err != nil {
		return err
	}

	return nil
}
