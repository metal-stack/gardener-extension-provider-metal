package validation

import (
	"fmt"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig *apismetal.ControlPlaneConfig, cloudProfileConfig *apismetal.CloudProfileConfig, partitionName string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	partition, partPath, errs := findMetalControlPlane(cloudProfileConfig, partitionName, fldPath)
	if len(errs) != 0 {
		return append(allErrs, errs...)
	}
	allErrs = append(allErrs, validateNetworkAccessFields(controlPlaneConfig, fldPath, partition, partPath)...)

	return allErrs
}

func validateNetworkAccessFields(controlPlaneConfig *apismetal.ControlPlaneConfig, cpcPath *field.Path, partition *apismetal.Partition, partPath *field.Path) field.ErrorList {

	if controlPlaneConfig.NetworkAccessType == nil || *controlPlaneConfig.NetworkAccessType == apismetal.NetworkAccessBaseline {
		return nil
	}

	allErrs := field.ErrorList{}
	natPath := cpcPath.Child("networkAccessType")
	partNiPath := partPath.Child("networkIsolation")

	if partition.NetworkIsolation == nil {
		allErrs = append(allErrs,
			field.Invalid(natPath, controlPlaneConfig.NetworkAccessType, "network access type requires partition's networkIsolation to be set"),
			field.Required(partNiPath, "network isolation required if control plane config networkAccess is not baseline"),
		)
	}

	return allErrs
}

func findMetalControlPlane(cloudProfileConfig *apismetal.CloudProfileConfig, partition string, cpcPath *field.Path) (*apismetal.Partition, *field.Path, field.ErrorList) {
	for mcpName, mcp := range cloudProfileConfig.MetalControlPlanes {
		for partitionName, p := range mcp.Partitions {
			if partitionName == partition {
				partitionPath := cpcPath.
					Child("metalControlPlanes").
					Key(mcpName).
					Child("partitions").
					Key(partitionName)
				return &p, partitionPath, nil
			}
		}
	}
	return nil, nil, field.ErrorList{
		field.Invalid(cpcPath.Child("metalControlPlanes"), cloudProfileConfig.MetalControlPlanes, fmt.Sprintf("missing partition with name %q", partition)),
	}
}
