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

	allErrs = append(allErrs, validateIAMConfig(controlPlaneConfig, fldPath)...)
	allErrs = append(allErrs, validateFeatureGates(controlPlaneConfig, fldPath)...)

	return allErrs
}

func validateIAMConfig(controlPlaneConfig *apismetal.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	iam := controlPlaneConfig.IAMConfig
	if iam == nil {
		// assume iam is disabled
		return allErrs
	}

	iamPath := fldPath.Child("iamconfig")

	issuer := iam.IssuerConfig
	issuerPath := iamPath.Child("issuerConfig")
	if issuer == nil {
		allErrs = append(allErrs, field.Required(issuerPath, "issuer config must be specified"))
	} else {
		if issuer.Url == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("url"), "url must be specified"))
		}
		if issuer.ClientId == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("clientId"), "clientId must be specified"))
		}
	}

	groupConfig := iam.GroupConfig
	groupConfigPath := iamPath.Child("groupConfig")
	if groupConfig != nil {
		if groupConfig.NamespaceMaxLength <= 0 {
			allErrs = append(allErrs, field.Required(groupConfigPath.Child("namespaceMaxLength"), "namespaceMaxLength must be a positive integer"))
		}
	}

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
