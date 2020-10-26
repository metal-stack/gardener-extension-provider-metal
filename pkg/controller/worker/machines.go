package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	metaltag "github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/controllerutils"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	controllererrors "github.com/gardener/gardener/extensions/pkg/controller/error"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal/client"

	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"
	metalv1alpha1 "github.com/metal-stack/machine-controller-manager-provider-metal/pkg/provider/migration/legacy-api/machine/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineClassKind yields the name of the metal machine class.
func (w *workerDelegate) MachineClassKind() string {
	ootDeployment, err := w.isOOTDeployment()
	if err != nil {
		w.logger.Error(err, "could not determine if oot control plane feature gate is set, defaulting to old MCM deployment type")
	}
	if ootDeployment {
		return "MachineClass"
	}
	return "MetalMachineClass"
}

// MachineClassList yields a newly initialized MetalMachineClassList object.
func (w *workerDelegate) MachineClassList() runtime.Object {
	ootDeployment, err := w.isOOTDeployment()
	if err != nil {
		w.logger.Error(err, "could not determine if oot control plane feature gate is set, defaulting to old MCM deployment type")
	}
	if ootDeployment {
		return &machinev1alpha1.MachineClassList{}
	}
	return &metalv1alpha1.MetalMachineClassList{}
}

func (w *workerDelegate) cleanupOldMachineClasses(ctx context.Context, namespace string, machineClassList runtime.Object, wantedMachineDeployments worker.MachineDeployments) error {
	if err := w.client.List(ctx, machineClassList, client.InNamespace(namespace)); err != nil {
		return err
	}

	return meta.EachListItem(machineClassList, func(machineClass runtime.Object) error {
		var (
			// we cannot take the finalizer name directly from the machine-controller-manager dependency because the binary would not build anymore
			deleteFinalizerName = "machine.sapcloud.io/machine-controller-manager"
		)

		oldClass := machineClass.(*metalv1alpha1.MetalMachineClass)

		// we can already set a deletion timestamp
		if err := w.client.Delete(ctx, machineClass); err != nil {
			return err
		}

		// the machine controller manager does not know the metalmachineclass and cannot do the cleanup after migration
		// therefore, we need to check if the migration was done and then remove the resources by removing the finalizer from our end
		if controllerutils.HasFinalizer(oldClass, deleteFinalizerName) {
			var newClass machinev1alpha1.MachineClass
			err := w.client.Get(ctx, types.NamespacedName{Name: oldClass.Name, Namespace: oldClass.Namespace}, &newClass)
			if err != nil {
				w.logger.Info("cannot remove old metal machine classes by now, new class not yet created")
				return &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}

			machines := &machinev1alpha1.MachineList{}
			if err := w.client.List(ctx, machines, client.InNamespace(namespace)); err != nil {
				return err
			}
			err = meta.EachListItem(machines, func(machineObject runtime.Object) error {
				machine := machineObject.(*machinev1alpha1.Machine)
				w.logger.Info("checking if machine was migrated", "machine", machine.Name)
				if machine.Spec.Class.Kind != "MachineClass" {
					return fmt.Errorf("cannot remove old metal machine classes by now, machine not yet migrated to new machine class")
				}
				return nil
			})
			if err != nil {
				w.logger.Info(err.Error())
				return &controllererrors.RequeueAfterError{
					Cause:        err,
					RequeueAfter: 30 * time.Second,
				}
			}

			w.logger.Info("all machines were migrated to new machine class, removing finalizer from old machine class...")

			err = controllerutils.RemoveFinalizer(ctx, w.client, oldClass, deleteFinalizerName)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// DeployMachineClasses generates and creates the metal specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}

	ootDeployment, err := w.isOOTDeployment()
	if err != nil {
		return err
	}

	if ootDeployment {
		// Delete any older version machine class CRs.
		if err := w.cleanupOldMachineClasses(ctx, w.worker.Namespace, &metalv1alpha1.MetalMachineClassList{}, nil); err != nil {
			w.logger.Info("unable to cleanup old metal machine classes by now, retrying later...", "error", err)
		}
	} else {
		err := w.errorWhenAlreadyMigrated(ctx)
		if err != nil {
			return err
		}
	}

	values := kubernetes.Values(map[string]interface{}{"machineClasses": w.machineClasses, "deployOOT": ootDeployment})

	return w.seedChartApplier.Apply(ctx, filepath.Join(metal.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", values)
}

func (w *workerDelegate) errorWhenAlreadyMigrated(ctx context.Context) error {
	machines := &machinev1alpha1.MachineList{}
	if err := w.client.List(ctx, machines, client.InNamespace(w.worker.Namespace)); err != nil {
		return err
	}
	return meta.EachListItem(machines, func(machineObject runtime.Object) error {
		machine := machineObject.(*machinev1alpha1.Machine)
		if machine.Spec.Class.Kind == "MachineClass" {
			return fmt.Errorf("cannot use legacy deployment method of MCM because machines were already migrated (at least partly), please enable worker feature gate for MCM OOT")
		}
		return nil
	})
}

// GenerateMachineDeployments generates the configuration for the desired machine deployments.
func (w *workerDelegate) GenerateMachineDeployments(ctx context.Context) (worker.MachineDeployments, error) {
	if w.machineDeployments == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return nil, err
		}
	}
	return w.machineDeployments, nil
}

func (w *workerDelegate) generateMachineConfig(ctx context.Context) error {
	var (
		machineDeployments = worker.MachineDeployments{}
		machineClasses     []map[string]interface{}
		machineImages      []apismetal.MachineImage
	)

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(w.cluster)
	if err != nil {
		return err
	}

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := w.decoder.Decode(w.cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return err
	}

	metalControlPlane, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	credentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, w.client, &w.worker.Spec.SecretRef)
	if err != nil {
		return err
	}

	mclient, err := metalclient.NewClientFromCredentials(metalControlPlane.Endpoint, credentials)
	if err != nil {
		return err
	}

	// TODO: this is a workaround to speed things for the time being...
	// the infrastructure controller writes the nodes cidr back into the infrastructure status, but the cluster resource does not contain it immediately
	// it would need the start of another reconcilation until the node cidr can be picked up from the cluster resource
	// therefore, we read it directly from the infrastructure status
	infrastructure := &extensionsv1alpha1.Infrastructure{}
	if err := w.client.Get(ctx, kutil.Key(w.worker.Namespace, w.cluster.Shoot.Name), infrastructure); err != nil {
		return err
	}

	projectID := infrastructureConfig.ProjectID
	nodeCIDR := infrastructure.Status.NodesCIDR

	if nodeCIDR == nil {
		return fmt.Errorf("nodeCIDR was not yet set by infrastructure controller")
	}

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, *nodeCIDR)
	if err != nil {
		return err
	}

	for _, pool := range w.worker.Spec.Pools {
		workerPoolHash, err := worker.WorkerPoolHash(pool, w.cluster)
		if err != nil {
			return err
		}

		machineImage, err := w.findMachineImage(pool.MachineImage.Name, pool.MachineImage.Version)
		if err != nil {
			return err
		}
		machineImages = appendMachineImage(machineImages, apismetal.MachineImage{
			Name:    pool.MachineImage.Name,
			Version: pool.MachineImage.Version,
			Image:   machineImage,
		})

		var (
			metalClusterIDTag      = fmt.Sprintf("%s=%s", metaltag.ClusterID, w.cluster.Shoot.GetUID())
			metalClusterNameTag    = fmt.Sprintf("%s=%s", metaltag.ClusterName, w.worker.Namespace)
			metalClusterProjectTag = fmt.Sprintf("%s=%s", metaltag.ClusterProject, infrastructureConfig.ProjectID)

			kubernetesClusterTag        = fmt.Sprintf("kubernetes.io/cluster=%s", w.worker.Namespace)
			kubernetesRoleTag           = "kubernetes.io/role=node"
			kubernetesInstanceTypeTag   = fmt.Sprintf("node.kubernetes.io/instance-type=%s", pool.MachineType)
			kubernetesTopologyRegionTag = fmt.Sprintf("topology.kubernetes.io/region=%s", w.worker.Spec.Region)
			kubernetesTopologyZoneTag   = fmt.Sprintf("topology.kubernetes.io/zone=%s", infrastructureConfig.PartitionID)
		)

		tags := []string{
			kubernetesClusterTag,
			kubernetesRoleTag,
			kubernetesInstanceTypeTag,
			kubernetesTopologyRegionTag,
			kubernetesTopologyZoneTag,

			metalClusterIDTag,
			metalClusterNameTag,
			metalClusterProjectTag,
		}

		for k, v := range pool.Labels {
			tags = append(tags, fmt.Sprintf("%s=%s", k, v))
		}

		machineClassSpec := map[string]interface{}{
			"partition": infrastructureConfig.PartitionID,
			"size":      pool.MachineType,
			"project":   projectID,
			"network":   privateNetwork.ID,
			"image":     machineImage,
			"tags":      tags,
			"sshkeys":   []string{string(w.worker.Spec.SSHPublicKey)},
			"secret": map[string]interface{}{
				"cloudConfig": string(pool.UserData),
			},
		}

		var (
			deploymentName = fmt.Sprintf("%s-%s", w.worker.Namespace, pool.Name)
			className      = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
		)

		machineDeployments = append(machineDeployments, worker.MachineDeployment{
			Name:           deploymentName,
			ClassName:      className,
			SecretName:     className,
			Minimum:        pool.Minimum,
			Maximum:        pool.Maximum,
			MaxSurge:       pool.MaxSurge,
			MaxUnavailable: pool.MaxUnavailable,
			Labels:         pool.Labels,
			Annotations:    pool.Annotations,
			Taints:         pool.Taints,
		})

		machineClassSpec["name"] = className
		machineClassSpec["labels"] = map[string]string{
			v1beta1constants.GardenerPurpose: genericworkeractuator.GardenPurposeMachineClass,
		}

		// if we'd move the endpoint out of this secret into the deployment spec (which would be the way to go)
		// it would roll all worker nodes...
		machineClassSpec["secret"].(map[string]interface{})["metalAPIURL"] = metalControlPlane.Endpoint
		machineClassSpec["secret"].(map[string]interface{})[metal.APIKey] = credentials.MetalAPIKey
		machineClassSpec["secret"].(map[string]interface{})[metal.APIHMac] = credentials.MetalAPIHMac

		machineClasses = append(machineClasses, machineClassSpec)
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses

	return nil
}
