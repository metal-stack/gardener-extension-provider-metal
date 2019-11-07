package infrastructure

import (
	"context"
	"fmt"
	"time"

	metalapi "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	metalclient "github.com/metal-pod/gardener-extension-provider-metal/pkg/metal/client"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener-extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

func (a *actuator) delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	infrastructureConfig := &metalapi.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return fmt.Errorf("could not decode provider config: %+v", err)
	}

	mclient, err := metalclient.NewClient(ctx, a.client, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if infrastructure.Status.ProviderStatus != nil {
		if _, _, err := a.decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return fmt.Errorf("could not decode infrastructure status: %+v", err)
		}

		if infrastructureStatus.Firewall.MachineID != "" {
			machineID := decodeMachineID(infrastructureStatus.Firewall.MachineID)
			if machineID != "" {
				_, err = mclient.MachineDelete(machineID)
				if err != nil {
					a.logger.Error(err, "failed to delete firewall", "infrastructure", infrastructure.Name, "firewallID", machineID)
					return &controllererrors.RequeueAfterError{
						Cause:        err,
						RequeueAfter: 30 * time.Second,
					}
				}
			}
		}
	}

	projectID := cluster.Shoot.Spec.Cloud.Metal.ProjectID
	nodeCIDR := cluster.Shoot.Spec.Cloud.Metal.Networks.Nodes
	clusterID := string(cluster.Shoot.ObjectMeta.UID)

	ipsToFree, ipsToUpdate, err := metalclient.GetEphemeralIPsFromCluster(mclient, projectID, clusterID)
	if err != nil {
		a.logger.Error(err, "failed to query ephemeral cluster ips", "infrastructure", infrastructure.Name, "clusterID", clusterID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range ipsToFree {
		_, err := mclient.IPFree(*ip.Ipaddress)
		a.logger.Error(err, "failed to release ephemeral cluster ip", "infrastructure", infrastructure.Name, "ip", *ip.Ipaddress)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range ipsToUpdate {
		err := metalclient.UpdateIPInCluster(mclient, ip, clusterID)
		if err != nil {
			a.logger.Error(err, "failed to remove cluster tags from ip which is member of other clusters", "infrastructure", infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	privateNetworks, err := metalclient.GetPrivateNetworksFromNodeNetwork(mclient, projectID, nodeCIDR)
	if err != nil {
		a.logger.Error(err, "failed to query private network", "infrastructure", infrastructure.Name, "nodeCIDR", nodeCIDR)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, pn := range privateNetworks {
		_, err := mclient.NetworkFree(*pn.ID)
		a.logger.Error(err, "failed to release private network", "infrastructure", infrastructure.Name, "networkID", *pn.ID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	return nil
}
