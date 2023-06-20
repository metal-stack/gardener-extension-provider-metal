package helper

import (
	"fmt"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	corev1 "k8s.io/api/core/v1"
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

// ImagePullPolicyFromString returns an image pull policy from string
// If the pull policy is unknown it returns "IfNotPresent"
func ImagePullPolicyFromString(policy string) corev1.PullPolicy {
	switch p := corev1.PullPolicy(policy); p {
	case corev1.PullAlways, corev1.PullIfNotPresent, corev1.PullNever:
		return p
	default:
		return corev1.PullIfNotPresent
	}
}

// GetNodeCIDR returns the node cidr from the shoot spec. if this is not yet set, it returns the
// node cidr from the infrastructure status. if it's set nowhere, it returns an error.
func GetNodeCIDR(infrastructure *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) (string, error) {
	var nodeCIDR string

	if cluster.Shoot.Spec.Networking.Nodes != nil {
		nodeCIDR = *cluster.Shoot.Spec.Networking.Nodes
	} else if infrastructure != nil && infrastructure.Status.NodesCIDR != nil {
		nodeCIDR = *infrastructure.Status.NodesCIDR
	}

	if nodeCIDR == "" {
		return "", fmt.Errorf("nodeCIDR is not yet set")
	}

	return nodeCIDR, nil
}
