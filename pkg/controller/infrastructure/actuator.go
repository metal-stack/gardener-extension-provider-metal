package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/controller/infrastructure"
	metalapi "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
)

// InfrastructureState represents the last known State of an Infrastructure resource.
// It is saved after a reconciliation and used during restore operations.
type InfrastructureState struct {
	// Firewalls contains the running firewalls.
	Firewalls []fcmv2.Firewall `json:"firewalls"`
}

type actuator struct {
	logger logr.Logger

	clientset         kubernetes.Interface
	gardenerClientset gardenerkubernetes.Interface
	restConfig        *rest.Config

	client  client.Client
	scheme  *runtime.Scheme
	decoder runtime.Decoder
}

// NewActuator creates a new Actuator that updates the status of the handled Infrastructure resources.
func NewActuator() infrastructure.Actuator {
	return &actuator{
		logger: log.Log.WithName("infrastructure-actuator"),
	}
}

func (a *actuator) InjectScheme(scheme *runtime.Scheme) error {
	a.scheme = scheme
	a.decoder = serializer.NewCodecFactory(a.scheme).UniversalDecoder()
	return nil
}

func (a *actuator) InjectClient(client client.Client) error {
	a.client = client
	return nil
}

func (a *actuator) InjectConfig(config *rest.Config) error {
	var err error
	a.clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("could not create Kubernetes client %w", err)
	}

	a.gardenerClientset, err = gardenerkubernetes.NewWithConfig(gardenerkubernetes.WithRESTConfig(config))
	if err != nil {
		return fmt.Errorf("could not create Gardener client %w", err)
	}

	a.restConfig = config
	return nil
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

	firewalls := &fcmv2.FirewallList{}
	err := c.List(ctx, firewalls, client.InNamespace(infrastructure.Namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewalls: %w", err)
	}

	infraState := &InfrastructureState{
		Firewalls: firewalls.Items,
	}

	infraStateBytes, err := json.Marshal(infraState)
	if err != nil {
		return err
	}

	infrastructure.Status.State = &runtime.RawExtension{Raw: infraStateBytes}
	infrastructure.Status.NodesCIDR = nodeCIDR
	infrastructure.Status.ProviderStatus = &runtime.RawExtension{Object: &metalv1alpha1.InfrastructureStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
			Kind:       "InfrastructureStatus",
		},
		Firewall: metalv1alpha1.FirewallStatus{
			MachineID: providerStatus.Firewall.MachineID,
		},
	}}

	return c.Status().Patch(ctx, infrastructure, patch)
}
