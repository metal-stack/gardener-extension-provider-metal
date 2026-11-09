package bastion

import (
	"context"
	"fmt"
	"time"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	reconcilerutils "github.com/gardener/gardener/pkg/controllerutils/reconciler"
	"github.com/go-logr/logr"
	metalgo "github.com/metal-stack/metal-go"
	"github.com/metal-stack/metal-go/api/client/machine"
	"k8s.io/utils/ptr"
)

// Delete frees the bastion machine and waits until it is gone.
func (a *actuator) Delete(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) error {
	data, err := a.getClusterData(ctx, bastion, cluster)
	if err != nil {
		return err
	}

	machineName, err := bastionMachineName(bastion, cluster)
	if err != nil {
		return err
	}

	clusterID := string(cluster.Shoot.GetUID())

	m, err := findBastionMachine(ctx, data.mclient, data.projectID, machineName, clusterID)
	if err != nil {
		return err
	}

	if m == nil {
		return nil
	}

	id := ptr.Deref(m.ID, "")
	if id == "" {
		return nil
	}

	log.Info("Freeing bastion machine", "name", machineName, "id", id)
	if _, err := data.mclient.Machine().FreeMachine(machine.NewFreeMachineParams().WithID(id).WithContext(ctx), nil); err != nil {
		log.Error(err, "Failed to free bastion machine, will retry", "id", id)
	}

	deleted, err := isBastionMachineDeleted(ctx, data.mclient, id)
	if err != nil {
		return err
	}
	if !deleted {
		return &reconcilerutils.RequeueAfterError{
			RequeueAfter: 30 * time.Second,
			Cause:        fmt.Errorf("bastion machine %q is still deleting", id),
		}
	}

	return nil
}

// ForceDelete tries to free the bastion machine on a best-effort basis and does not wait for its deletion.
func (a *actuator) ForceDelete(ctx context.Context, log logr.Logger, bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) error {
	data, err := a.getClusterData(ctx, bastion, cluster)
	if err != nil {
		log.Error(err, "Failed to determine cluster data for force delete")
		return nil
	}

	machineName, err := bastionMachineName(bastion, cluster)
	if err != nil {
		log.Error(err, "Failed to determine bastion machine name for force delete")
		return nil
	}

	clusterID := string(cluster.Shoot.GetUID())

	m, err := findBastionMachine(ctx, data.mclient, data.projectID, machineName, clusterID)
	if err != nil || m == nil {
		if err != nil {
			log.Error(err, "Failed to find bastion machine for force delete")
		}
		return nil
	}

	id := ptr.Deref(m.ID, "")
	if id == "" {
		return nil
	}

	if _, err := data.mclient.Machine().FreeMachine(machine.NewFreeMachineParams().WithID(id).WithContext(ctx), nil); err != nil {
		log.Error(err, "Failed to free bastion machine during force delete", "id", id)
	}

	return nil
}

// isBastionMachineDeleted returns true if the machine is gone or its allocation was released.
func isBastionMachineDeleted(ctx context.Context, mclient metalgo.Client, id string) (bool, error) {
	resp, err := mclient.Machine().FindMachine(machine.NewFindMachineParams().WithID(id).WithContext(ctx), nil)
	if err != nil {
		// the machine no longer exists
		return true, nil
	}
	if resp.Payload == nil || resp.Payload.Allocation == nil {
		return true, nil
	}

	return false, nil
}

// bastionMachineName resolves the machine name of a bastion. It prefers the name stored in the provider
// status and falls back to the deterministic name.
func bastionMachineName(bastion *extensionsv1alpha1.Bastion, cluster *extensionscontroller.Cluster) (string, error) {
	status, err := getProviderStatus(bastion)
	if err != nil {
		return "", err
	}
	if status != nil && status.MachineName != "" {
		return status.MachineName, nil
	}

	return generateBastionBaseResourceName(cluster.ObjectMeta.Name, bastion.Name)
}
