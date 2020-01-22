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

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateWorkers validates the workers of a Shoot.
func ValidateWorkers(workers []garden.Worker, cloudProfile *gardencorev1beta1.CloudProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableImages := sets.NewString()
	for _, image := range cloudProfile.Spec.MachineImages {
		for _, version := range image.Versions {
			availableImages.Insert(image.Name + "-" + version.Version)
		}
	}

	for i, worker := range workers {
		if worker.Volume != nil {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("volume"), "volumes are not yet supported and must be nil"))
		}

		if len(worker.Zones) != 0 {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("zones"), "zone spreading is not supported, specify partition via infrastructure config"))
		}

		wantedImage := worker.Machine.Image.Name + "-" + worker.Machine.Image.Version
		if !availableImages.Has(wantedImage) {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("machine.image"), fmt.Sprintf("machine image %s in version %s not offered, available images are: %v", worker.Machine.Image.Name, worker.Machine.Image.Version, availableImages.UnsortedList())))
		}
	}

	return allErrs
}
