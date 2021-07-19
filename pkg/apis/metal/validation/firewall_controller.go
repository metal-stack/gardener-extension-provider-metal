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

	for _, availableVersion := range availableVersions {
		if availableVersion.Version == specVersion {
			return &availableVersion, nil
		}
	}

	return nil, fmt.Errorf("firewall controller version:%s was not found in available versions: %s", specVersion, availableVersions)
}

func getLatestFirewallControllerVersion(availableVersions []apismetal.FirewallControllerVersion) (*apismetal.FirewallControllerVersion, error) {
	av := availableVersions
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
	return &av[len(availableVersions)-1], nil
}
