package validation_test

import (
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
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
			workers      []core.Worker
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
			workers = []core.Worker{
				{
					Machine: core.Machine{
						Type: "c1-xlarge-x86",
						Image: &core.ShootMachineImage{
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
			workers[0].Volume = &core.Volume{
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
