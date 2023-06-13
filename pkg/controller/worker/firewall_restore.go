package worker

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/metal-go/api/client/firewall"
	"github.com/metal-stack/metal-lib/pkg/pointer"
	"github.com/metal-stack/metal-lib/pkg/tag"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (a *actuator) firewallRestore(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	var (
		namespace = cluster.ObjectMeta.Name

		mons = &fcmv2.FirewallMonitorList{}
	)

	fcm := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "firewall-controller-manager",
			Namespace: namespace,
		},
	}
	err := a.client.Get(ctx, client.ObjectKeyFromObject(fcm), fcm)
	if err != nil {
		return fmt.Errorf("unable to get firewall-controller-manager deployment: %w", err)
	}

	if fcm.Status.Replicas != fcm.Status.ReadyReplicas {
		return fmt.Errorf("firewall-controller-manager deployment is not yet ready, waiting...")
	}

	_, shootClient, err := util.NewClientForShoot(ctx, a.client, namespace, client.Options{})
	if err != nil {
		return fmt.Errorf("unable to create shoot client: %w", err)
	}

	err = shootClient.List(ctx, mons, &client.ListOptions{Namespace: fcmv2.FirewallShootNamespace})
	if err != nil {
		return fmt.Errorf("error listing firewall monitors: %w", err)
	}

	a.logger.Info("restoring firewalls from monitors", "amount", len(mons.Items))

	err = a.restoreFirewalls(ctx, worker, cluster, mons)
	if err != nil {
		return fmt.Errorf("error restoring firewall: %w", err)
	}

	err = a.ensureFirewallDeployment(ctx, worker, cluster)
	if err != nil {
		return err
	}

	err = a.restoreRBAC(ctx, shootClient, namespace)
	if err != nil {
		return err
	}

	return nil
}

func (a *actuator) restoreFirewalls(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster, mons *fcmv2.FirewallMonitorList) error {
	d, err := a.getAdditionalData(ctx, worker, cluster)
	if err != nil {
		return fmt.Errorf("error getting additional data: %w", err)
	}

	fwcv, err := validation.ValidateFirewallControllerVersion(d.mcp.FirewallControllerVersions, d.infrastructureConfig.Firewall.ControllerVersion)
	if err != nil {
		return err
	}

	var (
		namespace = cluster.ObjectMeta.Name
		clusterID = string(cluster.Shoot.GetUID())
	)

	for _, mon := range mons.Items {
		resp, err := d.mclient.Firewall().FindFirewall(firewall.NewFindFirewallParams().WithID(mon.MachineStatus.MachineID).WithContext(ctx), nil)
		if err != nil {
			return fmt.Errorf("error finding firewall: %w", err)
		}

		if pointer.SafeDeref(resp.Payload.Allocation.Project) != d.infrastructureConfig.ProjectID {
			continue
		}

		f := &fcmv2.Firewall{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mon.Name,
				Namespace: namespace,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, a.client, f, func() error {
			if f.Labels == nil {
				f.Labels = map[string]string{}
			}
			f.Labels[tag.ClusterID] = clusterID

			f.Spec = fcmv2.FirewallSpec{
				Size:                    d.infrastructureConfig.Firewall.Size,
				Image:                   d.infrastructureConfig.Firewall.Image,
				Partition:               d.infrastructureConfig.PartitionID,
				Project:                 d.infrastructureConfig.ProjectID,
				Networks:                append(d.infrastructureConfig.Firewall.Networks, d.privateNetworkID),
				Userdata:                resp.Payload.Allocation.UserData,
				SSHPublicKeys:           resp.Payload.Allocation.SSHPubKeys,
				RateLimits:              mapRateLimits(d.infrastructureConfig.Firewall.RateLimits),
				InternalPrefixes:        a.controllerConfig.FirewallInternalPrefixes,
				EgressRules:             mapEgressRules(d.infrastructureConfig.Firewall.EgressRules),
				DryRun:                  false,
				ControllerVersion:       fwcv.Version,
				ControllerURL:           fwcv.URL,
				NftablesExporterVersion: d.mcp.NftablesExporter.Version,
				NftablesExporterURL:     d.mcp.NftablesExporter.URL,
				LogAcceptedConnections:  d.infrastructureConfig.Firewall.LogAcceptedConnections,
				DNSServerAddress:        "",
				DNSPort:                 nil,
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error restoring firewall resource: %w", err)
		}

		a.logger.Info("restored firewall", "name", f.Name, "cluster-id", clusterID)
	}

	return nil
}

func (a *actuator) restoreRBAC(ctx context.Context, shootClient client.Client, namespace string) error {
	migrationSecrets := &corev1.SecretList{}
	err := shootClient.List(ctx, migrationSecrets, &client.ListOptions{Namespace: fcmv2.FirewallShootNamespace}, client.MatchingLabels{
		migrationSecretKey: "",
	})
	if err != nil {
		return fmt.Errorf("error retrieving migration secrets")
	}

	a.logger.Info("copying previous service account secrets into the shoot", "amount", len(migrationSecrets.Items))

	for _, migrationSecret := range migrationSecrets.Items {
		migrationSecret := migrationSecret

		saSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      migrationSecret.Name,
				Namespace: namespace,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, a.client, saSecret, func() error {
			saSecret.Annotations = migrationSecret.Annotations
			saSecret.Labels = migrationSecret.Labels
			saSecret.Data = migrationSecret.Data
			saSecret.Type = migrationSecret.Type
			return nil
		})
		if err != nil {
			return fmt.Errorf("unable to create / update service account secret: %w", err)
		}

		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      migrationSecret.Name,
				Namespace: namespace,
			},
		}

		_, err = controllerutil.CreateOrUpdate(ctx, a.client, sa, func() error {
			sa.Secrets = []corev1.ObjectReference{
				{
					Name: saSecret.Name,
				},
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("error to create / update service account: %w", err)
		}

		err = shootClient.Delete(ctx, &migrationSecret)
		if err != nil {
			return fmt.Errorf("unable to cleanup shoot migration secret: %w", err)
		}

		a.logger.Info("restored service account secret from shoot", "name", migrationSecret.Name, "namespace", migrationSecret.Namespace, "sa", sa.Name)
	}

	return nil
}
