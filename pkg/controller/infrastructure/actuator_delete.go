package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"
	metalip "github.com/metal-stack/metal-go/api/client/ip"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/client/network"
	"github.com/metal-stack/metal-go/api/models"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener/extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

type firewallDeleter struct {
	ctx                  context.Context
	logger               logr.Logger
	c                    client.Client
	infrastructure       *extensionsv1alpha1.Infrastructure
	infrastructureConfig *metalapi.InfrastructureConfig
	providerStatus       *metalapi.InfrastructureStatus
	cluster              *extensionscontroller.Cluster
	mclient              metalgo.Client
	clusterID            string
	clusterTag           string
	machineID            string
	projectID            string
}

func (a *actuator) Delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
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

	deleter := &firewallDeleter{
		ctx:                  ctx,
		logger:               a.logger,
		c:                    a.client,
		infrastructure:       infrastructure,
		infrastructureConfig: internalInfrastructureConfig,
		providerStatus:       internalInfrastructureStatus,
		cluster:              cluster,
		mclient:              mclient,
		clusterID:            string(cluster.Shoot.GetUID()),
		clusterTag:           ClusterTag(string(cluster.Shoot.GetUID())),
		machineID:            decodeMachineID(internalInfrastructureStatus.Firewall.MachineID),
		projectID:            internalInfrastructureConfig.ProjectID,
	}

	return delete(ctx, deleter)
}

func delete(ctx context.Context, d *firewallDeleter) error {
	if d.machineID != "" {
		err := deleteFirewall(ctx, d.machineID, d.projectID, d.clusterTag, d.mclient)
		if err != nil {
			return err
		}
		d.logger.Info("firewall deleted", "clusterTag", d.clusterTag, "machineid", d.machineID)

		d.providerStatus.Firewall.MachineID = ""
		err = updateProviderStatus(ctx, d.c, d.infrastructure, d.providerStatus, d.infrastructure.Status.NodesCIDR)
		if err != nil {
			return err
		}
	}

	ipsToFree, ipsToUpdate, err := metalclient.GetEphemeralIPsFromCluster(ctx, d.mclient, d.projectID, d.clusterID)
	if err != nil {
		d.logger.Error(err, "failed to query ephemeral cluster ips", "infrastructure", d.infrastructure.Name, "clusterID", d.clusterID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range ipsToFree {
		_, err := d.mclient.IP().FreeIP(metalip.NewFreeIPParams().WithID(*ip.Ipaddress).WithContext(ctx), nil)
		if err != nil {
			d.logger.Error(err, "failed to release ephemeral cluster ip", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	for _, ip := range ipsToUpdate {
		err := metalclient.UpdateIPInCluster(ctx, d.mclient, ip, d.clusterID)
		if err != nil {
			d.logger.Error(err, "failed to remove cluster tags from ip which is member of other clusters", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	resp, err := d.mclient.IP().FindIPs(metalip.NewFindIPsParams().WithBody(&models.V1IPFindRequest{
		Projectid: d.projectID,
		Tags:      []string{egressTag(d.clusterID)},
		Type:      models.V1IPBaseTypeEphemeral,
	}).WithContext(ctx), nil)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        fmt.Errorf("failed to list egress ips of cluster %w", err),
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range resp.Payload {
		if err := clearIPTags(ctx, d.mclient, *ip.Ipaddress); err != nil {
			return &controllererrors.RequeueAfterError{
				Cause:        fmt.Errorf("could not remove egress tag from ip %s %w", *ip.Ipaddress, err),
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	if d.infrastructure.Status.NodesCIDR != nil {
		privateNetworks, err := metalclient.GetPrivateNetworksFromNodeNetwork(ctx, d.mclient, d.projectID, *d.infrastructure.Status.NodesCIDR)
		if err != nil {
			d.logger.Error(err, "failed to query private network", "infrastructure", d.infrastructure.Name, "nodeCIDR", *d.infrastructure.Status.NodesCIDR)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}

		for _, pn := range privateNetworks {
			_, err := d.mclient.Network().FreeNetwork(network.NewFreeNetworkParams().WithID(*pn.ID).WithContext(ctx), nil)
			if err != nil {
				d.logger.Error(err, "failed to release private network", "infrastructure", d.infrastructure.Name, "networkID", *pn.ID)
				return &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}
		}
	}

	return nil
}

func deleteFirewall(ctx context.Context, machineID string, projectID string, clusterTag string, mclient metalgo.Client) error {
	firewalls, err := metalclient.FindClusterFirewalls(ctx, mclient, clusterTag, projectID)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	switch len(firewalls) {
	case 0:
		return nil
	case 1:
		actualID := *firewalls[0].ID
		if actualID != machineID {
			return fmt.Errorf("firewall from provider status does not match actual cluster firewall, can't do anything")
		}

		_, err = mclient.Machine().FreeMachine(machine.NewFreeMachineParams().WithID(machineID).WithContext(ctx), nil)
		if err != nil {
			return &controllererrors.RequeueAfterError{
				Cause:        fmt.Errorf("failed to delete firewall %w", err),
				RequeueAfter: 30 * time.Second,
			}
		}
		return nil
	default:
		return fmt.Errorf("multiple firewalls exist for this cluster, which should not happen. please delete firewalls manually.")
	}
}
