package worker

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

func (a *actuator) firewallMigrate(ctx context.Context, log logr.Logger, cluster *extensionscontroller.Cluster) error {
	var (
		namespace = cluster.ObjectMeta.Name

		fwdeploys = &fcmv2.FirewallDeploymentList{}
		fwsets    = &fcmv2.FirewallSetList{}
		firewalls = &fcmv2.FirewallList{}
	)

	err := a.client.List(ctx, firewalls, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewalls: %w", err)
	}

	if len(firewalls.Items) == 0 {
		log.Info("firewalls already migrated")
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

	log.Info("shallow deleting firewall entities for shoot migration")

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
	return controllerutils.RemoveAllFinalizers(ctx, c, object)
}
