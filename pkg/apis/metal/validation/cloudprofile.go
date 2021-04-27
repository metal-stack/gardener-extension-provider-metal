// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package validation

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(cloudProfileConfig *apismetal.CloudProfileConfig, cloudProfile *core.CloudProfile, providerConfigPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableZones := sets.NewString()
	for _, region := range cloudProfile.Spec.Regions {
		for _, zone := range region.Zones {
			availableZones.Insert(zone.Name)
		}
	}

	controlPlanesPath := providerConfigPath.Child("metalControlPlanes")
	for mcpName, mcp := range cloudProfileConfig.MetalControlPlanes {
		for partitionName := range mcp.Partitions {
			if !availableZones.Has(partitionName) {
				allErrs = append(allErrs, field.Invalid(controlPlanesPath.Child(mcpName), partitionName, fmt.Sprintf("the control plane has a partition that is not a configured zone in any of the cloud profile regions: %v", availableZones.List())))
			}
		}
	}

	return allErrs
}
