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
	"sigs.k8s.io/yaml"

	fcmv2 "github.com/metal-stack/firewall-controller-manager/api/v2"
	corev1 "k8s.io/api/core/v1"
)

// InfrastructureState represents the last known State of an Infrastructure resource.
// It is saved after a reconciliation and used during restore operations.
// We use this for restoring firewalls, which are actually maintained by the worker controller
// because the worker controller does not allow adding our state to the worker resource as it
// is used by the MCM already.
type InfrastructureState struct {
	// Firewalls contains the running firewalls.
	Firewalls []string `json:"firewalls"`

	SeedAccess []SeedAccessState `json:"seedAccess"`
}

type SeedAccessState struct {
	ServiceAccount        string   `json:"serviceAccount"`
	ServiceAccountSecrets []string `json:"serviceAccountSecrets"`
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

func DecodeInfrastructure(infrastructure *extensionsv1alpha1.Infrastructure, decoder runtime.Decoder) (*metalapi.InfrastructureConfig, *metalapi.InfrastructureStatus, error) {
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

func UpdateProviderStatus(ctx context.Context, c client.Client, infrastructure *extensionsv1alpha1.Infrastructure, providerStatus *metalapi.InfrastructureStatus, nodeCIDR *string) error {
	patch := client.MergeFrom(infrastructure.DeepCopy())

	var (
		namespace = infrastructure.Namespace

		infraState = &InfrastructureState{}
		fwdeploys  = &fcmv2.FirewallDeploymentList{}
		firewalls  = &fcmv2.FirewallList{}
	)

	err := c.List(ctx, firewalls, client.InNamespace(infrastructure.Namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewalls: %w", err)
	}

	for _, fw := range firewalls.Items {
		fw := fw

		fw.ResourceVersion = ""
		fw.OwnerReferences = nil

		raw, err := yaml.Marshal(fw)
		if err != nil {
			return err
		}

		infraState.Firewalls = append(infraState.Firewalls, string(raw))
	}

	err = c.List(ctx, fwdeploys, client.InNamespace(infrastructure.Namespace))
	if err != nil {
		return fmt.Errorf("unable to list firewall deployments: %w", err)
	}

	for _, fwdeploy := range fwdeploys.Items {
		saName := fmt.Sprintf("firewall-controller-seed-access-%s", fwdeploy.Name) // TODO: name should be exposed by fcm
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      saName,
				Namespace: namespace,
			},
		}

		err := c.Get(ctx, client.ObjectKeyFromObject(sa), sa)
		if err != nil {
			continue
		}

		secrets := []string{}

		for _, ref := range sa.Secrets {

			saSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ref.Name,
					Namespace: namespace,
				},
			}

			err = c.Get(ctx, client.ObjectKeyFromObject(saSecret), saSecret)
			if err != nil {
				return fmt.Errorf("error getting service account secret: %w", err)
			}

			saSecret.ResourceVersion = ""

			raw, err := yaml.Marshal(*saSecret)
			if err != nil {
				return err
			}

			secrets = append(secrets, string(raw))
		}

		sa.ResourceVersion = ""

		raw, err := yaml.Marshal(*sa)
		if err != nil {
			return err
		}

		infraState.SeedAccess = append(infraState.SeedAccess, SeedAccessState{
			ServiceAccount:        string(raw),
			ServiceAccountSecrets: secrets,
		})
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
