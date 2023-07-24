package validation

import (
	"fmt"

	"github.com/gardener/gardener/pkg/apis/core"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var supportedVersionClassifications = sets.NewString(string(apismetal.ClassificationPreview), string(apismetal.ClassificationSupported), string(apismetal.ClassificationDeprecated))

// ValidateCloudProfileConfig validates a CloudProfileConfig object.
func ValidateCloudProfileConfig(cloudProfileConfig *apismetal.CloudProfileConfig, cloudProfile *core.CloudProfile, providerConfigPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableZones := sets.NewString()
	for _, region := range cloudProfile.Spec.Regions {
		for _, zone := range region.Zones {
			availableZones.Insert(zone.Name)
		}
	}

	controlPlanesPath := providerConfigPath.Child("metalControlPlanes")
	for mcpName, mcp := range cloudProfileConfig.MetalControlPlanes {
		mcpField := controlPlanesPath.Child(mcpName)
		versionSet := sets.NewString()
		for _, v := range mcp.FirewallControllerVersions {
			fwcField := mcpField.Child("firewallControllerVersions")
			if v.Classification != nil && !supportedVersionClassifications.Has(string(*v.Classification)) {
				allErrs = append(allErrs, field.NotSupported(fwcField.Child("classification"), *v.Classification, supportedVersionClassifications.List()))
			}

			versionSet.Insert(v.Version)
		}
		if versionSet.Len() != len(mcp.FirewallControllerVersions) {
			allErrs = append(allErrs, field.Invalid(mcpField.Child("firewallcontrollerversions"), "version", "contains duplicate entries"))
		}

		for partitionName := range mcp.Partitions {
			if !availableZones.Has(partitionName) {
				allErrs = append(allErrs, field.Invalid(mcpField, partitionName, fmt.Sprintf("the control plane has a partition that is not a configured zone in any of the cloud profile regions: %v", availableZones.List())))
			}
		}
	}

	return allErrs
}
