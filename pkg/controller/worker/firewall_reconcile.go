package worker

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/metal-stack/metal-lib/pkg/tag"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (a *actuator) firewallReconcile(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	if worker.DeletionTimestamp != nil {
		return nil
	}
	if extensionscontroller.IsHibernated(cluster) {
		return nil
	}

	name := "firewall-controller-manager-" + cluster.ObjectMeta.Name
	mwc := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := a.client.Get(ctx, client.ObjectKeyFromObject(mwc), mwc)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("mutating webhook configuration %q of firewall-controller-manager is not yet present, requeuing", name)
		}

		return err
	}

	err = a.ensureFirewallDeployment(ctx, worker, cluster)
	if err != nil {
		return err
	}

	return nil
}

func (a *actuator) ensureFirewallDeployment(ctx context.Context, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
	var (
		clusterID = string(cluster.Shoot.GetUID())
		namespace = cluster.ObjectMeta.Name
	)

	d, err := a.getAdditionalData(ctx, worker, cluster)
	if err != nil {
		return fmt.Errorf("error getting additional data: %w", err)
	}

	internalPrefixes := []string{}
	if a.controllerConfig.AccountingExporter.Enabled && a.controllerConfig.AccountingExporter.NetworkTraffic.Enabled {
		internalPrefixes = a.controllerConfig.AccountingExporter.NetworkTraffic.InternalNetworks
	}

	fwcv, err := validation.ValidateFirewallControllerVersion(d.mcp.FirewallControllerVersions, d.infrastructureConfig.Firewall.ControllerVersion)
	if err != nil {
		return err
	}

	deploy := &fcmv2.FirewallDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metal.FirewallDeploymentName,
			Namespace: namespace,
		},
		Spec: fcmv2.FirewallDeploymentSpec{
			Template: fcmv2.FirewallTemplateSpec{
				Spec: fcmv2.FirewallSpec{
					Partition: d.infrastructureConfig.PartitionID,
					Project:   d.infrastructureConfig.ProjectID,
				},
			},
		},
	}

	_, err = controllerutil.CreateOrUpdate(ctx, a.client, deploy, func() error {
		if deploy.Labels == nil {
			deploy.Labels = map[string]string{}
		}
		// this is the selector for the mutating webhook, without it the mutation will not happen
		deploy.Labels[MutatingWebhookObjectSelectorLabel] = cluster.ObjectMeta.Name

		deploy.Spec.Replicas = 1

		// we explicitly set the selector as otherwise firewall migration does not match, which should be prevented
		deploy.Spec.Selector = map[string]string{
			tag.ClusterID: clusterID,
		}

		if deploy.Spec.Template.Labels == nil {
			deploy.Spec.Template.Labels = map[string]string{}
		}
		deploy.Spec.Template.Labels[tag.ClusterID] = clusterID

		deploy.Spec.Template.Spec.Size = d.infrastructureConfig.Firewall.Size
		deploy.Spec.Template.Spec.Image = d.infrastructureConfig.Firewall.Image
		deploy.Spec.Template.Spec.Networks = append(d.infrastructureConfig.Firewall.Networks, d.privateNetworkID)
		deploy.Spec.Template.Spec.RateLimits = mapRateLimits(d.infrastructureConfig.Firewall.RateLimits)
		deploy.Spec.Template.Spec.InternalPrefixes = internalPrefixes
		deploy.Spec.Template.Spec.EgressRules = mapEgressRules(d.infrastructureConfig.Firewall.EgressRules)
		deploy.Spec.Template.Spec.ControllerVersion = fwcv.Version
		deploy.Spec.Template.Spec.ControllerURL = fwcv.URL
		deploy.Spec.Template.Spec.NftablesExporterVersion = d.mcp.NftablesExporter.Version
		deploy.Spec.Template.Spec.NftablesExporterURL = d.mcp.NftablesExporter.URL
		deploy.Spec.Template.Spec.LogAcceptedConnections = d.infrastructureConfig.Firewall.LogAcceptedConnections

		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating firewall deployment: %w", err)
	}

	a.logger.Info("reconciled firewall deployment", "name", deploy.Name, "cluster-id", clusterID)

	return nil
}

func mapRateLimits(limits []apismetal.RateLimit) []fcmv2.RateLimit {
	var result []fcmv2.RateLimit
	for _, l := range limits {
		result = append(result, fcmv2.RateLimit{
			NetworkID: l.NetworkID,
			Rate:      l.RateLimit,
		})
	}
	return result
}

func mapEgressRules(egress []apismetal.EgressRule) []fcmv2.EgressRuleSNAT {
	var result []fcmv2.EgressRuleSNAT
	for _, rule := range egress {
		rule := rule
		result = append(result, fcmv2.EgressRuleSNAT{
			NetworkID: rule.NetworkID,
			IPs:       rule.IPs,
		})
	}
	return result
}
