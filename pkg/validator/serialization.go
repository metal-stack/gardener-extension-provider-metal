package validator

import (
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/gardener/gardener/pkg/apis/core"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func decodeControlPlaneConfig(decoder runtime.Decoder, cp *core.ProviderConfig, fldPath *field.Path) (*metal.ControlPlaneConfig, error) {
	controlPlaneConfig := &metal.ControlPlaneConfig{}
	if cp != nil && cp.Raw != nil {
		if err := util.Decode(decoder, cp.Raw, controlPlaneConfig); err != nil {
			return nil, field.Invalid(fldPath, string(cp.Raw), "isn't a supported version")
		}
	}

	return controlPlaneConfig, nil
}

func decodeInfrastructureConfig(decoder runtime.Decoder, infra *core.ProviderConfig, fldPath *field.Path) (*metal.InfrastructureConfig, error) {
	infraConfig := &metal.InfrastructureConfig{}
	if infra != nil && infra.Raw != nil {
		if err := util.Decode(decoder, infra.Raw, infraConfig); err != nil {
			return nil, field.Invalid(fldPath, string(infra.Raw), "isn't a supported version")
		}
	}

	return infraConfig, nil
}
