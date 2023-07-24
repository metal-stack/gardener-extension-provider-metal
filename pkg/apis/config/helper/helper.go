package helper

import (
	"fmt"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
)

// FindImage takes a list of machine images, and the desired image name and version. It tries
// to find the image with the given name and version. If it cannot be found then an error
// is returned.
func FindImage(machineImages []config.MachineImage, imageName, version string) (string, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == imageName && machineImage.Version == version {
			return machineImage.Image, nil
		}
	}

	return "", fmt.Errorf("could not find an image for name %q in version %q", imageName, version)
}
