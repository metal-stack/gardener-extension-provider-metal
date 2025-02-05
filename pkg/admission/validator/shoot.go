package validator

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalvalidation "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"github.com/metal-stack/metal-lib/pkg/tag"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/utils/gardener"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewShootValidator returns a new instance of a shoot validator.
func NewShootValidator(mgr manager.Manager) extensionswebhook.Validator {
	return &shoot{
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

type shoot struct {
	client  client.Client
	decoder runtime.Decoder
}

// Validate validates the given shoot object.
func (s *shoot) Validate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*core.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	if old != nil {
		oldShoot, ok := old.(*core.Shoot)
		if !ok {
			return fmt.Errorf("wrong object type %T for old object", old)
		}
		return s.validateShootUpdate(ctx, oldShoot, shoot)
	}

	return s.validateShootCreation(ctx, shoot)
}

func (s *shoot) validateShoot(ctx context.Context, shoot *core.Shoot) error {
	// Provider validation
	fldPath := field.NewPath("spec", "provider")

	_, ok := shoot.Annotations[tag.ClusterTenant]
	if !ok {
		return field.Required(field.NewPath("metadata", "annotations"), fmt.Sprintf("cluster must be annotated with a tenant using the annotations: %s", tag.ClusterTenant))
	}

	// InfrastructureConfig
	infraConfigFldPath := fldPath.Child("infrastructureConfig")

	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(infraConfigFldPath, "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(s.decoder, shoot.Spec.Provider.InfrastructureConfig, infraConfigFldPath)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfig(infraConfig); len(errList) != 0 {
		return errList.ToAggregate()
	}

	shootV1Beta1 := &gardencorev1beta1.Shoot{}
	err = gardencorev1beta1.Convert_core_Shoot_To_v1beta1_Shoot(shoot, shootV1Beta1, nil)
	if err != nil {
		return err
	}

	cloudProfile, err := gardener.GetCloudProfile(ctx, s.client, shootV1Beta1)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfigAgainstCloudProfile(infraConfig, shoot, cloudProfile, cloudProfileConfig, infraConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	controlPlaneConfigFldPath := fldPath.Child("controlPlaneConfig")

	controlPlaneConfig, err := decodeControlPlaneConfig(s.decoder, shoot.Spec.Provider.ControlPlaneConfig, fldPath.Child("controlPlaneConfig"))
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateControlPlaneConfig(controlPlaneConfig, cloudProfile, controlPlaneConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	if errList := metalvalidation.ValidateControlPlaneConfigNetworkAccess(controlPlaneConfig, cloudProfileConfig, infraConfig.PartitionID, controlPlaneConfigFldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	// Shoot workers
	if errList := metalvalidation.ValidateWorkers(shoot.Spec.Provider.Workers, cloudProfile, fldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}

func (s *shoot) validateShootUpdate(ctx context.Context, oldShoot, shoot *core.Shoot) error {
	fldPath := field.NewPath("spec", "provider")

	// InfrastructureConfig update
	if shoot.Spec.Provider.InfrastructureConfig == nil {
		return field.Required(fldPath.Child("infrastructureConfig"), "InfrastructureConfig must be set for metal shoots")
	}

	infraConfig, err := decodeInfrastructureConfig(s.decoder, shoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	if oldShoot.Spec.Provider.InfrastructureConfig == nil {
		return field.InternalError(fldPath.Child("infrastructureConfig"), errors.New("InfrastructureConfig is not available on old shoot"))
	}

	oldInfraConfig, err := decodeInfrastructureConfig(s.decoder, oldShoot.Spec.Provider.InfrastructureConfig, fldPath)
	if err != nil {
		return err
	}

	shootV1Beta1 := &gardencorev1beta1.Shoot{}
	err = gardencorev1beta1.Convert_core_Shoot_To_v1beta1_Shoot(shoot, shootV1Beta1, nil)
	if err != nil {
		return err
	}

	cloudProfile, err := gardener.GetCloudProfile(ctx, s.client, shootV1Beta1)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(oldInfraConfig, infraConfig) {
		if errList := metalvalidation.ValidateInfrastructureConfigUpdate(oldInfraConfig, infraConfig, cloudProfileConfig); len(errList) != 0 {
			return errList.ToAggregate()
		}
	}

	if shoot.Annotations[tag.ClusterTenant] != oldShoot.Annotations[tag.ClusterTenant] {
		return field.Forbidden(field.NewPath("metadata", "annotations"), "tenant annotation of a shoot is immutable")
	}

	return s.validateShoot(ctx, shoot)
}

func (s *shoot) validateShootCreation(ctx context.Context, shoot *core.Shoot) error {
	fldPath := field.NewPath("spec", "provider")
	infraConfig, err := decodeInfrastructureConfig(s.decoder, shoot.Spec.Provider.InfrastructureConfig, fldPath.Child("infrastructureConfig"))
	if err != nil {
		return err
	}

	if err := s.validateAgainstCloudProfile(ctx, shoot, infraConfig, fldPath.Child("infrastructureConfig")); err != nil {
		return err
	}

	return s.validateShoot(ctx, shoot)
}

// func ValidateInfrastructureConfigAgainstCloudProfile(infra *apismetal.InfrastructureConfig, shoot *core.Shoot, cloudProfile *gardencorev1beta1.CloudProfile, cloudProfileConfig *apismetal.CloudProfileConfig, fldPath *field.Path) field.ErrorList {

func (s *shoot) validateAgainstCloudProfile(ctx context.Context, shoot *core.Shoot, infraConfig *apismetal.InfrastructureConfig, fldPath *field.Path) error {
	shootV1Beta1 := &gardencorev1beta1.Shoot{}
	err := gardencorev1beta1.Convert_core_Shoot_To_v1beta1_Shoot(shoot, shootV1Beta1, nil)
	if err != nil {
		return err
	}

	cloudProfile, err := gardener.GetCloudProfile(ctx, s.client, shootV1Beta1)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cloudProfile)
	if err != nil {
		return err
	}

	if errList := metalvalidation.ValidateInfrastructureConfigAgainstCloudProfile(infraConfig, shoot, cloudProfile, cloudProfileConfig, fldPath); len(errList) != 0 {
		return errList.ToAggregate()
	}

	return nil
}
