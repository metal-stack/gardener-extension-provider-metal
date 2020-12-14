package validation

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/metal-stack/firewall-controller/pkg/updater"
)

var (
	ErrSpecVersionUndefined = fmt.Errorf("firewall-controller version was not specified in the spec")
	ErrSpecVersionEmpty     = fmt.Errorf("firewall-controller version must not be empty")
	ErrNoSemver             = fmt.Errorf("firewall-controller versions must adhere to semver spec")
	ErrControllerTooOld     = fmt.Errorf("firewall-controller on machine is too old")
)

func ValidateFirewallControllerVersion(iv imagevector.ImageVector, specVersion *string, autoUpdate bool) (*string, error) {
	versionTag, err := validateFirewallControllerVersionWithoutGithub(iv, specVersion, autoUpdate)
	if err != nil {
		return nil, err
	}

	if versionTag == nil {
		return nil, nil
	}

	_, err = updater.DetermineGithubAsset(*versionTag)
	if err != nil {
		return nil, fmt.Errorf("firewall-controller version must be a github release but version %v was not found", *versionTag)
	}

	return versionTag, nil
}

func validateFirewallControllerVersionWithoutGithub(iv imagevector.ImageVector, specVersion *string, autoUpdate bool) (*string, error) {
	imageVectorVersion, err := getImageVectorVersion(iv)
	if err != nil {
		return nil, err
	}

	wantedVersion, err := determineWantedVersion(specVersion, imageVectorVersion, autoUpdate)
	if err != nil {
		return nil, err
	}

	if wantedVersion == nil {
		return nil, nil
	}

	versionTag := fmt.Sprintf("v%s", wantedVersion.String())
	return &versionTag, nil
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

func determineWantedVersion(specVersion *string, ivSemv *semver.Version, autoUpdate bool) (*semver.Version, error) {
	if specVersion == nil {
		return nil, ErrSpecVersionUndefined
	}

	if specVersion != nil && *specVersion == "" {
		return nil, ErrSpecVersionEmpty
	}

	var wantedSemv *semver.Version
	specSemv, err := semver.Make(strings.TrimPrefix(*specVersion, "v"))
	if err != nil {
		return nil, ErrNoSemver
	}

	if autoUpdate {
		wantedSemv = ivSemv
	} else {
		wantedSemv = &specSemv
	}

	if specSemv.Major != ivSemv.Major {
		return nil, ErrControllerTooOld
	}

	return wantedSemv, nil
}
