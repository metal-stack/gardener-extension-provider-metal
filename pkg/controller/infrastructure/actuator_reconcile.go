package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	metalapi "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	metalclient "github.com/metal-pod/gardener-extension-provider-metal/pkg/metal/client"

	metalgo "github.com/metal-pod/metal-go"
	metalfirewall "github.com/metal-pod/metal-go/api/client/firewall"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	controllererrors "github.com/gardener/gardener-extensions/pkg/controller/error"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
)

func (a *actuator) reconcile(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	infrastructureConfig := &metalapi.InfrastructureConfig{}
	if _, _, err := a.decoder.Decode(infrastructure.Spec.ProviderConfig.Raw, nil, infrastructureConfig); err != nil {
		return fmt.Errorf("could not decode provider config: %+v", err)
	}

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if infrastructure.Status.ProviderStatus != nil {
		if _, _, err := a.decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return fmt.Errorf("could not decode infrastructure status: %+v", err)
		}
	}

	firewallStatus := infrastructureStatus.Firewall
	if firewallStatus.Succeeded {
		return nil
	}

	mclient, err := metalclient.NewClient(ctx, a.client, &infrastructure.Spec.SecretRef)
	if err != nil {
		return err
	}

	if firewallStatus.MachineID != "" {
		// firewall was already created
		machineID := decodeMachineID(firewallStatus.MachineID)

		resp, err := mclient.FirewallGet(machineID)
		if err != nil {
			switch e := err.(type) {
			case *metalfirewall.FindFirewallDefault:
				if e.Code() >= 500 {
					return &controllererrors.RequeueAfterError{
						Cause:        e,
						RequeueAfter: 30 * time.Second,
					}
				}
			default:
				return e
			}
		}

		allocation := resp.Firewall.Allocation
		if allocation == nil {
			return fmt.Errorf("firewall %q was created but has no allocation", machineID)
		}

		firewallStatus.Succeeded = *resp.Firewall.Allocation.Succeeded
		return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus)
	}

	// we need to create a firewall
	uuid, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	// Example values:
	// cluster.Shoot.Status.TechnicalID  "shoot--dev--johndoe-metal"
	tid := cluster.Shoot.Status.TechnicalID
	name := tid + "-firewall-" + uuid.String()[:5]

	// find private network
	projectID := cluster.Shoot.Spec.Cloud.Metal.ProjectID
	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, cluster.Shoot.Spec.Cloud.Metal.Networks.Nodes)
	if err != nil {
		return err
	}

	// assemble firewall allocation request
	var networks []metalgo.MachineAllocationNetwork
	network := metalgo.MachineAllocationNetwork{
		NetworkID:   *privateNetwork.ID,
		Autoacquire: true,
	}
	networks = append(networks, network)
	for _, n := range infrastructureConfig.Firewall.Networks {
		network := metalgo.MachineAllocationNetwork{
			NetworkID:   n,
			Autoacquire: true,
		}
		networks = append(networks, network)
	}

	createRequest := &metalgo.FirewallCreateRequest{
		MachineCreateRequest: metalgo.MachineCreateRequest{
			Description:   name + " created by Gardener",
			Name:          name,
			Hostname:      name,
			Size:          infrastructureConfig.Firewall.Size,
			Project:       projectID,
			Partition:     infrastructureConfig.Firewall.Partition,
			Image:         infrastructureConfig.Firewall.Image,
			SSHPublicKeys: []string{string(infrastructure.Spec.SSHPublicKey)},
			Networks:      networks,
		},
	}

	a.logger.Info("create firewall from", "request", createRequest)

	fcr, err := mclient.FirewallCreate(createRequest)
	if err != nil {
		a.logger.Error(err, "failed to create firewall", "infrastructure", infrastructure.Name)
		return &controllererrors.RequeueAfterError{
			Cause:        err,
			RequeueAfter: 30 * time.Second,
		}
	}

	machineID := encodeMachineID(*fcr.Firewall.Partition.ID, *fcr.Firewall.ID)

	allocation := fcr.Firewall.Allocation
	if allocation == nil {
		return fmt.Errorf("firewall %q was created but has no allocation", machineID)
	}

	firewallStatus.MachineID = machineID
	firewallStatus.Succeeded = true

	return a.updateProviderStatus(ctx, infrastructure, infrastructureConfig, firewallStatus)
}

func (a *actuator) updateProviderStatus(ctx context.Context, infrastructure *extensionsv1alpha1.Infrastructure, infrastructureConfig *metalapi.InfrastructureConfig, status metalapi.FirewallStatus) error {
	return extensionscontroller.TryUpdateStatus(ctx, retry.DefaultBackoff, a.client, infrastructure, func() error {
		infrastructure.Status.ProviderStatus = &runtime.RawExtension{
			Object: &metalapi.InfrastructureStatus{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metalapi.SchemeGroupVersion.String(),
					Kind:       "InfrastructureStatus",
				},
				Firewall: status,
			},
		}
		return nil
	})
}
