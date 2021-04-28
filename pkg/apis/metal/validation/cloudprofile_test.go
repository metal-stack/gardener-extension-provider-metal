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
	"github.com/gardener/gardener/pkg/apis/core"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("CloudProfileConfig validation", func() {
	Describe("#ValidateCloudProfileConfig", func() {
		var (
			cloudProfile       *core.CloudProfile
			cloudProfileConfig *apismetal.CloudProfileConfig
			path               *field.Path
		)

		BeforeEach(func() {
			cloudProfile = &core.CloudProfile{
				Spec: core.CloudProfileSpec{
					Regions: []core.Region{
						{
							Name: "region-a",
							Zones: []core.AvailabilityZone{
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
							Zones: []core.AvailabilityZone{
								{
									Name: "partition-c",
								},
							},
						},
					},
				},
			}

			cloudProfileConfig = &apismetal.CloudProfileConfig{}
			path = field.NewPath("test")
		})

		It("should pass empty configuration", func() {
			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

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

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

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

			errorList := ValidateCloudProfileConfig(cloudProfileConfig, cloudProfile, path)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("test.metalControlPlanes.prod"),
				"BadValue": Equal("random-partition"),
				"Detail":   Equal("the control plane has a partition that is not a configured zone in any of the cloud profile regions: [partition-a partition-b partition-c]"),
			}))))
		})
	})
})
