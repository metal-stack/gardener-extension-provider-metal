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

package validation

import (
	"fmt"
	"reflect"
	"sort"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden"
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"

	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateInfrastructureConfigAgainstCloudProfile validates the given `InfrastructureConfig` against the given `CloudProfile`.
func ValidateInfrastructureConfigAgainstCloudProfile(infra *apismetal.InfrastructureConfig, shoot *garden.Shoot, cloudProfile *gardencorev1beta1.CloudProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	shootRegion := shoot.Spec.Region
	for _, region := range cloudProfile.Spec.Regions {
		if region.Name == shootRegion {
			allErrs = append(allErrs, validateInfrastructureConfigZones(infra, region.Zones, fldPath)...)
			break
		}
	}

	return allErrs
}

// validateInfrastructureConfigZones validates the given `InfrastructureConfig` against the given `Zones`.
func validateInfrastructureConfigZones(infra *apismetal.InfrastructureConfig, zones []gardencorev1beta1.AvailabilityZone, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableZones := sets.NewString()
	for _, zone := range zones {
		availableZones.Insert(zone.Name)
	}

	if !availableZones.Has(infra.PartitionID) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("partitionID"), infra.PartitionID, fmt.Sprintf("supported values: %v", availableZones.UnsortedList())))
	}

	return allErrs
}

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apismetal.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	if infra.ProjectID == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("projectID"), "projectID must be specified"))
	}
	if infra.PartitionID == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("partitionID"), "partitionID must be specified"))
	}

	firewallPath := field.NewPath("firewall")
	if infra.Firewall.Image == "" {
		allErrs = append(allErrs, field.Required(firewallPath.Child("image"), "firewall image must be specified"))
	}
	if infra.Firewall.Size == "" {
		allErrs = append(allErrs, field.Required(firewallPath.Child("size"), "firewall size must be specified"))
	}
	for i, network := range infra.Firewall.Networks {
		if network == "" {
			allErrs = append(allErrs, field.Required(firewallPath.Child("networks").Index(i), "firewall network must not be an empty string"))
		}
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apismetal.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.ProjectID, oldConfig.ProjectID, field.NewPath("projectID"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.PartitionID, oldConfig.PartitionID, field.NewPath("partitionID"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.Firewall.Image, oldConfig.Firewall.Image, field.NewPath("firewall.image"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.Firewall.Size, oldConfig.Firewall.Size, field.NewPath("firewall.size"))...)

	var oldNetworks []string
	for _, network := range oldConfig.Firewall.Networks {
		oldNetworks = append(oldNetworks, network)
	}
	var newNetworks []string
	for _, network := range newConfig.Firewall.Networks {
		newNetworks = append(newNetworks, network)
	}

	sort.Strings(oldNetworks)
	sort.Strings(newNetworks)

	if !reflect.DeepEqual(oldNetworks, newNetworks) {
		allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.Firewall.Networks, oldConfig.Firewall.Networks, field.NewPath("firewall.networks"))...)
	}

	return allErrs
}
