package helper

import (
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	api "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/install"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	// scheme is a scheme with the types relevant for metal actuators.
	scheme *runtime.Scheme

	decoder runtime.Decoder
)

func init() {
	scheme = runtime.NewScheme()

	install.Install(scheme)

	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()
}

// DecodeCloudProfileConfig decodes the cloud profile config
func DecodeCloudProfileConfig(cloudProfile *gardencorev1beta1.CloudProfile) (*api.CloudProfileConfig, error) {
	var cloudProfileConfig *api.CloudProfileConfig
	if cloudProfile != nil && cloudProfile.Spec.ProviderConfig != nil && cloudProfile.Spec.ProviderConfig.Raw != nil {
		cloudProfileConfig = &api.CloudProfileConfig{}
		if _, _, err := decoder.Decode(cloudProfile.Spec.ProviderConfig.Raw, nil, cloudProfileConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of cloudProfile for %q %w", cloudProfile.Name, err)
		}
	}
	return cloudProfileConfig, nil
}

// InfrastructureConfigFromInfrastructure extracts the InfrastructureConfig from the
// ProviderConfig section of the given Infrastructure.
func InfrastructureConfigFromInfrastructure(infra *extensionsv1alpha1.Infrastructure) (*api.InfrastructureConfig, error) {
	if infra.Spec.ProviderConfig != nil && infra.Spec.ProviderConfig.Raw != nil {
		config := &api.InfrastructureConfig{}
		if _, _, err := decoder.Decode(infra.Spec.ProviderConfig.Raw, nil, config); err != nil {
			return nil, err
		}
		return config, nil
	}
	return nil, fmt.Errorf("provider config is not set on the infrastructure resource")
}

// ControlPlaneConfigFromControlPlane extracts the ControlPlaneConfig from the
// ProviderConfig section of the given ControlPlane.
func ControlPlaneConfigFromControlPlane(cp *extensionsv1alpha1.ControlPlane) (*api.ControlPlaneConfig, error) {
	config := &api.ControlPlaneConfig{}
	if cp.Spec.ProviderConfig != nil && cp.Spec.ProviderConfig.Raw != nil {
		if _, _, err := decoder.Decode(cp.Spec.ProviderConfig.Raw, nil, config); err != nil {
			return nil, err
		}
		return config, nil
	}
	return config, nil
}

// ControlPlaneConfigFromClusterShootSpec extracts the ControlPlaneConfig from the shoot spec of a given cluster.
func ControlPlaneConfigFromClusterShootSpec(cluster *controller.Cluster) (*api.ControlPlaneConfig, error) {
	config := &api.ControlPlaneConfig{}
	if cluster != nil && cluster.Shoot != nil && cluster.Shoot.Spec.Provider.ControlPlaneConfig != nil && cluster.Shoot.Spec.Provider.ControlPlaneConfig.Raw != nil {
		if _, _, err := decoder.Decode(cluster.Shoot.Spec.Provider.ControlPlaneConfig.Raw, nil, config); err != nil {
			return nil, err
		}
		return config, nil
	}
	return config, nil
}

// CloudProfileConfigFromCluster decodes the provider specific cloud profile configuration for a cluster
func CloudProfileConfigFromCluster(cluster *controller.Cluster) (*api.CloudProfileConfig, error) {
	var cloudProfileConfig *api.CloudProfileConfig
	if cluster != nil && cluster.CloudProfile != nil && cluster.CloudProfile.Spec.ProviderConfig != nil && cluster.CloudProfile.Spec.ProviderConfig.Raw != nil {
		cloudProfileConfig = &api.CloudProfileConfig{}
		if _, _, err := decoder.Decode(cluster.CloudProfile.Spec.ProviderConfig.Raw, nil, cloudProfileConfig); err != nil {
			return nil, fmt.Errorf("could not decode providerConfig of cloudProfile for '%s' %w", cluster.CloudProfile.Name, err)
		}
	}
	return cloudProfileConfig, nil
}

// DecodeRawExtension decodes a raw extension into an object
func DecodeRawExtension[T runtime.Object](extension *runtime.RawExtension, object T, decoder runtime.Decoder) error {
	if extension != nil && extension.Raw != nil {
		if _, _, err := decoder.Decode(extension.Raw, nil, object); err != nil {
			return err
		}
	}
	return nil
}

// EncodeRawExtension encodes an object into a raw extension
func EncodeRawExtension(from runtime.Object) (*runtime.RawExtension, error) {
	encoded, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: encoded,
	}, nil
}
