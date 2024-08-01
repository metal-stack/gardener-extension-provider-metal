package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/metal-stack/metal-lib/pkg/tag"

	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	genericworkeractuator "github.com/gardener/gardener/extensions/pkg/controller/worker/genericactuator"
	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"github.com/metal-stack/gardener-extension-provider-metal/charts"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MutatingWebhookObjectSelectorLabel = "gardener-shoot-namespace"
)

// MachineClassKind yields the name of the machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "MachineClass"
}

// MachineClass yields a newly initialized MachineClass object.
func (w *workerDelegate) MachineClass() client.Object {
	return &machinev1alpha1.MachineClass{}
}

// MachineClassList yields a newly initialized MachineClassList object.
func (w *workerDelegate) MachineClassList() client.ObjectList {
	return &machinev1alpha1.MachineClassList{}
}

// DeployMachineClasses generates and creates the metal specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}

	values := kubernetes.Values(map[string]interface{}{"machineClasses": w.machineClasses})

	return w.seedChartApplier.ApplyFromEmbeddedFS(ctx, charts.InternalChart, filepath.Join("internal", "machineclass"), w.worker.Namespace, "machineclass", values)
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

	infrastructureConfig := &apismetal.InfrastructureConfig{}
	if _, _, err := w.decoder.Decode(w.cluster.Shoot.Spec.Provider.InfrastructureConfig.Raw, nil, infrastructureConfig); err != nil {
		return err
	}

	keepHash := func(deploymentName string) (string, bool, error) {
		if _, ok := w.cluster.Shoot.Annotations["cluster.metal-stack.io/keep-worker-hash"]; !ok {
			return "", false, nil
		}

		classes := &machinev1alpha1.MachineClassList{}
		err := w.client.List(ctx, classes, client.InNamespace(w.worker.Namespace))
		if err != nil {
			return "", false, err
		}

		var hash string
		for _, class := range classes.Items {
			class := class

			_, h, ok := strings.Cut(class.Name, deploymentName+"-")
			if !ok {
				continue
			}
			if len(h) != 5 {
				continue
			}

			hash = h
		}

		if hash == "" {
			w.logger.Info("no machine classes found, allow creation of a new one", "name", deploymentName)
			return "", false, nil
		}

		return hash, true, nil
	}

	for _, pool := range w.worker.Spec.Pools {
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
			metalClusterIDTag      = fmt.Sprintf("%s=%s", tag.ClusterID, w.cluster.Shoot.GetUID())
			metalClusterNameTag    = fmt.Sprintf("%s=%s", tag.ClusterName, w.worker.Namespace)
			metalClusterProjectTag = fmt.Sprintf("%s=%s", tag.ClusterProject, w.additionalData.infrastructureConfig.ProjectID)

			kubernetesClusterTag        = fmt.Sprintf("kubernetes.io/cluster=%s", w.worker.Namespace)
			kubernetesRoleTag           = "kubernetes.io/role=node"
			kubernetesInstanceTypeTag   = fmt.Sprintf("node.kubernetes.io/instance-type=%s", pool.MachineType)
			kubernetesTopologyRegionTag = fmt.Sprintf("topology.kubernetes.io/region=%s", w.worker.Spec.Region)
			kubernetesTopologyZoneTag   = fmt.Sprintf("topology.kubernetes.io/zone=%s", w.additionalData.infrastructureConfig.PartitionID)
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
			"partition": w.additionalData.infrastructureConfig.PartitionID,
			"size":      pool.MachineType,
			"project":   w.additionalData.infrastructureConfig.ProjectID,
			"network":   w.additionalData.privateNetworkID,
			"image":     machineImage,
			"tags":      tags,
			"sshkeys":   []string{string(w.worker.Spec.SSHPublicKey)},
			"secret": map[string]interface{}{
				"cloudConfig": string(pool.UserData),
			},
			"credentialsSecretRef": map[string]interface{}{
				"name":      w.worker.Spec.SecretRef.Name,
				"namespace": w.worker.Spec.SecretRef.Namespace,
			},
		}

		workerPoolHash, err := worker.WorkerPoolHash(pool, w.cluster)
		if err != nil {
			return err
		}

		var (
			deploymentName = fmt.Sprintf("%s-%s", w.worker.Namespace, pool.Name)
		)

		override, ok, err := keepHash(deploymentName)
		if err != nil {
			return err
		}

		if ok {
			w.logger.Info("using existing worker hash", "value", override)
			workerPoolHash = override
		}

		var (
			className = fmt.Sprintf("%s-%s", deploymentName, workerPoolHash)
		)

		machineDeployments = append(machineDeployments, worker.MachineDeployment{
			Name:                 deploymentName,
			ClassName:            className,
			SecretName:           className,
			Minimum:              pool.Minimum,
			Maximum:              pool.Maximum,
			MaxSurge:             pool.MaxSurge,
			MaxUnavailable:       pool.MaxUnavailable,
			Labels:               pool.Labels,
			Annotations:          pool.Annotations,
			Taints:               pool.Taints,
			MachineConfiguration: genericworkeractuator.ReadMachineConfiguration(pool),
		})

		machineClassSpec["name"] = className
		machineClassSpec["labels"] = map[string]string{
			v1beta1constants.GardenerPurpose: v1beta1constants.GardenPurposeMachineClass,
		}

		// if we'd move the endpoint out of this secret into the deployment spec (which would be the way to go)
		// it would roll all worker nodes...
		machineClassSpec["secret"].(map[string]interface{})["metalAPIURL"] = w.additionalData.mcp.Endpoint
		machineClassSpec["secret"].(map[string]interface{})[metal.APIKey] = w.additionalData.credentials.MetalAPIKey
		machineClassSpec["secret"].(map[string]interface{})[metal.APIHMac] = w.additionalData.credentials.MetalAPIHMac

		machineClasses = append(machineClasses, machineClassSpec)
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses

	return nil
}
