package bastion

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	extensionsbastion "github.com/gardener/gardener/extensions/pkg/bastion"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	confighelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/helper"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
)

// maxLengthForBaseName is the maximum length of the deterministic part of the bastion machine name.
const maxLengthForBaseName = 33

// Options contains provider-related information required for setting up a bastion instance.
type Options struct {
	// MachineName is the deterministic name of the bastion machine.
	MachineName string
	// Size is the metal machine size id of the bastion.
	Size string
	// Image is the metal image id of the bastion.
	Image string
	// Partition is the metal partition id the bastion is allocated in.
	Partition string
	// Project is the metal project id the bastion is allocated in.
	Project string
	// NetworkIDs contains all networks the bastion is allocated in (private node network + public networks).
	NetworkIDs []string
	// PublicNetworkIDs contains the public networks the bastion is allocated in.
	PublicNetworkIDs []string
}

// providerStatusRaw is the provider specific status stored in the Bastion resource.
type providerStatusRaw struct {
	MachineName string `json:"machineName"`
	MachineID   string `json:"machineId"`
}

// DetermineOptions determines the required information that is required to reconcile a Bastion on metal-stack.
// This function does not create any IaaS resources.
func DetermineOptions(
	bastion *extensionsv1alpha1.Bastion,
	cluster *extensionscontroller.Cluster,
	cloudProfileConfig *apismetal.CloudProfileConfig,
	infrastructureConfig *apismetal.InfrastructureConfig,
	privateNetworkID string,
	machineImages []config.MachineImage,
) (*Options, error) {
	if cluster == nil || cluster.CloudProfile == nil {
		return nil, fmt.Errorf("cloud profile is not set for cluster")
	}

	machineSpec, err := extensionsbastion.GetMachineSpecFromCloudProfile(cluster.CloudProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to determine bastion machine spec from cloud profile: %w", err)
	}

	image, err := confighelper.FindImage(machineImages, machineSpec.ImageBaseName, machineSpec.ImageVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to find bastion machine image: %w", err)
	}

	if _, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID); err != nil {
		return nil, err
	}

	machineName, err := generateBastionBaseResourceName(cluster.ObjectMeta.Name, bastion.Name)
	if err != nil {
		return nil, err
	}

	publicNetworkIDs := infrastructureConfig.Firewall.Networks
	networkIDs := make([]string, 0, len(publicNetworkIDs)+1)
	networkIDs = append(networkIDs, privateNetworkID)
	networkIDs = append(networkIDs, publicNetworkIDs...)

	return &Options{
		MachineName:      machineName,
		Size:             machineSpec.MachineTypeName,
		Image:            image,
		Partition:        infrastructureConfig.PartitionID,
		Project:          infrastructureConfig.ProjectID,
		NetworkIDs:       networkIDs,
		PublicNetworkIDs: publicNetworkIDs,
	}, nil
}

// generateBastionBaseResourceName generates a deterministic, unique name for the bastion machine.
func generateBastionBaseResourceName(clusterName string, bastionName string) (string, error) {
	if clusterName == "" {
		return "", fmt.Errorf("clusterName can't be empty")
	}
	if bastionName == "" {
		return "", fmt.Errorf("bastionName can't be empty")
	}

	staticName := clusterName + "-" + bastionName
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(staticName)))
	if len(staticName) > maxLengthForBaseName {
		staticName = staticName[:maxLengthForBaseName]
	}

	return fmt.Sprintf("%s-bastion-%s", staticName, hash[:5]), nil
}

func getProviderStatus(bastion *extensionsv1alpha1.Bastion) (*providerStatusRaw, error) {
	if bastion.Status.ProviderStatus != nil && bastion.Status.ProviderStatus.Raw != nil {
		return unmarshalProviderStatus(bastion.Status.ProviderStatus.Raw)
	}
	return nil, nil
}

func marshalProviderStatus(status *providerStatusRaw) ([]byte, error) {
	return json.Marshal(status)
}

func unmarshalProviderStatus(bytes []byte) (*providerStatusRaw, error) {
	status := &providerStatusRaw{}
	if err := json.Unmarshal(bytes, status); err != nil {
		return nil, fmt.Errorf("failed to parse json for status.ProviderStatus: %w", err)
	}
	return status, nil
}
