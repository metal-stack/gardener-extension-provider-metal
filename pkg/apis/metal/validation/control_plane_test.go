package validation_test

import (
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	. "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("ControlPlaneconfig validation", func() {
	var (
		controlPlaneConfig *apismetal.ControlPlaneConfig
		cloudProfile       *gardencorev1beta1.CloudProfile
	)

	BeforeEach(func() {
		oot := true
		ca := true
		as := false
		controlPlaneConfig = &apismetal.ControlPlaneConfig{
			FeatureGates: apismetal.ControlPlaneFeatures{
				MachineControllerManagerOOT: &oot,
				ClusterAudit:                &ca,
				AuditToSplunk:               &as,
			},
		}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))).To(BeEmpty())
		})

		It("should not allow auditToSplunk without clusterAudit", func() {
			*controlPlaneConfig.FeatureGates.ClusterAudit = false
			*controlPlaneConfig.FeatureGates.AuditToSplunk = true

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":     Equal(field.ErrorTypeInvalid),
				"Field":    Equal("spec.featureGates.auditToSplunk"),
				"BadValue": Equal(true),
				"Detail":   Equal("cluster audit feature gate has to be enabled when using audit to splunk feature gate"),
			}))))
		})
	})
})
