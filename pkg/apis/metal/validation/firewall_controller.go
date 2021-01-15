package validation

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/metal-stack/firewall-controller/pkg/updater"
)

const (
	FirewallControllerVersionAuto = "auto"
)

var (
	ErrSpecVersionUndefined = fmt.Errorf("firewall-controller version was not specified in the spec")
	ErrNoSemver             = fmt.Errorf("firewall-controller versions must adhere to semver spec")
	ErrControllerTooOld     = fmt.Errorf("firewall-controller on machine is too old")
)

func ValidateFirewallControllerVersion(iv imagevector.ImageVector, specVersion string) (string, error) {
	versionTag, err := validateFirewallControllerVersionWithoutGithub(iv, specVersion)
	if err != nil {
		return "", err
	}

	_, err = updater.DetermineGithubAsset(versionTag)
	if err != nil {
		return "", fmt.Errorf("firewall-controller version must be a github release but version %v was not found", versionTag)
	}

	return versionTag, nil
}

func validateFirewallControllerVersionWithoutGithub(iv imagevector.ImageVector, specVersion string) (string, error) {
	imageVectorVersion, err := getImageVectorVersion(iv)
	if err != nil {
		return "", err
	}

	wantedVersion, err := determineWantedVersion(specVersion, imageVectorVersion)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("v%s", wantedVersion.String()), nil
}

func getImageVectorVersion(imageVector imagevector.ImageVector) (*semver.Version, error) {
	ivi, err := imageVector.FindImage("firewall-controller")
	if err != nil {
		return nil, fmt.Errorf("firewall-controller is not present in image-vector")
	}

	version := *ivi.Tag
	semv, err := semver.Make(strings.TrimPrefix(version, "v"))
	if err != nil {
		return nil, ErrNoSemver
	}
	return &semv, nil
}

func determineWantedVersion(specVersion string, ivSemv *semver.Version) (*semver.Version, error) {
	if specVersion == "" {
		return nil, ErrSpecVersionUndefined
	}

	var wantedSemv *semver.Version
	if specVersion == FirewallControllerVersionAuto {
		wantedSemv = ivSemv
	} else {
		specSemv, err := semver.Make(strings.TrimPrefix(specVersion, "v"))
		if err != nil {
			return nil, ErrNoSemver
		}
		wantedSemv = &specSemv
	}

	if wantedSemv.Major != ivSemv.Major {
		return nil, ErrControllerTooOld
	}

	return wantedSemv, nil
}
