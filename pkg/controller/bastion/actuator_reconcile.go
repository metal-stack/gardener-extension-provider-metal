package bastion

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	reconcilerutils "github.com/gardener/gardener/pkg/controllerutils/reconciler"
	"github.com/go-logr/logr"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/machine"
	"github.com/metal-stack/metal-go/api/models"
	"github.com/metal-stack/metal-lib/pkg/tag"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// bastionTag is a metal tag attached to the bastion machine to distinguish it from other machines.
	bastionTag = "metal-stack.io/bastion"
	// Should not have the clusterID tag as otherwise mcm list the bastion as cluster machine which would force the mcm to delete this as a orphan machine
	bastionClusterID = tag.ClusterID + "-bastion"
)

// Reconcile ensures that the bastion machine is allocated in the private node network and all public
// networks of the shoot and publishes the public endpoint in the bastion status.
func (a *actuator) Reconcile(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) error {
	data, err := a.getClusterData(ctx, bastion, cluster)
	if err != nil {
		return err
	}

	privateNetworkID, err := a.getPrivateNetworkID(ctx, data, bastion, cluster)
	if err != nil {
		return err
	}

	opt, err := DetermineOptions(bastion, cluster, data.cloudProfileConfig, data.infrastructureConfig, privateNetworkID, a.machineImageMapping)
	if err != nil {
		return fmt.Errorf("failed to determine bastion options: %w", err)
	}

	status, err := getProviderStatus(bastion)
	if err != nil {
		return err
	}
	if status == nil {
		status = &providerStatusRaw{}
	}
	if status.MachineName == "" {
		status.MachineName = opt.MachineName
	}
	if err := a.updateProviderStatus(ctx, bastion, status); err != nil {
		return fmt.Errorf("failed to store status.providerStatus: %w", err)
	}

	clusterID := string(cluster.Shoot.Status.UID)

	machine, err := findBastionMachine(ctx, data.mclient, opt.Project, opt.MachineName, clusterID)
	if err != nil {
		return err
	}

	if machine == nil {
		userData, err := transpileUserData(bastion.Spec.UserData)
		if err != nil {
			return fmt.Errorf("failed to prepare bastion userdata: %w", err)
		}

		log.Info("Allocating bastion machine", "name", opt.MachineName)
		machine, err = allocateBastionMachine(ctx, data.mclient, bastion, opt, cluster, userData)
		if err != nil {
			return fmt.Errorf("failed to allocate bastion machine: %w", err)
		}
	}

	if id := ptr.Deref(machine.ID, ""); id != "" && status.MachineID != id {
		status.MachineID = id
		if err := a.updateProviderStatus(ctx, bastion, status); err != nil {
			return fmt.Errorf("failed to store status.providerStatus: %w", err)
		}
	}

	ingress := determineIngress(machine, opt)
	if ingress == nil {
		return &reconcilerutils.RequeueAfterError{
			RequeueAfter: 15 * time.Second,
			Cause:        fmt.Errorf("bastion machine has no public ip yet"),
		}
	}

	patch := client.MergeFrom(bastion.DeepCopy())
	bastion.Status.Ingress = ingress
	if err := a.client.Status().Patch(ctx, bastion, patch); err != nil {
		return fmt.Errorf("failed to store status.ingress: %w", err)
	}

	return nil
}

func (a *actuator) updateProviderStatus(ctx context.Context, bastion *extensionsv1alpha1.Bastion, status *providerStatusRaw) error {
	raw, err := marshalProviderStatus(status)
	if err != nil {
		return err
	}

	if bastion.Status.ProviderStatus != nil && bytes.Equal(bastion.Status.ProviderStatus.Raw, raw) {
		return nil
	}

	patch := client.MergeFrom(bastion.DeepCopy())
	bastion.Status.ProviderStatus = &runtime.RawExtension{Raw: raw}
	return a.client.Status().Patch(ctx, bastion, patch)
}

// findBastionMachine returns the bastion machine if it exists, nil otherwise.
func findBastionMachine(ctx context.Context, mclient metalgo.Client, projectID, machineName, clusterID string) (*models.V1MachineResponse, error) {
	mfr := &models.V1MachineFindRequest{
		AllocationProject: projectID,
		AllocationName:    machineName,
		AllocationRole:    models.V1MachineAllocationRoleMachine,
		Tags:              []string{fmt.Sprintf("%s=%s", bastionClusterID, clusterID)},
	}

	resp, err := mclient.Machine().FindMachines(machine.NewFindMachinesParams().WithBody(mfr).WithContext(ctx), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to find bastion machine: %w", err)
	}

	// Filter the results by name as well, so that we do not depend on the metal-api supporting the
	// allocation_name filter.
	var matches []*models.V1MachineResponse
	for _, m := range resp.Payload {
		if m == nil {
			continue
		}
		if m.Allocation != nil && ptr.Deref(m.Allocation.Name, "") == machineName {
			matches = append(matches, m)
		}
	}

	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("found more than one machine for bastion %q", machineName)
	}
}

// allocateBastionMachine allocates a new machine in the private node network and all public networks,
// based on machine-controller-manager-provider-metal.
func allocateBastionMachine(ctx context.Context, mclient metalgo.Client, bastion *extensionsv1alpha1.Bastion, opt *Options, cluster *extensionscontroller.Cluster, userData []byte) (*models.V1MachineResponse, error) {
	networks := make([]*models.V1MachineAllocationNetwork, 0, len(opt.NetworkIDs))
	for _, id := range opt.NetworkIDs {
		if id == "" {
			continue
		}
		networks = append(networks, &models.V1MachineAllocationNetwork{
			Autoacquire: new(true),
			Networkid:   new(id),
		})
	}

	tags := []string{
		fmt.Sprintf("%s=%s", bastionClusterID, string(cluster.Shoot.Status.UID)),
		fmt.Sprintf("%s=%s", tag.ClusterName, bastion.Namespace),
		fmt.Sprintf("%s=%s", tag.ClusterProject, opt.Project),
		fmt.Sprintf("%s=%s", bastionTag, bastion.Name),
	}

	req := &models.V1MachineAllocateRequest{
		Description: fmt.Sprintf("Bastion %s for shoot %s created by Gardener.", bastion.Name, cluster.Shoot.Name),
		Name:        opt.MachineName,
		Hostname:    opt.MachineName,
		UserData:    strings.TrimSpace(string(userData)),
		Sizeid:      new(opt.Size),
		Projectid:   new(opt.Project),
		Partitionid: new(opt.Partition),
		Imageid:     new(opt.Image),
		Networks:    networks,
		Tags:        tags,
	}

	resp, err := mclient.Machine().AllocateMachine(machine.NewAllocateMachineParams().WithBody(req).WithContext(ctx), nil)
	if err != nil {
		return nil, err
	}
	if resp.Payload == nil {
		return nil, fmt.Errorf("metal-api returned no machine for allocation request")
	}

	return resp.Payload, nil
}

// determineIngress picks the public IP of the bastion machine from one of the public networks.
func determineIngress(m *models.V1MachineResponse, opt *Options) *corev1.LoadBalancerIngress {
	if m == nil || m.Allocation == nil {
		return nil
	}

	for _, nw := range m.Allocation.Networks {
		if nw == nil || len(nw.Ips) == 0 {
			continue
		}

		if nw.Networktypev2 == nil || *nw.Networktypev2 != "external" {
			continue
		}

		return &corev1.LoadBalancerIngress{IP: nw.Ips[0]}
	}

	return nil
}
