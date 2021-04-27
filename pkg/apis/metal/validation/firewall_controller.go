package validation

import (
	"context"
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
)

const (
	FirewallControllerVersionAuto = "auto"
)

var (
	ErrSpecVersionUndefined = fmt.Errorf("firewall-controller version was not specified in the spec")
	ErrNoSemver             = fmt.Errorf("firewall-controller versions must adhere to semver spec")
	ErrControllerTooOld     = fmt.Errorf("firewall-controller on machine is too old")

	ghclient      = github.NewClient(nil)
	versionsCache = make(map[string]bool)
)

func ValidateFirewallControllerVersion(iv imagevector.ImageVector, specVersion string) (string, error) {
	versionTag, err := validateFirewallControllerVersionWithoutGithub(iv, specVersion)
	if err != nil {
		return "", err
	}

	err = isFirewallControllerVersionValid(versionTag)
	if err != nil {
		return "", err
	}

	return versionTag, nil
}

func isFirewallControllerVersionValid(versionTag string) error {
	if versionsCache[versionTag] {
		return nil
	}

	releases, _, err := ghclient.Repositories.ListReleases(context.Background(), "metal-stack", "firewall-controller", &github.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "unable to list github firewall-controller releases")
	}

	var rel *github.RepositoryRelease
	for _, r := range releases {
		if r.TagName != nil && *r.TagName == versionTag {
			rel = r
			break
		}
	}

	if rel == nil {
		return fmt.Errorf("could not find release with tag %s", versionTag)
	}

	var asset *github.ReleaseAsset
	for _, ra := range rel.Assets {
		if ra.GetName() == "firewall-controller" {
			asset = &ra
			break
		}
	}

	if asset == nil {
		return fmt.Errorf("could not find artifact %q in github release with tag %s", "firewall-controller", versionTag)
	}

	versionsCache[versionTag] = true

	return nil
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
