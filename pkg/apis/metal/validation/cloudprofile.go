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

			dnsServers := partition.NetworkIsolation.DNSServers
			dnsServersField := networkIsolationField.Child("dnsServers")
			allErrs = append(allErrs, validateDNSServers(dnsServers, dnsServersField)...)

			ntpServers := partition.NetworkIsolation.NTPServers
			ntpServersField := networkIsolationField.Child("ntpServers")
			allErrs = append(allErrs, validateNTPServers(ntpServers, ntpServersField)...)

			allowedNetworks := partition.NetworkIsolation.AllowedNetworks
			allowedNetworksField := networkIsolationField.Child("allowedNetworks")
			allErrs = append(allErrs, validateAllowedNetworks(allowedNetworksField, allowedNetworks)...)

			registryMirrors := partition.NetworkIsolation.RegistryMirrors
			registryMirrorsField := networkIsolationField.Child("registryMirrors")
			allErrs = append(allErrs, validateRegistryMirrors(registryMirrors, registryMirrorsField)...)
		}
	}

	return allErrs
}

func validateDNSServers(dnsServers []string, dnsField *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if len(dnsServers) == 0 {
		errs = append(errs, field.Invalid(dnsField, dnsServers, "may not be empty"))
	}
	if len(dnsServers) > 3 {
		errs = append(errs, field.Invalid(dnsField, dnsServers, "only up to 3 dns servers are allowed"))
	}
	for index, ip := range dnsServers {
		ipField := dnsField.Index(index)
		if _, err := netip.ParseAddr(ip); err != nil {
			errs = append(errs, field.Invalid(ipField, ip, "invalid ip address"))
		}
	}
	return errs
}

func validateNTPServers(ntpServers []string, ntpField *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if len(ntpServers) == 0 {
		errs = append(errs, field.Invalid(ntpField, ntpServers, "may not be empty"))
	}
	for index, ip := range ntpServers {
		ipField := ntpField.Index(index)
		if _, err := netip.ParseAddr(ip); err != nil {
			errs = append(errs, field.Invalid(ipField, ip, "invalid ip address"))
		}
	}
	return errs
}

func validateAllowedNetworks(allowedNetworksField *field.Path, allowedNetworks apismetal.AllowedNetworks) field.ErrorList {
	errs := field.ErrorList{}

	egress := allowedNetworks.Egress
	egressField := allowedNetworksField.Child("egress")
	if len(egress) == 0 {
		errs = append(errs, field.Invalid(egressField, egress, "may not be empty"))
	}
	for index, cidr := range egress {
		ipField := egressField.Index(index)
		if _, err := netip.ParsePrefix(cidr); err != nil {
			errs = append(errs, field.Invalid(ipField, cidr, "invalid cidr"))
		}
	}
	ingress := allowedNetworks.Ingress
	ingressField := allowedNetworksField.Child("ingress")
	if len(ingress) == 0 {
		errs = append(errs, field.Invalid(ingressField, ingress, "may not be empty"))
	}
	for index, cidr := range ingress {
		ipField := ingressField.Index(index)
		if _, err := netip.ParsePrefix(cidr); err != nil {
			errs = append(errs, field.Invalid(ipField, cidr, "invalid cidr"))
		}
	}
	return errs
}

func validateRegistryMirrors(registryMirrors []apismetal.RegistryMirror, registryMirrorsField *field.Path) field.ErrorList {
	errs := field.ErrorList{}
	if len(registryMirrors) == 0 {
		errs = append(errs, field.Invalid(registryMirrorsField, registryMirrors, "may not be empty"))
	}
	for mirrIndex, mirr := range registryMirrors {
		mirrorField := registryMirrorsField.Index(mirrIndex)
		if mirr.Name == "" {
			errs = append(errs, field.Invalid(mirrorField.Child("name"), mirr.Name, "name of mirror may not be empty"))
		}
		endpointUrl, err := url.Parse(mirr.Endpoint)
		if err != nil {
			errs = append(errs, field.Invalid(mirrorField.Child("endpoint"), mirr.Endpoint, "not a valid url"))
		} else if endpointUrl.Scheme != "http" && endpointUrl.Scheme != "https" {
			errs = append(errs, field.Invalid(mirrorField.Child("endpoint"), mirr.Endpoint, "url must have the scheme http/s"))
		}
		if _, err := netip.ParseAddr(mirr.IP); err != nil {
			errs = append(errs, field.Invalid(mirrorField.Child("ip"), mirr.IP, "invalid ip address"))
		}
		if mirr.Port == 0 {
			errs = append(errs, field.Invalid(mirrorField.Child("port"), mirr.Port, "must be a valid port"))
		}
		if len(mirr.MirrorOf) == 0 {
			errs = append(errs, field.Invalid(mirrorField.Child("mirrorOf"), mirr.MirrorOf, "registry mirror must replace existing registries"))
		}

		for regIndex, reg := range mirr.MirrorOf {
			regField := mirrorField.Child("mirrorOf").Index(regIndex)
			if reg == "" {
				errs = append(errs, field.Invalid(regField, reg, "cannot be empty"))
			}
			regUrl, err := url.Parse("https://" + reg + "/")
			if err != nil {
				errs = append(errs, field.Invalid(regField, reg, "invalid registry"))
			}
			if regUrl.Host != reg {
				errs = append(errs, field.Invalid(regField, reg, "not a valid registry host"))
			}
		}
	}
	return errs
}

func ValidateImmutableCloudProfileConfig(
	newCloudProfileConfig *apismetal.CloudProfileConfig,
	oldCloudProfileConfig *apismetal.CloudProfileConfig,
	providerConfigPath *field.Path,
) field.ErrorList {
	if oldCloudProfileConfig == nil {
		return nil
	}
	allErrs := field.ErrorList{}

	controlPlanesPath := providerConfigPath.Child("metalControlPlanes")
	for mcpName, mcp := range newCloudProfileConfig.MetalControlPlanes {
		oldMcp, ok := oldCloudProfileConfig.MetalControlPlanes[mcpName]
		if !ok {
			continue
		}
		mcpField := controlPlanesPath.Child(mcpName)

		for partitionName, partition := range mcp.Partitions {
			oldPartition, ok := oldMcp.Partitions[partitionName]
			if !ok {
				continue
			}

			if oldPartition.NetworkIsolation == nil {
				continue
			}

			networkIsolationField := mcpField.Child(partitionName, "networkIsolation")

			if oldPartition.NetworkIsolation != nil && partition.NetworkIsolation == nil {
				allErrs = append(allErrs, field.Required(networkIsolationField, "cannot remove existing network isolations"))
				continue
			}

			if len(partition.NetworkIsolation.DNSServers) != len(oldPartition.NetworkIsolation.DNSServers) {
				dnsField := networkIsolationField.Child("dnsServers")
				allErrs = append(allErrs, field.NotSupported(
					dnsField,
					partition.NetworkIsolation.DNSServers,
					[]string{
						fmt.Sprintf("%s", partition.NetworkIsolation.DNSServers),
					},
				))
				continue
			}
			for index, ip := range partition.NetworkIsolation.DNSServers {
				ipField := networkIsolationField.Child("dnsServers").Index(index)
				oldIp := oldPartition.NetworkIsolation.DNSServers[index]
				if ip != oldIp {
					allErrs = append(allErrs, field.NotSupported(
						ipField,
						ip,
						[]string{oldIp},
					))
				}
			}
		}
	}

	return allErrs
}
