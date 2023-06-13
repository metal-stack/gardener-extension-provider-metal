package worker

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurectrl "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/infrastructure"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func (a *actuator) firewallMigrate(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	var (
		namespace = cluster.ObjectMeta.Name

		fwdeploys = &fcmv2.FirewallDeploymentList{}
		fwsets    = &fcmv2.FirewallSetList{}
		firewalls = &fcmv2.FirewallList{}
	)

	d, err := a.getAdditionalData(ctx, worker, cluster)
	if err != nil {
		return fmt.Errorf("error getting additional data: %w", err)
	}

	_, infraStatus, err := infrastructurectrl.DecodeInfrastructure(d.infrastructure, a.decoder)
	if err != nil {
		return err
	}

	err = infrastructurectrl.UpdateProviderStatus(ctx, a.client, d.infrastructure, infraStatus, d.infrastructure.Status.NodesCIDR)
	if err != nil {
		return fmt.Errorf("error updating infrastructure status")
	}

	err = a.client.List(ctx, firewalls, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewalls: %w", err)
	}

	if len(firewalls.Items) == 0 {
		a.logger.Info("firewalls already migrated")
		return nil
	}

	err = a.client.List(ctx, fwsets, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewall sets: %w", err)
	}

	err = a.client.List(ctx, fwdeploys, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewall deployments: %w", err)
	}

	a.logger.Info("shallow deleting firewall entities for shoot migration")

	if err := shallowDeleteAllObjects(ctx, a.client, fwdeploys); err != nil {
		return fmt.Errorf("error shallow deleting firewall deployments: %w", err)
	}
	if err := shallowDeleteAllObjects(ctx, a.client, fwsets); err != nil {
		return fmt.Errorf("error shallow deleting firewall sets: %w", err)
	}
	if err := shallowDeleteAllObjects(ctx, a.client, firewalls); err != nil {
		return fmt.Errorf("error shallow deleting firewalls: %w", err)
	}

	return nil
}

func shallowDeleteAllObjects(ctx context.Context, c client.Client, objectList client.ObjectList) error {
	return meta.EachListItem(objectList, func(obj runtime.Object) error {
		object := obj.(client.Object)
		return shallowDeleteObject(ctx, c, object)
	})
}

func shallowDeleteObject(ctx context.Context, c client.Client, object client.Object) error {
	if err := removeFinalizersObject(ctx, c, object); err != nil {
		return err
	}
	if err := c.Delete(ctx, object, &client.DeleteOptions{
		PropagationPolicy: pointer.Pointer(metav1.DeletePropagationOrphan),
	}); client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

func removeFinalizersObject(ctx context.Context, c client.Client, object client.Object) error {
	return controllerutils.RemoveAllFinalizers(ctx, c, c, object)
}
