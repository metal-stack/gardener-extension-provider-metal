package validation

import (
	"fmt"
	"net"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	apivalidation "k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
)

// ValidateInfrastructureConfigAgainstCloudProfile validates the given `InfrastructureConfig` against the given `CloudProfile`.
func ValidateInfrastructureConfigAgainstCloudProfile(infra *apismetal.InfrastructureConfig, shoot *core.Shoot, cloudProfile *gardencorev1beta1.CloudProfile, cloudProfileConfig *apismetal.CloudProfileConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	shootRegion := shoot.Spec.Region
	for _, region := range cloudProfile.Spec.Regions {
		if region.Name == shootRegion {
			allErrs = append(allErrs, validateInfrastructureConfigZones(infra, region.Zones, fldPath)...)
			break
		}
	}

	firewallPath := fldPath.Child("firewall")

	if len(infra.Firewall.Networks) == 0 {
		allErrs = append(allErrs, field.Required(firewallPath.Child("networks"), "at least one external network needs to be defined as otherwise the cluster will under no circumstances be able to bootstrap"))
	}

	if cloudProfileConfig == nil {
		return allErrs
	}

	availableFirewallImages := sets.NewString()
	var availableFirewallControllerVersions []apismetal.FirewallControllerVersion
	for _, mcp := range cloudProfileConfig.MetalControlPlanes {
		availableFirewallImages.Insert(mcp.FirewallImages...)
		availableFirewallControllerVersions = append(
			availableFirewallControllerVersions,
			mcp.FirewallControllerVersions...,
		)
	}

	// Check if firewall image is allowed
	found := false
	for _, image := range availableFirewallImages.List() {
		if infra.Firewall.Image == image {
			found = true
			break
		}
	}
	if !found {
		allErrs = append(allErrs, field.Invalid(firewallPath.Child("image"), infra.Firewall.Image, fmt.Sprintf("supported values: %v", availableFirewallImages.List())))
	}

	// Check if firewall-controller version is allowed
	if _, err := ValidateFirewallControllerVersion(
		imagevector.ImageVector(),
		availableFirewallControllerVersions,
		infra.Firewall.ControllerVersion,
	); err != nil {
		allErrs = append(allErrs, field.Required(field.NewPath("controllerVersion"), err.Error()))
	}

	_, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infra.PartitionID)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("partitionID"), infra.PartitionID, "cloud profile does not define the given shoot partition"))
	}

	return allErrs
}

// validateInfrastructureConfigZones validates the given `InfrastructureConfig` against the given `Zones`.
func validateInfrastructureConfigZones(infra *apismetal.InfrastructureConfig, zones []gardencorev1beta1.AvailabilityZone, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableZones := sets.NewString()
	for _, zone := range zones {
		availableZones.Insert(zone.Name)
	}

	if !availableZones.Has(infra.PartitionID) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("partitionID"), infra.PartitionID, fmt.Sprintf("supported values: %v", availableZones.UnsortedList())))
	}

	return allErrs
}

// ValidateInfrastructureConfig validates a InfrastructureConfig object.
func ValidateInfrastructureConfig(infra *apismetal.InfrastructureConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	if infra.ProjectID == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("projectID"), "projectID must be specified"))
	}
	if infra.PartitionID == "" {
		allErrs = append(allErrs, field.Required(field.NewPath("partitionID"), "partitionID must be specified"))
	}

	firewallPath := field.NewPath("firewall")
	if infra.Firewall.Image == "" {
		allErrs = append(allErrs, field.Required(firewallPath.Child("image"), "firewall image must be specified"))
	}
	if infra.Firewall.Size == "" {
		allErrs = append(allErrs, field.Required(firewallPath.Child("size"), "firewall size must be specified"))
	}

	availableNetworks := sets.NewString()
	for i, network := range infra.Firewall.Networks {
		if network == "" {
			allErrs = append(allErrs, field.Required(firewallPath.Child("networks").Index(i), "firewall network must not be an empty string"))
			continue
		}
		availableNetworks.Insert(network)
	}

	for i, rateLimit := range infra.Firewall.RateLimits {
		fp := firewallPath.Child("rateLimit").Index(i)
		if rateLimit.NetworkID == "" {
			allErrs = append(allErrs, field.Required(fp, "rate limit network must not be an empty string"))
			continue
		}
		if !availableNetworks.Has(rateLimit.NetworkID) {
			allErrs = append(allErrs, field.Required(fp, "rate limit network must be present as cluster network"))
			continue
		}
	}

	for i, egress := range infra.Firewall.EgressRules {
		fp := firewallPath.Child("egressRules").Index(i)
		if egress.NetworkID == "" {
			allErrs = append(allErrs, field.Required(fp, "egress rule network must not be an empty string"))
			continue
		}
		if !availableNetworks.Has(egress.NetworkID) {
			allErrs = append(allErrs, field.Required(fp, "egress rule network must be present as cluster network"))
			continue
		}
		if len(egress.IPs) == 0 {
			allErrs = append(allErrs, field.Required(fp, "egress rule must contain ip addresses to use"))
			continue
		}
		for _, ip := range egress.IPs {
			if net.ParseIP(ip) == nil {
				allErrs = append(allErrs, field.Required(fp, "egress rule contains a malformed ip address"))
			}
		}
	}

	return allErrs
}

// ValidateInfrastructureConfigUpdate validates a InfrastructureConfig object.
func ValidateInfrastructureConfigUpdate(oldConfig, newConfig *apismetal.InfrastructureConfig, cloudProfileConfig *apismetal.CloudProfileConfig) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.ProjectID, oldConfig.ProjectID, field.NewPath("projectID"))...)
	allErrs = append(allErrs, apivalidation.ValidateImmutableField(newConfig.PartitionID, oldConfig.PartitionID, field.NewPath("partitionID"))...)

	firewallPath := field.NewPath("firewall")

	if len(newConfig.Firewall.Networks) == 0 {
		allErrs = append(allErrs, field.Required(firewallPath.Child("networks"), "at least one external network needs to be defined as otherwise the cluster will under no circumstances be able to bootstrap"))
	}

	_, _, err := helper.FindMetalControlPlane(cloudProfileConfig, newConfig.PartitionID)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(field.NewPath("partitionID"), newConfig.PartitionID, "cloud profile does not define the given shoot partition"))
	}

	return allErrs
}
