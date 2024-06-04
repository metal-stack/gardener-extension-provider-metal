package validation_test

import (
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ControlPlaneconfig validation", func() {
	var (
		controlPlaneConfig *apismetal.ControlPlaneConfig
		cloudProfile       *gardencorev1beta1.CloudProfile
	)

	BeforeEach(func() {
		oot := true
		controlPlaneConfig = &apismetal.ControlPlaneConfig{
			FeatureGates: apismetal.ControlPlaneFeatures{
				MachineControllerManagerOOT: &oot,
			},
		}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))).To(BeEmpty())
		})
	})
})
