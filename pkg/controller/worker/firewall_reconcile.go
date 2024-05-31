package worker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/go-logr/logr"
	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/metal-stack/metal-lib/pkg/tag"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (a *actuator) firewallReconcile(ctx context.Context, log logr.Logger, worker *extensionsv1alpha1.Worker, cluster *extensionscontroller.Cluster) error {
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

	sshSecret, err := helper.GetLatestSSHSecret(ctx, a.client, worker.Namespace)
	if err != nil {
		return fmt.Errorf("could not find current ssh secret: %w", err)
	}

	d, err := a.getAdditionalData(ctx, worker, cluster)
	if err != nil {
		return fmt.Errorf("error getting additional data: %w", err)
	}

	err = a.ensureFirewallDeployment(ctx, log, d, cluster, string(sshSecret.Data["id_rsa.pub"]))
	if err != nil {
		return err
	}

	err = a.updateState(ctx, log, d.infrastructure)
	if err != nil {
		return fmt.Errorf("unable to update firewall state: %w", err)
	}

	return nil
}

func (a *actuator) ensureFirewallDeployment(ctx context.Context, log logr.Logger, d *additionalData, cluster *extensionscontroller.Cluster, sshKey string) error {
	var (
		clusterID = string(cluster.Shoot.GetUID())
		namespace = cluster.ObjectMeta.Name
	)

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

	controlPlaneConfig, err := helper.ControlPlaneConfigFromClusterShootSpec(cluster)
	if err != nil {
		return err
	}

	networkAccessType := apismetal.NetworkAccessBaseline
	if controlPlaneConfig.NetworkAccessType != nil {
		networkAccessType = *controlPlaneConfig.NetworkAccessType
	}

	_, err = controllerutil.CreateOrUpdate(ctx, a.client, deploy, func() error {
		if deploy.Annotations == nil {
			deploy.Annotations = map[string]string{}
		}
		deploy.Annotations[fcmv2.ReconcileAnnotation] = strconv.FormatBool(true)

		if deploy.Labels == nil {
			deploy.Labels = map[string]string{}
		}
		// this is the selector for the mutating webhook, without it the mutation will not happen
		deploy.Labels[MutatingWebhookObjectSelectorLabel] = cluster.ObjectMeta.Name

		_ = controllerutil.AddFinalizer(deploy, fcmv2.FinalizerName)

		deploy.Spec.Replicas = 1
		deploy.Spec.AutoUpdate.MachineImage = d.infrastructureConfig.Firewall.AutoUpdateMachineImage

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
		deploy.Spec.Template.Spec.InternalPrefixes = a.controllerConfig.FirewallInternalPrefixes
		deploy.Spec.Template.Spec.EgressRules = mapEgressRules(d.infrastructureConfig.Firewall.EgressRules)
		deploy.Spec.Template.Spec.ControllerVersion = fwcv.Version
		deploy.Spec.Template.Spec.ControllerURL = fwcv.URL
		deploy.Spec.Template.Spec.NftablesExporterVersion = d.mcp.NftablesExporter.Version
		deploy.Spec.Template.Spec.NftablesExporterURL = d.mcp.NftablesExporter.URL
		deploy.Spec.Template.Spec.LogAcceptedConnections = d.infrastructureConfig.Firewall.LogAcceptedConnections
		deploy.Spec.Template.Spec.SSHPublicKeys = []string{sshKey}

		if d.partition.NetworkIsolation != nil &&
			len(d.partition.NetworkIsolation.DNSServers) > 0 &&
			networkAccessType != apismetal.NetworkAccessBaseline {
			dnsAddr, portStr, ok := strings.Cut(d.partition.NetworkIsolation.DNSServers[0], ":")
			deploy.Spec.Template.Spec.DNSServerAddress = dnsAddr

			if ok {
				p, err := strconv.ParseUint(portStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid dns port:%q", portStr)
				}
				port := uint(p)
				deploy.Spec.Template.Spec.DNSPort = &port
			}
		} else {
			deploy.Spec.Template.Spec.DNSServerAddress = ""
		}

		if networkAccessType == apismetal.NetworkAccessForbidden {
			if d.partition.NetworkIsolation == nil || len(d.partition.NetworkIsolation.AllowedNetworks.Egress) == 0 {
				// we need at least some egress rules to reach our own registry etcpp, so no single egress rule MUST be an error
				return fmt.Errorf("error creating firewall deployment: control plane with network access forbidden requires partition %q to have networkIsolation.allowedNetworks", d.infrastructureConfig.PartitionID)
			}
			deploy.Spec.Template.Spec.AllowedNetworks = fcmv2.AllowedNetworks{
				Ingress: d.partition.NetworkIsolation.AllowedNetworks.Ingress,
				Egress:  d.partition.NetworkIsolation.AllowedNetworks.Egress,
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating firewall deployment: %w", err)
	}

	log.Info("reconciled firewall deployment", "name", deploy.Name, "cluster-id", clusterID)

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
