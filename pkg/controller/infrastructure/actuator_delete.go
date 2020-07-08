package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"
	metalgo "github.com/metal-stack/metal-go"

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
	mclient              *metalgo.Driver
	clusterID            string
	clusterTag           string
	machineID            string
	projectID            string
}

func (a *actuator) Delete(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	return Delete(ctx, a.logger, a.restConfig, a.client, a.decoder, infrastructure, cluster)
}

func Delete(
	ctx context.Context,
	logger logr.Logger,
	restConfig *rest.Config,
	c client.Client,
	decoder runtime.Decoder,
	infrastructure *extensionsv1alpha1.Infrastructure,
	cluster *extensionscontroller.Cluster,
) error {
	internalInfrastructureConfig, internalInfrastructureStatus, err := decodeInfrastructure(infrastructure, decoder)
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

	mclient, err := metalclient.NewClient(ctx, c, metalControlPlane.Endpoint, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	deleter := &firewallDeleter{
		logger:               logger,
		c:                    c,
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
		err := deleteFirewall(d.logger, d.machineID, d.projectID, d.clusterTag, d.mclient)
		if err != nil {
			return err
		}

		d.providerStatus.Firewall.MachineID = ""
		err = updateProviderStatus(ctx, d.c, d.infrastructure, d.providerStatus, d.infrastructure.Status.NodesCIDR)
		if err != nil {
			return err
		}

	}

	ipsToFree, ipsToUpdate, err := metalclient.GetEphemeralIPsFromCluster(d.mclient, d.projectID, d.clusterID)
	if err != nil {
		d.logger.Error(err, "failed to query ephemeral cluster ips", "infrastructure", d.infrastructure.Name, "clusterID", d.clusterID)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	for _, ip := range ipsToFree {
		_, err := d.mclient.IPFree(*ip.Ipaddress)
		if err != nil {
			d.logger.Error(err, "failed to release ephemeral cluster ip", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	for _, ip := range ipsToUpdate {
		err := metalclient.UpdateIPInCluster(d.mclient, ip, d.clusterID)
		if err != nil {
			d.logger.Error(err, "failed to remove cluster tags from ip which is member of other clusters", "infrastructure", d.infrastructure.Name, "ip", *ip.Ipaddress)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}
	}

	if d.infrastructure.Status.NodesCIDR != nil {
		privateNetworks, err := metalclient.GetPrivateNetworksFromNodeNetwork(d.mclient, d.projectID, *d.infrastructure.Status.NodesCIDR)
		if err != nil {
			d.logger.Error(err, "failed to query private network", "infrastructure", d.infrastructure.Name, "nodeCIDR", *d.infrastructure.Status.NodesCIDR)
			return &controllererrors.RequeueAfterError{
				Cause:        err,
				RequeueAfter: 30 * time.Second,
			}
		}

		for _, pn := range privateNetworks {
			_, err := d.mclient.NetworkFree(*pn.ID)
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

func deleteFirewall(logger logr.Logger, machineID string, projectID string, clusterTag string, mclient *metalgo.Driver) error {
	firewalls, err := metalclient.FindClusterFirewalls(mclient, clusterTag, projectID)
	if err != nil {
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	if len(firewalls) > 0 {
		if *firewalls[0].ID != machineID {
			return fmt.Errorf("firewall from provider status does not match actual cluster firewall, can't do anything")
		}

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
