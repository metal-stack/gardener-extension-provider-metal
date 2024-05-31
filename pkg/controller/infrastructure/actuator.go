package infrastructure

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type actuator struct {
	restConfig *rest.Config
	client     client.Client
	decoder    runtime.Decoder
}

// NewActuator creates a new Actuator that updates the status of the handled Infrastructure resources.
func NewActuator(mgr manager.Manager) infrastructure.Actuator {
	return &actuator{
		client:     mgr.GetClient(),
		decoder:    serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
		restConfig: mgr.GetConfig(),
	}
}

func decodeInfrastructure(infrastructure *extensionsv1alpha1.Infrastructure, decoder runtime.Decoder) (*metalapi.InfrastructureConfig, *metalapi.InfrastructureStatus, error) {
	infrastructureConfig, err := helper.InfrastructureConfigFromInfrastructure(infrastructure)
	if err != nil {
		return nil, nil, err
	}

	infrastructureStatus := &metalapi.InfrastructureStatus{}
	if infrastructure.Status.ProviderStatus != nil {
		if _, _, err := decoder.Decode(infrastructure.Status.ProviderStatus.Raw, nil, infrastructureStatus); err != nil {
			return nil, nil, fmt.Errorf("could not decode infrastructure status: %w", err)
		}
	}

	return infrastructureConfig, infrastructureStatus, nil
}

func updateProviderStatus(ctx context.Context, c client.Client, infrastructure *extensionsv1alpha1.Infrastructure, providerStatus *metalapi.InfrastructureStatus, nodeCIDR *string) error {
	patch := client.MergeFrom(infrastructure.DeepCopy())
	infrastructure.Status.ProviderStatus = &runtime.RawExtension{Object: &metalv1alpha1.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
		Firewall: metalv1alpha1.FirewallStatus{
			MachineID: providerStatus.Firewall.MachineID,
		},
	}}
	infrastructure.Status.NodesCIDR = nodeCIDR
	return c.Status().Patch(ctx, infrastructure, patch)
}
