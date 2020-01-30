package helper

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/util"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	api "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"

	"github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/install"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	// Scheme is a scheme with the types relevant for OpenStack actuators.
	Scheme *runtime.Scheme

	decoder runtime.Decoder
)

func init() {
	Scheme = runtime.NewScheme()
	utilruntime.Must(install.AddToScheme(Scheme))

	decoder = serializer.NewCodecFactory(Scheme).UniversalDecoder()
}

// InfrastructureConfigFromInfrastructure extracts the InfrastructureConfig from the
// ProviderConfig section of the given Infrastructure.
func InfrastructureConfigFromInfrastructure(infra *extensionsv1alpha1.Infrastructure) (*api.InfrastructureConfig, error) {
	config := &api.InfrastructureConfig{}
	if infra.Spec.ProviderConfig != nil && infra.Spec.ProviderConfig.Raw != nil {
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
	return nil, fmt.Errorf("provider config is not set on the control plane resource")
}

// CloudProfileConfigFromCluster decodes the provider specific cloud profile configuration for a cluster
func CloudProfileConfigFromCluster(cluster *controller.Cluster) (*api.CloudProfileConfig, error) {
	var cloudProfileConfig *api.CloudProfileConfig
	if cluster != nil && cluster.CloudProfile != nil && cluster.CloudProfile.Spec.ProviderConfig != nil && cluster.CloudProfile.Spec.ProviderConfig.Raw != nil {
		cloudProfileConfig = &api.CloudProfileConfig{}
		if _, _, err := decoder.Decode(cluster.CloudProfile.Spec.ProviderConfig.Raw, nil, cloudProfileConfig); err != nil {
			return nil, errors.Wrapf(err, "could not decode providerConfig of cloudProfile for '%s'", util.ObjectName(cluster.CloudProfile))
		}
	}
	return cloudProfileConfig, nil
}
