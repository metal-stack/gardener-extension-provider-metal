package validator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalvalidation "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewCloudProfileValidator returns a new instance of a cloud profile validator.
func NewCloudProfileValidator() extensionswebhook.Validator {
	return &cloudProfile{}
}

type cloudProfile struct {
	decoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (cp *cloudProfile) InjectScheme(scheme *runtime.Scheme) error {
	cp.decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
	return nil
}

// Validate validates the given cloud profile objects.
func (cp *cloudProfile) Validate(_ context.Context, new, old client.Object) error {
	cloudProfile, ok := new.(*core.CloudProfile)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	providerConfigPath := field.NewPath("spec").Child("providerConfig")
	if cloudProfile.Spec.ProviderConfig == nil {
		return field.Required(providerConfigPath, "providerConfig must be set for metal cloud profiles")
	}

	cpConfig := &metal.CloudProfileConfig{}
	err := helper.DecodeRawExtension(cloudProfile.Spec.ProviderConfig, cpConfig, cp.decoder)
	if err != nil {
		return err
	}

	errs := metalvalidation.ValidateCloudProfileConfig(cpConfig, cloudProfile, providerConfigPath)
	if old == nil {
		return errs.ToAggregate()
	}

	oldCloudProfile, ok := old.(*core.CloudProfile)
	if !ok {
		return fmt.Errorf("wrong old object type %T", new)
	}

	if oldCloudProfile.Spec.ProviderConfig == nil {
		return errs.ToAggregate()
	}

	oldCpConfig := &metal.CloudProfileConfig{}
	err = helper.DecodeRawExtension(oldCloudProfile.Spec.ProviderConfig, oldCpConfig, cp.decoder)
	if err != nil {
		return err
	}

	errs = append(errs, metalvalidation.ValidateImmutableCloudProfileConfig(cpConfig, oldCpConfig, providerConfigPath)...)
	return errs.ToAggregate()
}
