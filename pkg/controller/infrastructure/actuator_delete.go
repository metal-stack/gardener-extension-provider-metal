package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"
	metalip "github.com/metal-stack/metal-go/api/client/ip"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/pkg/controllerutils/reconciler"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

type networkDeleter struct {
	ctx                  context.Context
	logger               logr.Logger
	infrastructure       *extensionsv1alpha1.Infrastructure
	infrastructureConfig *metalapi.InfrastructureConfig
	mclient              metalgo.Client
	clusterID            string
	cluster              *extensionscontroller.Cluster
}

func (a *actuator) Delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	internalInfrastructureConfig, _, err := decodeInfrastructure(infrastructure, a.decoder)
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

	deleter := &networkDeleter{
		ctx:                  ctx,
		logger:               a.logger,
		infrastructure:       infrastructure,
		infrastructureConfig: internalInfrastructureConfig,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		cluster:              cluster,
	}

	err = a.releaseNetworkResources(deleter)
	if err != nil {
		return &reconciler.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	return nil
}

func (a *actuator) releaseNetworkResources(d *networkDeleter) error {
	ipsToFree, ipsToUpdate, err := metalclient.GetEphemeralIPsFromCluster(d.ctx, d.mclient, d.infrastructureConfig.ProjectID, d.clusterID)
	if err != nil {
		a.logger.Error(err, "failed to query ephemeral cluster ips", "infrastructure", d.infrastructure.Name, "clusterID", d.clusterID)
		return err
	}

	for _, ip := range ipsToFree {
		_, err := d.mclient.IP().FreeIP(metalip.NewFreeIPParams().WithID(*ip.Ipaddress).WithContext(d.ctx), nil)
		if err != nil {
			d.logger.Error(err, "failed to release ephemeral cluster ip", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return err
		}
	}

	for _, ip := range ipsToUpdate {
		err := metalclient.UpdateIPInCluster(d.ctx, d.mclient, ip, d.clusterID)
		if err != nil {
			d.logger.Error(err, "failed to remove cluster tags from ip which is member of other clusters", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return err
		}
	}

	resp, err := d.mclient.IP().FindIPs(metalip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
		Projectid: d.infrastructureConfig.ProjectID,
		Tags:      []string{egressTag(d.clusterID)},
		Type:      models.V1IPBaseTypeStatic,
	}).WithContext(d.ctx), nil)
	if err != nil {
		return fmt.Errorf("failed to list egress ips of cluster %w", err)
	}

	for _, ip := range resp.Payload {
		if err := clearIPTags(d.ctx, d.mclient, *ip.Ipaddress); err != nil {
			return fmt.Errorf("could not remove egress tag from ip %s %w", *ip.Ipaddress, err)
		}
	}

	nodeCIDR, err := helper.GetNodeCIDR(d.infrastructure, d.cluster)
	if err != nil {
		privateNetworks, err := metalclient.GetPrivateNetworksFromNodeNetwork(d.ctx, d.mclient, d.infrastructureConfig.ProjectID, nodeCIDR)
		if err != nil {
			d.logger.Error(err, "failed to query private network", "infrastructure", d.infrastructure.Name, "nodeCIDR", nodeCIDR)
			return err
		}

		for _, pn := range privateNetworks {
			_, err := d.mclient.Network().FreeNetwork(network.NewFreeNetworkParams().WithID(*pn.ID).WithContext(d.ctx), nil)
			if err != nil {
				d.logger.Error(err, "failed to release private network", "infrastructure", d.infrastructure.Name, "networkID", *pn.ID)
				return err
			}
		}
	}

	return nil
}
