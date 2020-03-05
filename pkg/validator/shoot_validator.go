package validator

import (
	"context"
	"errors"
	"reflect"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalvalidation "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (v *Shoot) validateShoot(ctx context.Context, shoot *core.Shoot) error {
	// Provider validation
	fldPath := field.NewPath("spec", "provider")

	// InfrastructureConfig
	infraConfigFldPath := fldPath.Child("infrastructureConfig")

	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(infraConfigFldPath, "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfig(infraConfig); len(errList) != 0 {
		return errList.ToAggregate()
	}

	cloudProfile := &gardencorev1beta1.CloudProfile{}
	if err := v.client.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), cloudProfile); err != nil {
		return err
	}

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfigAgainstCloudProfile(infraConfig, shoot, cloudProfile, cloudProfileConfig, infraConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// ControlPlaneConfig
	controlPlaneConfigFldPath := fldPath.Child("controlPlaneConfig")

	controlPlaneConfig, err := decodeControlPlaneConfig(v.decoder, shoot.Spec.Provider.ControlPlaneConfig, fldPath.Child("controlPlaneConfig"))
	if err != nil {
		return err
	}

	controlPlaneConfig.IAMConfig, err = helper.MergeIAMConfig(controlPlaneConfig.IAMConfig, cloudProfileConfig.IAMConfig)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, controlPlaneConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// Shoot workers
	if errList := metalvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, cloudProfile, fldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (v *Shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	fldPath := field.NewPath("spec", "provider")

	// InfrastructureConfig update
	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(fldPath.Child("infrastructureConfig"), "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(v.decoder, shoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	if oldShoot.Spec.Provider.InfrastructureConfig == nil {
		return field.InternalError(fldPath.Child("infrastructureConfig"), errors.New("InfrastructureConfig is not available on old shoot"))
	}

	oldInfraConfig, err := decodeInfrastructureConfig(v.decoder, oldShoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldInfraConfig, infraConfig) {
		v.Logger.Info("differs!")
		if errList := metalvalidation.ValidateInfrastructureConfigUpdate(oldInfraConfig, infraConfig); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	return v.validateShoot(ctx, shoot)
}
