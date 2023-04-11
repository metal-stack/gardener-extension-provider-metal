package validation

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateControlPlaneConfig validates a ControlPlaneConfig object.
func ValidateControlPlaneConfig(controlPlaneConfig *apismetal.ControlPlaneConfig, cloudProfile *gardencorev1beta1.CloudProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateFeatureGates(controlPlaneConfig, fldPath)...)

	return allErrs
}

func validateFeatureGates(controlPlaneConfig *apismetal.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	fgPath := fldPath.Child("featureGates")
	auditToSplunkPath := fgPath.Child("auditToSplunk")

	if auditToSplunkEnabled(controlPlaneConfig) && !clusterAuditEnabled(controlPlaneConfig) {
		allErrs = append(allErrs, field.Invalid(auditToSplunkPath, true, "cluster audit feature gate has to be enabled when using audit to splunk feature gate"))
	}

	return allErrs
}

func ClusterAuditEnabled(controllerConfig *config.ControllerConfiguration, cpConfig *apismetal.ControlPlaneConfig) bool {
	if !controllerConfig.ClusterAudit.Enabled {
		return false
	}
	return clusterAuditEnabled(cpConfig)
}

func clusterAuditEnabled(cpConfig *apismetal.ControlPlaneConfig) bool {
	if cpConfig.FeatureGates.ClusterAudit != nil && *cpConfig.FeatureGates.ClusterAudit {
		return true
	}
	return false
}

func AuditToSplunkEnabled(controllerConfig *config.ControllerConfiguration, cpConfig *apismetal.ControlPlaneConfig) bool {
	if !controllerConfig.AuditToSplunk.Enabled {
		return false
	}
	return auditToSplunkEnabled(cpConfig)
}

func auditToSplunkEnabled(cpConfig *apismetal.ControlPlaneConfig) bool {
	if cpConfig.FeatureGates.AuditToSplunk != nil && *cpConfig.FeatureGates.AuditToSplunk {
		return true
	}
	return false
}
