package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"

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
				ControllerVersion: "auto",
			},
		}
	})

	Describe("#ValidateInfrastructureConfigAgainstCloudProfile", func() {
		var (
			cloudProfile       *gardencorev1beta1.CloudProfile
			cloudProfileConfig *apismetal.CloudProfileConfig
			shoot              *core.Shoot
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
				shoot = &core.Shoot{
					Spec: core.ShootSpec{
						Region: "region-a",
					},
				}

				cloudProfileConfig = createCloudProfileConfig()
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
		})
	})

	Describe("#ValidateInfrastructureConfig", func() {
		Context("Zones", func() {
			It("should forbid empty partition", func() {
				infrastructureConfig.PartitionID = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("partitionID"),
					"Detail": Equal("partitionID must be specified"),
				}))))
			})

			It("should forbid empty project", func() {
				infrastructureConfig.ProjectID = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("projectID"),
					"Detail": Equal("projectID must be specified"),
				}))))
			})

			It("should forbid empty firewall image", func() {
				infrastructureConfig.Firewall.Image = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.image"),
					"Detail": Equal("firewall image must be specified"),
				}))))
			})

			It("should forbid empty firewall size", func() {
				infrastructureConfig.Firewall.Size = ""

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.size"),
					"Detail": Equal("firewall size must be specified"),
				}))))
			})

			It("should forbid empty network", func() {
				infrastructureConfig.Firewall.Networks = []string{"internet", ""}

				errorList := ValidateInfrastructureConfig(infrastructureConfig)

				Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(field.ErrorTypeRequired),
					"Field":  Equal("firewall.networks[1]"),
					"Detail": Equal("firewall network must not be an empty string"),
				}))))
			})
		})
	})

	Describe("#ValidateInfrastructureConfigUpdate", func() {
		var (
			cloudProfileConfig *apismetal.CloudProfileConfig
		)
		BeforeEach(func() {
			cloudProfileConfig = &apismetal.CloudProfileConfig{
				MetalControlPlanes: map[string]apismetal.MetalControlPlane{
					"prod": {
						FirewallImages: []string{"image"},
						Partitions: map[string]apismetal.Partition{
							"partition-a": {},
						},
					},
				},
			}
		})

		It("should return no errors for an unchanged config", func() {
			Expect(ValidateInfrastructureConfigUpdate(infrastructureConfig, infrastructureConfig, cloudProfileConfig)).To(BeEmpty())
		})

		It("should not allow changing partition", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.PartitionID = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig, cloudProfileConfig)

			Expect(errorList).To(ContainElements(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("partitionID"),
			}))))
		})

		It("should not allow changing project", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.ProjectID = "unknown"

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig, cloudProfileConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("projectID"),
			}))))
		})

		It("should not allow removing all networks", func() {
			newInfrastructureConfig := infrastructureConfig.DeepCopy()
			newInfrastructureConfig.Firewall.Networks = []string{}

			errorList := ValidateInfrastructureConfigUpdate(infrastructureConfig, newInfrastructureConfig, cloudProfileConfig)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("firewall.networks"),
				"Detail": Equal("at least one external network needs to be defined as otherwise the cluster will under no circumstances be able to bootstrap"),
			}))))
		})
	})
})

func createCloudProfileConfig() *apismetal.CloudProfileConfig {
	iv := imagevector.ImageVector()
	ivi, err := iv.FindImage("firewall-controller")
	Expect(err).To(BeNil())
	specVersion := *ivi.Tag
	return &apismetal.CloudProfileConfig{
		MetalControlPlanes: map[string]apismetal.MetalControlPlane{
			"prod": {
				FirewallImages: []string{"image"},
				Partitions: map[string]apismetal.Partition{
					"partition-a": {},
				},
				FirewallControllerVersions: []apismetal.FirewallControllerVersion{
					{Version: specVersion},
				},
			},
		},
	}
}
