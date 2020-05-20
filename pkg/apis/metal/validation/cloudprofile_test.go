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

package validation_test

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/gardener/gardener/pkg/utils/validation/gomega"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var (
			cloudProfile       *gardencorev1beta1.CloudProfile
			cloudProfileConfig *apismetal.CloudProfileConfig
		)

		BeforeEach(func() {
			cloudProfile = &gardencorev1beta1.CloudProfile{
				Spec: gardencorev1beta1.CloudProfileSpec{
					Regions: []gardencorev1beta1.Region{
						{
							Name: "region-a",
							Zones: []gardencorev1beta1.AvailabilityZone{
								{
									Name: "partition-a",
								},
								{
									Name: "partition-b",
								},
							},
						},
						{
							Name: "region-b",
							Zones: []gardencorev1beta1.AvailabilityZone{
								{
									Name: "partition-c",
								},
							},
						},
					},
				},
			}

			cloudProfileConfig = &apismetal.CloudProfileConfig{}
		})

		It("should pass empty configuration", func() {
			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile)

			Expect(errorList).To(BeEmpty())
		})

		It("should pass properly configured control plane partitions", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"partition-b": {},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile)

			Expect(errorList).To(BeEmpty())
		})

		It("should prevent declaring partitions that are not configured in zones", func() {
			cloudProfileConfig.MetalControlPlanes = map[string]apismetal.MetalControlPlane{
				"prod": {
					Partitions: map[string]apismetal.Partition{
						"random-partition": {},
					},
				},
			}

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile)

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("metalControlPlanes.prod"),
				"BadValue": Equal("random-partition"),
				"Detail":   Equal("the control plane has a partition that is not a configured zone in any of the cloud profile regions: [partition-a partition-b partition-c]"),
			}))
		})
	})
})
