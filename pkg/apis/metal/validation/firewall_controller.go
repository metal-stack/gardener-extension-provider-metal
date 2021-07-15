package validation

import (
	"fmt"

	"github.com/gardener/gardener/pkg/utils/imagevector"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

const (
	FirewallControllerVersionAuto = "auto"
)

func ValidateFirewallControllerVersion(iv imagevector.ImageVector, availableVersions []apismetal.FirewallControllerVersion, specVersion string) (*apismetal.FirewallControllerVersion, error) {
	if specVersion == FirewallControllerVersionAuto {
		ivi, err := iv.FindImage("firewall-controller")
		if err != nil {
			return nil, fmt.Errorf("firewall-controller is not present in image-vector")
		}

		specVersion = *ivi.Tag
	}

	for _, availableVersion := range availableVersions {
		if availableVersion.Version == specVersion {
			return &availableVersion, nil
		}
	}

	return nil, fmt.Errorf("firewall controller url was not found in available versions: %s", specVersion)
}
