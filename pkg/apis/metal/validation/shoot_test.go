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

package validation_test

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/apis/garden"
	. "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	// . "github.com/gardener/gardener/pkg/utils/validation/gomega"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Shoot validation", func() {
	Describe("#ValidateWorkerConfig", func() {
		var (
			cloudProfile *gardencorev1beta1.CloudProfile
			workers      []garden.Worker
		)

		BeforeEach(func() {
			cloudProfile = &gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					MachineImages: []gardencorev1beta1.MachineImage{
						{
							Name: "ubuntu",
							Versions: []gardencorev1beta1.ExpirableVersion{
								{
									Version: "19.04",
								},
								{
									Version: "19.10",
								},
							},
						},
					},
				},
			}
			workers = []garden.Worker{
				{
					Machine: garden.Machine{
						Type: "c1-xlarge-x86",
						Image: &garden.ShootMachineImage{
							Name:    "ubuntu",
							Version: "19.04",
						},
					},
				},
			}
		})

		It("should pass because workers are configured correctly", func() {
			errorList := ValidateWorkers(workers, cloudProfile, field.NewPath("workers"))

			Expect(errorList).To(BeEmpty())
		})

		It("volume must be nil", func() {
			workers[0].Volume = &garden.Volume{
				Type: strPtr("fancy-storage"),
			}

			errorList := ValidateWorkers(workers, cloudProfile, field.NewPath("workers"))

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("workers[0].volume"),
					"Detail": Equal("volumes are not yet supported and must be nil"),
				})),
			))
		})

		It("zones must be empty", func() {
			workers[0].Zones = []string{"a"}

			errorList := ValidateWorkers(workers, cloudProfile, field.NewPath("workers"))

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("workers[0].zones"),
					"Detail": Equal("zone spreading is not supported, specify partition via infrastructure config"),
				})),
			))
		})

		It("image must be present in cloud profile", func() {
			workers[0].Machine.Image.Name = "coreos"

			errorList := ValidateWorkers(workers, cloudProfile, field.NewPath("workers"))

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("workers[0].machine.image"),
					"Detail": Equal("machine image coreos in version 19.04 not offered, available images are: [ubuntu-19.04 ubuntu-19.10]"),
				})),
			))
		})

		It("image version must be present in cloud profile", func() {
			workers[0].Machine.Image.Version = "1.0"

			errorList := ValidateWorkers(workers, cloudProfile, field.NewPath("workers"))

			Expect(errorList).To(ConsistOf(
				PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("workers[0].machine.image"),
					"Detail": Equal("machine image ubuntu in version 1.0 not offered, available images are: [ubuntu-19.04 ubuntu-19.10]"),
				})),
			))
		})
	})
})

func strPtr(str string) *string {
	return &str
}
