package infrastructure

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal-stack/metal-lib/pkg/tag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/ip"
	metalip "github.com/metal-stack/metal-go/api/client/ip"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/controllerutils/reconciler"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

type networkReconciler struct {
	logger               logr.Logger
	infrastructure       *extensionsv1alpha1.Infrastructure
	infrastructureConfig *metalapi.InfrastructureConfig
	cluster              *extensionscontroller.Cluster
	mclient              metalgo.Client
	clusterID            string
}

type egressIPReconciler struct {
	logger               logr.Logger
	infrastructureConfig *metalapi.InfrastructureConfig
	mclient              metalgo.Client
	clusterID            string
	egressTag            string
}

func (a *actuator) Reconcile(ctx context.Context, logger logr.Logger, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	internalInfrastructureConfig, internalInfrastructureStatus, err := decodeInfrastructure(infrastructure, a.decoder)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, internalInfrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	mclient, err := metalclient.NewClient(ctx, a.client, metalControlPlane.Endpoint, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	networkReconciler := &networkReconciler{
		logger:               logger,
		infrastructure:       infrastructure,
		infrastructureConfig: internalInfrastructureConfig,
		cluster:              cluster,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
	}
	nodeCIDR, err := ensureNodeNetwork(ctx, networkReconciler)
	if err != nil {
		return &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	err = updateProviderStatus(ctx, a.client, infrastructure, internalInfrastructureStatus, &nodeCIDR)
	if err != nil {
		return err
	}

	egressIPReconciler := &egressIPReconciler{
		logger:               logger,
		infrastructureConfig: internalInfrastructureConfig,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		egressTag:            egressTag(string(cluster.Shoot.GetUID())),
	}
	err = reconcileEgressIPs(ctx, egressIPReconciler)
	if err != nil {
		return &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	err = a.maintainFirewallDeployment(ctx, logger, infrastructure.Namespace)
	if err != nil {
		return err
	}

	return nil
}

func (a *actuator) maintainFirewallDeployment(ctx context.Context, logger logr.Logger, namespace string) error {
	// we need to run the following code from the infrastructure controller because we know
	// that the gardener-controller-manager reconciles the infrastructure resource only in maintenance mode.
	// a controller has no possibility to find out by itself if a reconciliation was triggered from the maintenance controller
	// so it cannot be put to the worker controller.

	deploy := &fcmv2.FirewallDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metal.FirewallDeploymentName,
			Namespace: namespace,
		},
	}

	err := a.client.Get(ctx, client.ObjectKeyFromObject(deploy), deploy)
	if err != nil && apierrors.IsNotFound(err) {
		logger.Info("not maintaining firewall deployment as resource is unpresent")
		return nil
	}

	if err != nil {
		return err
	}

	deploy.Annotations[fcmv2.MaintenanceAnnotation] = strconv.FormatBool(true)

	err = a.client.Update(ctx, deploy)
	if err != nil {
		return fmt.Errorf("unable to trigger firewall deployment maintenance reconciliation %w", err)
	}

	return nil
}

func reconcileEgressIPs(ctx context.Context, r *egressIPReconciler) error {
	currentEgressIPs := sets.NewString()

	resp, err := r.mclient.IP().FindIPs(ip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
		Projectid: r.infrastructureConfig.ProjectID,
		Tags:      []string{r.egressTag},
		Type:      models.V1IPBaseTypeStatic,
	}).WithContext(ctx), nil)
	if err != nil {
		return fmt.Errorf("failed to list egress ips of cluster %w", err)
	}

	for _, ip := range resp.Payload {
		currentEgressIPs.Insert(*ip.Ipaddress)
	}

	wantEgressIPs := sets.NewString()
	for _, egressRule := range r.infrastructureConfig.Firewall.EgressRules {
		wantEgressIPs.Insert(egressRule.IPs...)

		for _, ip := range egressRule.IPs {
			ip := ip
			if currentEgressIPs.Has(ip) {
				continue
			}

			resp, err := r.mclient.IP().FindIPs(metalip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
				Ipaddress: ip,
				Projectid: r.infrastructureConfig.ProjectID,
				Networkid: egressRule.NetworkID,
			}).WithContext(ctx), nil)
			if err != nil {
				return fmt.Errorf("error when retrieving ip %s for egress rule %w", ip, err)
			}

			switch len(resp.Payload) {
			case 0:
				return fmt.Errorf("ip %s for egress rule does not exist", ip)
			case 1:
			default:
				return fmt.Errorf("ip %s found multiple times", ip)
			}

			dbIP := resp.Payload[0]
			if dbIP.Type != nil && *dbIP.Type != models.V1IPBaseTypeStatic {
				return fmt.Errorf("ips for egress rule must be static, but %s is not static", ip)
			}

			if len(dbIP.Tags) > 0 {
				return fmt.Errorf("won't use ip %s for egress rules because it does not have an egress tag but it has other tags", *dbIP.Ipaddress)
			}

			_, err = r.mclient.IP().UpdateIP(metalip.NewUpdateIPParams().WithBody(&models.V1IPUpdateRequest{
				Ipaddress: dbIP.Ipaddress,
				Tags:      []string{r.egressTag},
			}).WithContext(ctx), nil)
			if err != nil {
				return fmt.Errorf("could not tag ip %s for egress usage %w", ip, err)
			}
		}
	}

	if !currentEgressIPs.Equal(wantEgressIPs) {
		toUnTag := currentEgressIPs.Difference(wantEgressIPs)
		for _, ip := range toUnTag.List() {
			err := clearIPTags(ctx, r.mclient, ip)
			if err != nil {
				return fmt.Errorf("could not remove egress tag from ip %s %w", ip, err)
			}
		}
	}

	return nil
}

func egressTag(clusterID string) string {
	return fmt.Sprintf("%s=%s", tag.ClusterEgress, clusterID)
}

func clearIPTags(ctx context.Context, mclient metalgo.Client, ip string) error {
	_, err := mclient.IP().UpdateIP(metalip.NewUpdateIPParams().WithBody(&models.V1IPUpdateRequest{
		Ipaddress: &ip,
		Tags:      []string{},
	}).WithContext(ctx), nil)

	return err
}

func ensureNodeNetwork(ctx context.Context, r *networkReconciler) (string, error) {
	if r.cluster.Shoot.Spec.Networking != nil && r.cluster.Shoot.Spec.Networking.Nodes != nil {
		return *r.cluster.Shoot.Spec.Networking.Nodes, nil
	}

	if r.infrastructure.Status.NodesCIDR != nil {
		resp, err := r.mclient.Network().FindNetworks(network.NewFindNetworksParams().WithBody(&models.V1NetworkFindRequest{
			Projectid:   r.infrastructureConfig.ProjectID,
			Partitionid: r.infrastructureConfig.PartitionID,
			Labels:      map[string]string{tag.ClusterID: r.clusterID},
		}).WithContext(ctx), nil)
		if err != nil {
			return "", err
		}

		if len(resp.Payload) != 0 {
			return *r.infrastructure.Status.NodesCIDR, nil
		}

		return "", fmt.Errorf("node network disappeared from cloud provider: %s", *r.infrastructure.Status.NodesCIDR)
	}

	resp, err := r.mclient.Network().AllocateNetwork(network.NewAllocateNetworkParams().WithBody(&models.V1NetworkAllocateRequest{
		Projectid:   r.infrastructureConfig.ProjectID,
		Partitionid: r.infrastructureConfig.PartitionID,
		Name:        r.cluster.Shoot.GetName(),
		Description: r.clusterID,
		Labels:      map[string]string{tag.ClusterID: r.clusterID},
	}).WithContext(ctx), nil)
	if err != nil {
		return "", err
	}

	nodeCIDR := resp.Payload.Prefixes[0]

	return nodeCIDR, nil
}
