package validation

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

const (
	FirewallControllerVersionAuto = "auto"
)

func ValidateFirewallControllerVersion(availableVersions []apismetal.FirewallControllerVersion, specVersion string) (*apismetal.FirewallControllerVersion, error) {
	// If auto or "" is specified in the shoot spec, take the latest available version
	if specVersion == FirewallControllerVersionAuto || specVersion == "" {
		return getLatestFirewallControllerVersion(availableVersions)
	}

	var versions []string
	for _, v := range availableVersions {
		availableVersion := v

		if availableVersion.Version == specVersion {
			return &availableVersion, nil
		}

		versions = append(versions, availableVersion.Version)
	}

	return nil, fmt.Errorf("firewall controller version %q was not found in available versions: %v", specVersion, versions)
}

func getLatestFirewallControllerVersion(availableVersions []apismetal.FirewallControllerVersion) (*apismetal.FirewallControllerVersion, error) {

	av := []apismetal.FirewallControllerVersion{}
	for _, v := range availableVersions {
		_, err := semver.NewVersion(v.Version)
		if err != nil {
			continue
		}
		// no given classification considered as preview
		if v.Classification == nil {
			continue
		}
		// only "supported" counts
		if v.Classification != nil && *v.Classification != apismetal.ClassificationSupported {
			continue
		}
		av = append(av, v)
	}

	sort.Slice(av, func(i, j int) bool {
		ri := av[i]
		rj := av[j]
		vi, err := semver.NewVersion(ri.Version)
		if err != nil {
			return true
		}
		vj, err := semver.NewVersion(rj.Version)
		if err != nil {
			return false
		}

		return vi.LessThan(vj)
	})

	if len(av) == 0 {
		return nil, fmt.Errorf("unable to detect most recent firewallcontrollerversion")
	}
	return &av[len(av)-1], nil
}
