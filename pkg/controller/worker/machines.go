package worker

import (
	"context"
	"fmt"
	"path/filepath"

	metaltag "github.com/metal-stack/metal-lib/pkg/tag"
	"github.com/pkg/errors"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MachineClassKind yields the name of the metal machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "MachineClass"
}

// MachineClassList yields a newly initialized MetalMachineClassList object.
func (w *workerDelegate) MachineClassList() runtime.Object {
	return &machinev1alpha1.MachineClassList{}
}

// cleanupOldMachineClasses sets deletionTimestamp any older version machine class CRs.
func (w *workerDelegate) cleanupOldMachineClasses(ctx context.Context, namespace string, machineClassList runtime.Object, wantedMachineDeployments worker.MachineDeployments) error {
	if err := w.client.List(ctx, machineClassList, client.InNamespace(namespace)); err != nil {
		return err
	}

	return meta.EachListItem(machineClassList, func(machineClass runtime.Object) error {
		if err := w.client.Delete(ctx, machineClass); err != nil {
			return err
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

	// Delete any older version machine class CRs.
	if err := w.cleanupOldMachineClasses(ctx, w.worker.Namespace, &metalv1alpha1.MetalMachineClassList{}, nil); err != nil {
		return errors.Wrapf(err, "cleaning up older version of metal machine class CRs failed")
	}

	return w.seedChartApplier.Apply(ctx, filepath.Join(metal.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", kubernetes.Values(map[string]interface{}{"machineClasses": w.machineClasses}))
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

		machineClassSpec := map[string]interface{}{
			"partition": infrastructureConfig.PartitionID,
			"size":      pool.MachineType,
			"project":   projectID,
			"network":   privateNetwork.ID,
			"image":     machineImage,
			"tags": []string{
				kubernetesClusterTag,
				kubernetesRoleTag,
				kubernetesInstanceTypeTag,
				kubernetesTopologyRegionTag,
				kubernetesTopologyZoneTag,

				metalClusterIDTag,
				metalClusterNameTag,
				metalClusterProjectTag,
			},
			"sshkeys": []string{string(w.worker.Spec.SSHPublicKey)},
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
