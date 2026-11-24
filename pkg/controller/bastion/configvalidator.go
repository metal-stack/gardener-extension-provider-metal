package bastion

import (
	"context"
	"fmt"

	extensionsbastion "github.com/gardener/gardener/extensions/pkg/bastion"
	"github.com/gardener/gardener/extensions/pkg/controller/bastion"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/extensions"
	confighelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/helper"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
)

type configValidator struct {
	decoder             runtime.Decoder
	machineImageMapping []config.MachineImage
}

// NewConfigValidator creates a new ConfigValidator.
func NewConfigValidator(mgr manager.Manager, machineImages []config.MachineImage) bastion.ConfigValidator {
	return &configValidator{
		decoder:             serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
		machineImageMapping: machineImages,
	}
}

// Validate validates the bastion configuration against the cloud provider.
func (c *configValidator) Validate(_ context.Context, _ *extensionsv1alpha1.Bastion, cluster *extensions.Cluster) field.ErrorList {
	allErrs := field.ErrorList{}

	cloudProfileConfig, err := helper.CloudProfileConfigFromCluster(cluster)
	if err != nil {
		return append(allErrs, field.InternalError(nil, err))
	}
	if cloudProfileConfig == nil {
		return append(allErrs, field.InternalError(nil, fmt.Errorf("cloud profile config is not set")))
	}

	infrastructureConfig, err := getInfrastructureConfig(c.decoder, cluster)
	if err != nil {
		return append(allErrs, field.InternalError(nil, err))
	}

	if _, _, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID); err != nil {
		return append(allErrs, field.InternalError(nil, err))
	}

	if len(infrastructureConfig.Firewall.Networks) == 0 {
		return append(allErrs, field.Required(field.NewPath("firewall", "networks"), "at least one public network is required for a bastion"))
	}

	if cluster.CloudProfile == nil {
		return append(allErrs, field.InternalError(field.NewPath("cloudProfile"), fmt.Errorf("cloud profile is not set")))
	}

	machineSpec, err := extensionsbastion.GetMachineSpecFromCloudProfile(cluster.CloudProfile)
	if err != nil {
		return append(allErrs, field.InternalError(field.NewPath("bastion"), err))
	}

	if _, err := confighelper.FindImage(c.machineImageMapping, machineSpec.ImageBaseName, machineSpec.ImageVersion); err != nil {
		allErrs = append(allErrs, field.InternalError(field.NewPath("machineImage"), err))
	}

	return allErrs
}
