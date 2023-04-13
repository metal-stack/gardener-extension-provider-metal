package helper

import (
	"fmt"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
)

// FindMachineImage takes a list of machine images and tries to find the first entry
// whose name, version, and zone matches with the given name, version, and zone. If no such entry is
// found then an error will be returned.
func FindMachineImage(machineImages []metal.MachineImage, name, version string) (*metal.MachineImage, error) {
	for _, machineImage := range machineImages {
		if machineImage.Name == name && machineImage.Version == version {
			return &machineImage, nil
		}
	}
	return nil, fmt.Errorf("no machine image with name %q, version %q found", name, version)
}

// FindMetalControlPlane returns the metal control plane from a given cluster spec
func FindMetalControlPlane(cloudProfileConfig *metal.CloudProfileConfig, partition string) (*metal.MetalControlPlane, *metal.Partition, error) {
	for _, mcp := range cloudProfileConfig.MetalControlPlanes {
		for partitionName, p := range mcp.Partitions {
			if partitionName == partition {
				return &mcp, &p, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("no metal control plane found for partition %s in cloud profile config", partition)
}
