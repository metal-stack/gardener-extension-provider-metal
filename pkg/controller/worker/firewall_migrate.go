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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/metal-lib/pkg/pointer"
)

const (
	migrationSecretKey = "gardener-extension-provider-metal/shoot-migration"
)

func (a *actuator) firewallMigrate(ctx context.Context, cluster *extensionscontroller.Cluster) error {
	// approach is to restore firewalls from the firewall monitors in the shoot cluster
	// we also need to store the service accounts secret for the firewall-controller to access the seed

	var (
		namespace = cluster.ObjectMeta.Name

		fwdeploys = &fcmv2.FirewallDeploymentList{}
		fwsets    = &fcmv2.FirewallSetList{}
		firewalls = &fcmv2.FirewallList{}
		mons      = &fcmv2.FirewallMonitorList{}
	)

	err := a.client.List(ctx, firewalls, client.InNamespace(namespace))
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

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create shoot client: %w", err)
	}

	err = shootClient.List(ctx, mons, &client.ListOptions{Namespace: fcmv2.FirewallShootNamespace})
	if err != nil {
		return fmt.Errorf("error listing firewall monitors: %w", err)
	}

	if !everyFirewallHasAMonitor(firewalls, mons) {
		return fmt.Errorf("every firewall needs to have a corresponding firewall monitor before migration, because firewalls are restored from the monitors")
	}

	err = a.migrateRBAC(ctx, shootClient, fwdeploys, namespace)
	if err != nil {
		return err
	}

	a.logger.Info("migrating firewalls", "amount", len(firewalls.Items), "monitors-amount", len(mons.Items))

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

func (a *actuator) migrateRBAC(ctx context.Context, shootClient client.Client, fwdeploys *fcmv2.FirewallDeploymentList, namespace string) error {
	a.logger.Info("copying service account secrets into the shoot", "amount", len(fwdeploys.Items))

	for _, fwdeploy := range fwdeploys.Items {
		saName := fmt.Sprintf("firewall-controller-seed-access-%s", fwdeploy.Name) // TODO: name should be exposed by fcm
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: namespace,
			},
		}

		err := a.client.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		if err != nil {
			return fmt.Errorf("error getting service account: %w", err)
		}

		if len(sa.Secrets) != 1 {
			return fmt.Errorf("firewall service account %q needs to reference exactly one token secret", sa.Name)
		}

		saSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      sa.Secrets[0].Name,
				Namespace: namespace,
			},
		}
		err = a.client.Get(ctx, client.ObjectKeyFromObject(saSecret), saSecret)
		if err != nil {
			return fmt.Errorf("error getting service account secret: %w", err)
		}

		migrationSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: fcmv2.FirewallShootNamespace,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, shootClient, migrationSecret, func() error {
			migrationSecret.Annotations = saSecret.Annotations
			migrationSecret.Labels = saSecret.Labels
			migrationSecret.Labels[migrationSecretKey] = ""
			migrationSecret.Data = saSecret.Data
			migrationSecret.Type = saSecret.Type
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to create / update migration secret: %w", err)
		}
	}

	return nil
}
