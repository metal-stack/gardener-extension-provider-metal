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

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	"github.com/gardener/gardener-extensions/pkg/util"
	confighelper "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/config/helper"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	metalgo "github.com/metal-pod/metal-go"

	machinev1alpha1 "github.com/gardener/machine-controller-manager/pkg/apis/machine/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
)

// MachineClassKind yields the name of the AWS machine class.
func (w *workerDelegate) MachineClassKind() string {
	return "MetalMachineClass"
}

// MachineClassList yields a newly initialized AWSMachineClassList object.
func (w *workerDelegate) MachineClassList() runtime.Object {
	return &machinev1alpha1.MetalMachineClassList{}
}

// DeployMachineClasses generates and creates the AWS specific machine classes.
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

func (w *workerDelegate) generateMachineClassSecretData(ctx context.Context) (map[string][]byte, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, w.client, &w.worker.Spec.SecretRef)
	if err != nil {
		return nil, err
	}

	credentials, err := metal.ReadCredentialsSecret(secret)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		machinev1alpha1.MetalAPIURL:  credentials.MetalAPIURL,
		machinev1alpha1.MetalAPIKey:  credentials.MetalAPIKey,
		machinev1alpha1.MetalAPIHMac: credentials.MetalAPIHMac,
	}, nil
}

func (w *workerDelegate) generateMachineConfig(ctx context.Context) error {
	var (
		machineDeployments = worker.MachineDeployments{}
		machineClasses     []map[string]interface{}
	)

	machineClassSecretData, err := w.generateMachineClassSecretData(ctx)
	if err != nil {
		return err
	}

	shootVersionMajorMinor, err := util.VersionMajorMinor(w.cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return err
	}

	url := string(machineClassSecretData[machinev1alpha1.MetalAPIURL])
	token := string(machineClassSecretData[machinev1alpha1.MetalAPIKey])
	hmac := string(machineClassSecretData[machinev1alpha1.MetalAPIHMac])

	svc, err := metalgo.NewDriver(url, token, hmac)
	if err != nil {
		return err
	}

	// find private network
	// TODO: Can we pass through the private network ID from the infrastructure actuator?
	projectID := w.cluster.Shoot.Spec.Cloud.Metal.ProjectID
	nodeCIDR := *w.cluster.Shoot.Spec.Cloud.Metal.Networks.Nodes
	networkFindRequest := metalgo.NetworkFindRequest{
		ProjectID: &projectID,
		Prefixes:  []string{string(nodeCIDR)},
	}
	networkFindResponse, err := svc.NetworkFind(&networkFindRequest)
	if err != nil {
		return err
	}
	if len(networkFindResponse.Networks) != 1 {
		return fmt.Errorf("no distinct private network for project id %q found: %s", projectID, nodeCIDR)
	}
	privateNetwork := networkFindResponse.Networks[0]

	for _, pool := range w.worker.Spec.Pools {
		zoneLen := len(pool.Zones)

		imageID, err := confighelper.FindImageID(w.machineImages, pool.MachineImage.Name, pool.MachineImage.Version)
		if err != nil {
			return err
		}

		for zoneIndex, zone := range pool.Zones {
			machineClassSpec := map[string]interface{}{
				"partition": zone,
				"size":      pool.MachineType,
				"project":   projectID,
				"network":   privateNetwork.ID,
				"image":     imageID,
				"tags": []string{
					fmt.Sprintf("kubernetes.io/cluster/%s", w.worker.Namespace),
					"kubernetes.io/role/node",
				},
				"sshkeys": []string{string(w.worker.Spec.SSHPublicKey)},
				"secret": map[string]interface{}{
					"cloudConfig": string(pool.UserData),
				},
			}

			var (
				machineClassSpecHash = worker.MachineClassHash(machineClassSpec, shootVersionMajorMinor)
				deploymentName       = fmt.Sprintf("%s-%s-z%d", w.worker.Namespace, pool.Name, zoneIndex+1)
				className            = fmt.Sprintf("%s-%s", deploymentName, machineClassSpecHash)
			)

			machineDeployments = append(machineDeployments, worker.MachineDeployment{
				Name:           deploymentName,
				ClassName:      className,
				SecretName:     className,
				Minimum:        worker.DistributeOverZones(zoneIndex, pool.Minimum, zoneLen),
				Maximum:        worker.DistributeOverZones(zoneIndex, pool.Maximum, zoneLen),
				MaxSurge:       worker.DistributePositiveIntOrPercent(zoneIndex, pool.MaxSurge, zoneLen, pool.Maximum),
				MaxUnavailable: worker.DistributePositiveIntOrPercent(zoneIndex, pool.MaxUnavailable, zoneLen, pool.Minimum),
				Labels:         pool.Labels,
				Annotations:    pool.Annotations,
				Taints:         pool.Taints,
			})

			machineClassSpec["name"] = className
			machineClassSpec["secret"].(map[string]interface{})[metal.APIKey] = string(machineClassSecretData[machinev1alpha1.MetalAPIKey])
			machineClassSpec["secret"].(map[string]interface{})[metal.APIHMac] = string(machineClassSecretData[machinev1alpha1.MetalAPIHMac])
			machineClassSpec["secret"].(map[string]interface{})[metal.APIURL] = string(machineClassSecretData[machinev1alpha1.MetalAPIURL])

			machineClasses = append(machineClasses, machineClassSpec)
		}
	}

	w.machineDeployments = machineDeployments
	w.machineClasses = machineClasses

	return nil
}
