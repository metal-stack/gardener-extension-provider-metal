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
			IAMConfig: &apismetal.IAMConfig{
				IssuerConfig: &apismetal.IssuerConfig{
					Url:      "https://somewhere",
					ClientId: "abc",
				},
				IdmConfig: &apismetal.IDMConfig{
					Idmtype: "UX",
				},
			},
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

		It("should forbid empty iam config", func() {
			controlPlaneConfig.IAMConfig = nil

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig"),
				"Detail": Equal("iam config must be specified"),
			}))))
		})

		It("should forbid empty issuer url", func() {
			controlPlaneConfig.IAMConfig.IssuerConfig.Url = ""

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.issuerConfig.url"),
				"Detail": Equal("url must be specified"),
			}))))
		})

		It("should forbid empty client id", func() {
			controlPlaneConfig.IAMConfig.IssuerConfig.ClientId = ""

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.issuerConfig.clientId"),
				"Detail": Equal("clientId must be specified"),
			}))))
		})

		It("should forbid group namespace length of zero", func() {
			controlPlaneConfig.IAMConfig.GroupConfig = &apismetal.NamespaceGroupConfig{NamespaceMaxLength: 0}

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.groupConfig.namespaceMaxLength"),
				"Detail": Equal("namespaceMaxLength must be a positive integer"),
			}))))
		})

		It("should not allow auditToSplunk without clusterAudit", func() {
			*controlPlaneConfig.FeatureGates.ClusterAudit = false
			*controlPlaneConfig.FeatureGates.AuditToSplunk = true

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, field.NewPath("spec"))

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.featureGates.clusterAudit"),
				"Detail": Equal("is required for spec.featureGates.auditToSplunk but not set"),
			}))))
		})
	})
})
