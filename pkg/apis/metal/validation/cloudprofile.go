package validation

import (
	"fmt"
	"net/netip"
	"net/url"

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

		for partitionName, partition := range mcp.Partitions {
			if !availableZones.Has(partitionName) {
				allErrs = append(allErrs, field.Invalid(mcpField, partitionName, fmt.Sprintf("the control plane has a partition that is not a configured zone in any of the cloud profile regions: %v", availableZones.List())))
			}

			if partition.NetworkIsolation == nil {
				continue
			}

			networkIsolationField := mcpField.Child(partitionName, "networkIsolation")

			for index, ip := range partition.NetworkIsolation.DNSServers {
				ipField := networkIsolationField.Child("dnsServers").Index(index)
				if _, err := netip.ParseAddr(ip); err != nil {
					allErrs = append(allErrs, field.Invalid(ipField, ip, "invalid ip address"))
				}
			}
			for index, ip := range partition.NetworkIsolation.NTPServers {
				ipField := networkIsolationField.Child("ntpServers").Index(index)
				if _, err := netip.ParseAddr(ip); err != nil {
					allErrs = append(allErrs, field.Invalid(ipField, ip, "invalid ip address"))
				}
			}
			for index, cidr := range partition.NetworkIsolation.AllowedNetworks.Egress {
				ipField := networkIsolationField.Child("allowedNetworks", "egress").Index(index)
				if _, err := netip.ParsePrefix(cidr); err != nil {
					allErrs = append(allErrs, field.Invalid(ipField, cidr, "invalid cidr"))
				}
			}
			for index, cidr := range partition.NetworkIsolation.AllowedNetworks.Ingress {
				ipField := networkIsolationField.Child("allowedNetworks", "ingress").Index(index)
				if _, err := netip.ParsePrefix(cidr); err != nil {
					allErrs = append(allErrs, field.Invalid(ipField, cidr, "invalid cidr"))
				}
			}

			for mirrIndex, mirr := range partition.NetworkIsolation.RegistryMirrors {
				mirrorField := networkIsolationField.Child("registryMirrors").Index(mirrIndex)
				if mirr.Name == "" {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("name"), mirr.Name, "name of mirror may not be empty"))
				}
				endpointUrl, err := url.Parse(mirr.Endpoint)
				if err != nil {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("endpoint"), mirr.Endpoint, "not a valid url"))
				} else if endpointUrl.Scheme != "http" && endpointUrl.Scheme != "https" {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("endpoint"), mirr.Endpoint, "url must have the scheme http/s"))
				}
				if _, err := netip.ParseAddr(mirr.IP); err != nil {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("ip"), mirr.IP, "invalid ip address"))
				}
				if mirr.Port == 0 {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("port"), mirr.Port, "must be a vaid port"))
				}
				if len(mirr.MirrorOf) == 0 {
					allErrs = append(allErrs, field.Invalid(mirrorField.Child("mirrorOf"), mirr.MirrorOf, "registry mirror must replace existing registries"))
				}

				for regIndex, reg := range mirr.MirrorOf {
					regField := mirrorField.Child("mirrorOf").Index(regIndex)
					if reg == "" {
						allErrs = append(allErrs, field.Invalid(regField, reg, "cannot be empty"))
					}
					regUrl, err := url.Parse("https://" + reg + "/")
					if err != nil {
						allErrs = append(allErrs, field.Invalid(regField, reg, "invalid registry"))
					}
					if regUrl.Host != reg {
						allErrs = append(allErrs, field.Invalid(regField, reg, "not a valid registry host"))
					}
				}
			}
		}
	}

	return allErrs
}
