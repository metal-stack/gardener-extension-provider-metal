package mutator

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewShootMutator returns a new instance of a shoot mutator.
func NewShootMutator() extensionswebhook.Mutator {
	return &mutator{}
}

type mutator struct {
	client  client.Client
	decoder runtime.Decoder
}

// InjectScheme injects the given scheme into the validator.
func (m *mutator) InjectScheme(scheme *runtime.Scheme) error {
	m.decoder = serializer.NewCodecFactory(scheme, serializer.EnableStrict).UniversalDecoder()
	return nil
}

// InjectClient injects the given client into the mutator.
func (s *mutator) InjectClient(client client.Client) error {
	s.client = client
	return nil
}

// Mutate mutates the given shoot object.
func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	shoot, ok := new.(*gardenv1beta1.Shoot)
	if !ok {
		return fmt.Errorf("wrong object type %T", new)
	}

	profile := &gardenv1beta1.CloudProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: shoot.Spec.CloudProfileName,
		},
	}
	if err := m.client.Get(ctx, kutil.Key(shoot.Spec.CloudProfileName), profile); err != nil {
		return err
	}

	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	err := helper.DecodeRawExtension(shoot.Spec.Provider.InfrastructureConfig, infrastructureConfig, m.decoder)
	if err != nil {
		return err
	}

	cloudConfig := &metal.CloudProfileConfig{}
	err = helper.DecodeRawExtension(profile.Spec.ProviderConfig, cloudConfig, m.decoder)
	if err != nil {
		return err
	}

	controlPlane, partition, err := helper.FindMetalControlPlane(cloudConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	d := defaulter{
		decoder:      m.decoder,
		controlPlane: controlPlane,
		partition:    partition,
	}

	return d.defaultShoot(shoot)
}
