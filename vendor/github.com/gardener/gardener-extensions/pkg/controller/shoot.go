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

package controller

import (
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
)

// GetPodNetwork returns the pod network CIDR of the given Shoot.
func GetPodNetwork(shoot *gardenv1beta1.Shoot) gardencorev1alpha1.CIDR {
	cloud := shoot.Spec.Cloud
	switch {
	case cloud.AWS != nil:
		return *cloud.AWS.Networks.K8SNetworks.Pods
	case cloud.Azure != nil:
		return *cloud.Azure.Networks.K8SNetworks.Pods
	case cloud.GCP != nil:
		return *cloud.GCP.Networks.K8SNetworks.Pods
	case cloud.OpenStack != nil:
		return *cloud.OpenStack.Networks.K8SNetworks.Pods
	case cloud.Alicloud != nil:
		return *cloud.Alicloud.Networks.K8SNetworks.Pods
	case cloud.Packet != nil:
		return *cloud.Packet.Networks.K8SNetworks.Pods
	default:
		return ""
	}
}

// IsHibernated returns true if the shoot is hibernated, or false otherwise.
func IsHibernated(shoot *gardenv1beta1.Shoot) bool {
	return shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled
}

// GetReplicas returns the woken up replicas of the given Shoot.
func GetReplicas(shoot *gardenv1beta1.Shoot, wokenUp int) int {
	if IsHibernated(shoot) {
		return 0
	}
	return wokenUp
}

// GetControlPlaneReplicas returns the woken up replicas for controlplane components of the given Shoot
// that should only be scaled down at the end of the flow.
func GetControlPlaneReplicas(shoot *gardenv1beta1.Shoot, scaledDown bool, wokenUp int) int {
	if IsHibernated(shoot) && scaledDown {
		return 0
	}
	return wokenUp
}
