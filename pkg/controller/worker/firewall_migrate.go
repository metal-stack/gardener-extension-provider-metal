package worker

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/gardener/gardener/pkg/controllerutils"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
)

func (a *actuator) firewallMigrate(ctx context.Context, cluster *extensionscontroller.Cluster) error {
	// approach is to restore firewalls from the firewall monitors in the shoot cluster

	var (
		namespace = cluster.ObjectMeta.Name

		fwdeploys = &fcmv2.FirewallDeploymentList{}
		fwsets    = &fcmv2.FirewallSetList{}
		firewalls = &fcmv2.FirewallList{}
		mons      = &fcmv2.FirewallMonitorList{}
	)

	err := a.client.List(ctx, fwdeploys, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewall deployments: %w", err)
	}

	err = a.client.List(ctx, fwsets, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewall sets: %w", err)
	}

	err = a.client.List(ctx, firewalls, client.InNamespace(namespace))
	if err != nil {
		return fmt.Errorf("error listing firewalls: %w", err)
	}

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create shoot client: %w", err)
	}

	err = shootClient.List(ctx, mons, &client.ListOptions{Namespace: fcmv2.FirewallShootNamespace})
	if err != nil {
		return fmt.Errorf("error listing firewall monitors: %w", err)
	}

	a.logger.Info("migrating firewalls", "amount", len(firewalls.Items), "monitors-amount", len(mons.Items))

	if !everyFirewallHasAMonitor(firewalls, mons) {
		return fmt.Errorf("every firewall needs to have a corresponding firewall monitor before migration, because firewalls are restored from the monitors")
	}

	a.logger.Info("shallow deleting firewall entities for shoot migration")

	if err := removeFinalizersAllObjects(ctx, a.client, mons); err != nil {
		return err
	}
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

func removeFinalizersAllObjects(ctx context.Context, c client.Client, objectList client.ObjectList) error {
	return meta.EachListItem(objectList, func(obj runtime.Object) error {
		object := obj.(client.Object)
		return removeFinalizersObject(ctx, c, object)
	})
}

func removeFinalizersObject(ctx context.Context, c client.Client, object client.Object) error {
	return controllerutils.RemoveAllFinalizers(ctx, c, c, object)
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
	if err := c.Delete(ctx, object); client.IgnoreNotFound(err) != nil {
		return err
	}

	return nil
}

func everyFirewallHasAMonitor(firewalls *fcmv2.FirewallList, mons *fcmv2.FirewallMonitorList) bool {
	var (
		fwNames  []string
		monNames []string
	)

	for _, fw := range firewalls.Items {
		fwNames = append(fwNames, fw.Name)
	}
	for _, mon := range mons.Items {
		monNames = append(monNames, mon.Name)
	}

	return sets.NewString(monNames...).Equal(sets.NewString(fwNames...))
}
