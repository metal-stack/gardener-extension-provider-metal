package mutator

import (
	"context"
	"encoding/json"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"

	"github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewShootMutator returns a new instance of a shoot mutator.
func NewShootMutator() extensionswebhook.Mutator {
	return &shoot{}
}

type shoot struct {
	decoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (s *shoot) InjectScheme(scheme *runtime.Scheme) error {
	s.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// Mutate mutates the given shoot object.
func (s *shoot) Mutate(ctx context.Context, new, old client.Object) error {
	// overlay := &calicov1alpha1.Overlay{Enabled: false}

	shoot, ok := new.(*gardenv1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	// source/destination checks are only disabled for kubernetes >= 1.22
	// see https://github.com/gardener/machine-controller-manager-provider-aws/issues/36 for details
	// greaterEqual122, err := versionutils.CompareVersions(shoot.Spec.Kubernetes.Version, ">=", "1.22")
	// if err != nil {
	// 	return err
	// }
	// if !greaterEqual122 {
	// 	return nil
	// }

	networkConfig, err := s.decodeNetworkingConfig(shoot.Spec.Networking.ProviderConfig)
	if err != nil {
		return err
	}

	// if old == nil && networkConfig.Overlay == nil {
	// 	networkConfig.Overlay = overlay
	// }

	// if old != nil && networkConfig.Overlay == nil {
	// 	oldShoot, ok := old.(*gardencorev1beta1.Shoot)
	// 	if !ok {
	// 		return fmt.Errorf("wrong object type %T", old)
	// 	}
	// 	if oldShoot.DeletionTimestamp != nil {
	// 		return nil
	// 	}
	// 	oldNetworkConfig, err := s.decodeNetworkingConfig(oldShoot.Spec.Networking.ProviderConfig)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if oldNetworkConfig.Overlay != nil {
	// 		networkConfig.Overlay = oldNetworkConfig.Overlay
	// 	}
	// }

	shoot.Spec.Networking.ProviderConfig = &runtime.RawExtension{
		Object: networkConfig,
	}

	controlPlaneConfig, err := s.decodeControlplaneConfig(shoot.Spec.Provider.ControlPlaneConfig)
	if err != nil {
		return err
	}

	if controlPlaneConfig.CloudControllerManager == nil {
		controlPlaneConfig.CloudControllerManager = &metalv1alpha1.CloudControllerManagerConfig{}
	}

	// if networkConfig.Overlay != nil && !networkConfig.Overlay.Enabled {
	// 	if controlPlaneConfig.CloudControllerManager.UseCustomRouteController == nil {
	// 		controlPlaneConfig.CloudControllerManager.UseCustomRouteController = pointer.Bool(true)
	// 	} else {
	// 		*controlPlaneConfig.CloudControllerManager.UseCustomRouteController = true
	// 	}
	// } else {
	// 	if controlPlaneConfig.CloudControllerManager.UseCustomRouteController == nil {
	// 		controlPlaneConfig.CloudControllerManager.UseCustomRouteController = pointer.Bool(false)
	// 	} else {
	// 		*controlPlaneConfig.CloudControllerManager.UseCustomRouteController = false
	// 	}
	// }

	shoot.Spec.Provider.ControlPlaneConfig = &runtime.RawExtension{
		Object: controlPlaneConfig,
	}

	infrastructureConfig, err := s.decodeInfrastructureConfig(shoot.Spec.Provider.InfrastructureConfig)
	if err != nil {
		return err
	}

	infrastructureConfig.Firewall = metalv1alpha1.Firewall{
		Image:    "firewall-ubuntu-2.0.20221025",
		Networks: []string{"internet"},
		Size:     "n1-medium-x86",
	}

	shoot.Spec.Provider.InfrastructureConfig = &runtime.RawExtension{
		Object: infrastructureConfig,
	}

	backend := calico.None
	networkingConfig, err := encodeProviderConfig(calico.NetworkConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: calico.SchemeGroupVersion.String(),
			Kind:       "NetworkConfig",
		},
		Backend: &backend,
		Typha: &calico.Typha{
			Enabled: false,
		},
	})
	if err != nil {
		return err
	}

	shoot.Spec.CloudProfileName = "metal"
	shoot.Spec.SecretBindingName = "seed-provider-secret"
	shoot.Spec.Networking = gardenv1beta1.Networking{
		Type:           "calico",
		ProviderConfig: networkingConfig,
	}

	return nil
}

func (s *shoot) decodeInfrastructureConfig(config *runtime.RawExtension) (*metalv1alpha1.InfrastructureConfig, error) {
	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	if config != nil && config.Raw != nil {
		if _, _, err := s.decoder.Decode(config.Raw, nil, infrastructureConfig); err != nil {
			return nil, err
		}
	}

	return infrastructureConfig, nil
}

func (s *shoot) decodeNetworkingConfig(network *runtime.RawExtension) (*calico.NetworkConfig, error) {
	networkConfig := &calico.NetworkConfig{}
	if network != nil && network.Raw != nil {
		if _, _, err := s.decoder.Decode(network.Raw, nil, networkConfig); err != nil {
			return nil, err
		}
	}
	return networkConfig, nil
}

func (s *shoot) decodeControlplaneConfig(controlPlaneConfig *runtime.RawExtension) (*metalv1alpha1.ControlPlaneConfig, error) {
	cp := &metalv1alpha1.ControlPlaneConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
			Kind:       "ControlPlaneConfig",
		},
	}
	if controlPlaneConfig != nil && controlPlaneConfig.Raw != nil {
		if _, _, err := s.decoder.Decode(controlPlaneConfig.Raw, nil, cp); err != nil {
			return nil, err
		}
	}
	return cp, nil
}

func encodeProviderConfig(from any) (*runtime.RawExtension, error) {
	encoded, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	return &runtime.RawExtension{
		Raw: encoded,
	}, nil
}
