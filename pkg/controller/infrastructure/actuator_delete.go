package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/metal-stack/metal-lib/pkg/tag"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener/extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

func (a *actuator) Delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	internalInfrastructureConfig, internalInfrastructureStatus, err := a.decodeInfrastructure(infrastructure)
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

	return delete(ctx,
		a.logger,
		a.client,
		infrastructure,
		internalInfrastructureConfig,
		internalInfrastructureStatus,
		cluster,
		mclient,
	)
}

func delete(
	ctx context.Context,
	logger logr.Logger,
	c client.Client,
	infrastructure *extensionsv1alpha1.Infrastructure,
	infrastructureConfig *metalapi.InfrastructureConfig,
	providerStatus *metalapi.InfrastructureStatus,
	cluster *extensionscontroller.Cluster,
	mclient *metalgo.Driver,
) error {
	var (
		clusterID  = string(cluster.Shoot.GetUID())
		clusterTag = fmt.Sprintf("%s=%s", tag.ClusterID, clusterID)
		machineID  = decodeMachineID(providerStatus.Firewall.MachineID)
		projectID  = infrastructureConfig.ProjectID
	)

	if machineID != "" {
		err := deleteFirewall(logger, machineID, projectID, clusterTag, mclient)
		if err != nil {
			return err
		}

		providerStatus.Firewall.MachineID = ""
		err = updateProviderStatus(ctx, c, infrastructure, providerStatus, infrastructure.Status.NodesCIDR)
		if err != nil {
			return err
		}

	}

	ipsToFree, ipsToUpdate, err := metalclient.GetEphemeralIPsFromCluster(mclient, projectID, clusterID)
	if err != nil {
		logger.Error(err, "failed to query ephemeral cluster ips", "infrastructure", infrastructure.Name, "clusterID", clusterID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range ipsToFree {
		_, err := mclient.IPFree(*ip.Ipaddress)
		if err != nil {
			logger.Error(err, "failed to release ephemeral cluster ip", "infrastructure", infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	for _, ip := range ipsToUpdate {
		err := metalclient.UpdateIPInCluster(mclient, ip, clusterID)
		if err != nil {
			logger.Error(err, "failed to remove cluster tags from ip which is member of other clusters", "infrastructure", infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	if infrastructure.Status.NodesCIDR != nil {
		privateNetworks, err := metalclient.GetPrivateNetworksFromNodeNetwork(mclient, projectID, *infrastructure.Status.NodesCIDR)
		if err != nil {
			logger.Error(err, "failed to query private network", "infrastructure", infrastructure.Name, "nodeCIDR", *infrastructure.Status.NodesCIDR)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}

		for _, pn := range privateNetworks {
			_, err := mclient.NetworkFree(*pn.ID)
			if err != nil {
				logger.Error(err, "failed to release private network", "infrastructure", infrastructure.Name, "networkID", *pn.ID)
				return &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}
		}
	}

	return nil
}

func deleteFirewall(logger logr.Logger, machineID string, projectID string, clusterTag string, mclient *metalgo.Driver) error {
	resp, err := mclient.FirewallFind(&metalgo.FirewallFindRequest{
		MachineFindRequest: metalgo.MachineFindRequest{
			ID:                &machineID,
			AllocationProject: &projectID,
			Tags:              []string{clusterTag},
		},
	})
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	if len(resp.Firewalls) > 0 {
		logger.Info("deleting firewall", "clusterTag", clusterTag, "machineid", machineID)
		_, err = mclient.MachineDelete(machineID)
		if err != nil {
			return &controllererrors.RequeueAfterError{
				Cause:        errors.Wrap(err, "failed to delete firewall"),
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	return nil
}
