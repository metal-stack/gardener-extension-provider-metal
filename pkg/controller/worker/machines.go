// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	"github.com/gardener/gardener-extensions/pkg/util"
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"

	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	metalclient "github.com/metal-pod/gardener-extension-provider-metal/pkg/metal/client"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

// MachineClassKind yields the name of the metal machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "MetalMachineClass"
}

// MachineClassList yields a newly initialized MetalMachineClassList object.
func (w *workerDelegate) MachineClassList() runtime.Object {
	return &machinev1alpha1.MetalMachineClassList{}
}

// DeployMachineClasses generates and creates the metal specific machine classes.
func (w *workerDelegate) DeployMachineClasses(ctx context.Context) error {
	if w.machineClasses == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return err
		}
	}
	return w.seedChartApplier.ApplyChart(ctx, filepath.Join(metal.InternalChartsPath, "machineclass"), w.worker.Namespace, "machineclass", map[string]interface{}{"machineClasses": w.machineClasses}, nil)
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

	shootVersionMajorMinor, err := util.VersionMajorMinor(w.cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	credentials, err := metalclient.ReadCredentialsFromSecretRef(ctx, w.client, &w.worker.Spec.SecretRef)
	if err != nil {
		return err
	}

	mclient, err := metalclient.NewClientFromCredentials(credentials)
	if err != nil {
		return err
	}

	projectID := infrastructureConfig.ProjectID
	nodeCIDR := w.cluster.Shoot.Spec.Networking.Nodes

	privateNetwork, err := metalclient.GetPrivateNetworkFromNodeNetwork(mclient, projectID, nodeCIDR)
	if err != nil {
		return err
	}

	clusterID := w.cluster.Shoot.GetUID()
	clusterTag := fmt.Sprintf("%s=%s", metal.ShootAnnotationClusterID, clusterID)
	regionTag := fmt.Sprintf("topology.kubernetes.io/region=%s", w.worker.Spec.Region)
	zoneTag := fmt.Sprintf("topology.kubernetes.io/zone=%s", w.worker.Name)

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

		machineClassSpec := map[string]interface{}{
			"partition": infrastructureConfig.PartitionID,
			"size":      pool.MachineType,
			"project":   projectID,
			"network":   privateNetwork.ID,
			"image":     machineImage,
			"tags": []string{
				fmt.Sprintf("kubernetes.io/cluster=%s", w.worker.Namespace),
				"kubernetes.io/role=node",
				regionTag,
				zoneTag,
				fmt.Sprintf("node.kubernetes.io/instance-type=%s", pool.MachineType),
				clusterTag,
				fmt.Sprintf("machine.metal-pod.io/project-id=%s", projectID),
			},
			"sshkeys": []string{string(w.worker.Spec.SSHPublicKey)},
			"secret": map[string]interface{}{
				"cloudConfig": string(pool.UserData),
			},
		}

		var (
			machineClassSpecHash = worker.MachineClassHash(machineClassSpec, shootVersionMajorMinor)
			deploymentName       = fmt.Sprintf("%s-%s-z", w.worker.Namespace, pool.Name)
			className            = fmt.Sprintf("%s-%s", deploymentName, machineClassSpecHash)
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
		machineClassSpec["secret"].(map[string]interface{})[metal.APIURL] = credentials.MetalAPIURL
		machineClassSpec["secret"].(map[string]interface{})[metal.APIKey] = credentials.MetalAPIKey
		machineClassSpec["secret"].(map[string]interface{})[metal.APIHMac] = credentials.MetalAPIHMac

		machineClasses = append(machineClasses, machineClassSpec)
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses

	return nil
}
