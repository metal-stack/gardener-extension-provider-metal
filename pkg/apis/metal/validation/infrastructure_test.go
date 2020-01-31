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
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	. "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/gardener/gardener/pkg/utils/validation/gomega"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("InfrastructureConfig validation", func() {
	var (
		infrastructureConfig *apismetal.InfrastructureConfig
	)

	BeforeEach(func() {
		infrastructureConfig = &apismetal.InfrastructureConfig{
			PartitionID: "partition-a",
			ProjectID:   "project-1",
			Firewall: apismetal.Firewall{
				Size:  "c1-xlarge-x86",
				Image: "image",
				Networks: []string{
					"internet",
				},
			},
		}
	})

	Describe("#ValidateInfrastructureConfigAgainstCloudProfile", func() {
		var (
			cloudProfile       *gardencorev1beta1.CloudProfile
			cloudProfileConfig *apismetal.CloudProfileConfig
			shoot              *garden.Shoot
		)

		Context("zones validation", func() {
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
						},
					},
				}
				shoot = &garden.Shoot{
					Spec: garden.ShootSpec{
						Region: "region-a",
					},
				}
				cloudProfileConfig = &apismetal.CloudProfileConfig{
					FirewallNetworks: map[string][]string{"partition-a": []string{"internet"}},
					FirewallImages:   []string{"image"},
				}
			})

			It("should pass because zone is configured in CloudProfile", func() {
				errorList := ValidateInfrastructureConfigAgainstCloudProfile(infrastructureConfig, shoot, cloudProfile, cloudProfileConfig, &field.Path{})

				Expect(errorList).To(BeEmpty())
			})

			It("should forbid because zone is not specified in CloudProfile", func() {
				infrastructureConfig.PartitionID = "not-available"
				errorList := ValidateInfrastructureConfigAgainstCloudProfile(infrastructureConfig, shoot, cloudProfile, cloudProfileConfig, field.NewPath("spec"))

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":  Equal(field.ErrorTypeInvalid),
					"Field": Equal("spec.partitionID"),
				}))))
			})

			It("should forbid firewall image is not specified in CloudProfileConfig", func() {
				infrastructureConfig.Firewall.Image = "no-image"
				errorList := ValidateInfrastructureConfigAgainstCloudProfile(infrastructureConfig, shoot, cloudProfile, cloudProfileConfig, field.NewPath("spec"))

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeInvalid),
					"Field":  Equal("spec.firewall.image"),
					"Detail": Equal("supported values: [image]"),
				}))))
			})

			It("should forbid because no firewall networks given", func() {
				infrastructureConfig.Firewall.Networks = nil
				errorList := ValidateInfrastructureConfigAgainstCloudProfile(infrastructureConfig, shoot, cloudProfile, cloudProfileConfig, field.NewPath("spec"))

				Expect(errorList).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("spec.firewall.networks"),
					"Detail": Equal("at least one external network needs to be defined as otherwise the cluster will under no circumstances be able to bootstrap"),
				}))))
			})
		})
	})

	Describe("#ValidateInfrastructureConfig", func() {
		Context("Zones", func() {
			It("should forbid empty partition", func() {
				infrastructureConfig.PartitionID = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("partitionID"),
					"Detail": Equal("partitionID must be specified"),
				}))
			})

			It("should forbid empty project", func() {
				infrastructureConfig.ProjectID = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("projectID"),
					"Detail": Equal("projectID must be specified"),
				}))
			})

			It("should forbid empty firewall image", func() {
				infrastructureConfig.Firewall.Image = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.image"),
					"Detail": Equal("firewall image must be specified"),
				}))
			})

			It("should forbid empty firewall size", func() {
				infrastructureConfig.Firewall.Size = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.size"),
					"Detail": Equal("firewall size must be specified"),
				}))
			})

			It("should forbid empty network", func() {
				infrastructureConfig.Firewall.Networks = []string{"internet", ""}

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOfFields(Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.networks[1]"),
					"Detail": Equal("firewall network must not be an empty string"),
				}))
			})
		})
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig)).To(BeEmpty())
		})

		It("should not allow changing partition", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.PartitionID = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("partitionID"),
			}))))
		})

		It("should not allow changing project", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.ProjectID = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("projectID"),
			}))))
		})

		It("should not allow changing firewall image", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Image = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("firewall.image"),
			}))))
		})

		It("should not allow changing firewall size", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Size = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("firewall.size"),
			}))))
		})

		It("should not allow adding networks", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Networks = append(newInfrastructureConfig.Firewall.Networks, "b")

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("firewall.networks"),
			}))))
		})

		It("should not allow removing networks", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Networks = []string{}

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("firewall.networks"),
			}))))
		})

		It("order of networks does not matter", func() {
			infrastructureConfig.Firewall.Networks = []string{"a", "b"}
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Networks = []string{"b", "a"}

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig)

			Expect(errorList).To(BeEmpty())
		})
	})

})
