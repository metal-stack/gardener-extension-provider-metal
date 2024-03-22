package controlplane

import (
	"context"
	"errors"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener/extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/original/components/kubelet"
	oscutils "github.com/gardener/gardener/pkg/component/extensions/operatingsystemconfig/utils"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
	logger            = log.Log.WithName("metal-controlplane-webhook")
)

// AddOptions are options to apply when adding the metal infrastructure controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	ControllerConfig config.ControllerConfiguration
}

func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) (*extensionswebhook.Webhook, error) {
	logger.Info("Adding webhook to manager")
	return controlplane.New(mgr, controlplane.Args{
		Kind:     controlplane.KindShoot,
		Provider: metal.Type,
		Types: []extensionswebhook.Type{
			{Obj: &appsv1.Deployment{}},
			{Obj: &extensionsv1alpha1.OperatingSystemConfig{}},
		},
		Mutator: newMutator(mgr, opts.ControllerConfig),
	})
}

// AddToManager creates a webhook and adds it to the manager.
func AddToManager(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}

type mutator struct {
	logger          logr.Logger
	client          client.Client
	decoder         runtime.Decoder
	gardenerMutator extensionswebhook.Mutator
}

func newMutator(mgr manager.Manager, c config.ControllerConfiguration) extensionswebhook.Mutator {
	fciCodec := oscutils.NewFileContentInlineCodec()

	gardenerMutator := genericmutator.NewMutator(mgr, NewEnsurer(mgr, logger, c), oscutils.NewUnitSerializer(),
		kubelet.NewConfigCodec(fciCodec), fciCodec, logger)

	return &mutator{
		logger:          logger,
		gardenerMutator: gardenerMutator,
		client:          mgr.GetClient(),
		decoder:         serializer.NewCodecFactory(mgr.GetScheme()).UniversalDecoder(),
	}
}

func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if new.GetDeletionTimestamp() != nil {
		return nil
	}
	gctx := extensionscontextwebhook.NewGardenContext(m.client, new)

	switch x := new.(type) {
	case *extensionsv1alpha1.OperatingSystemConfig:
		var oldOSC *extensionsv1alpha1.OperatingSystemConfig
		if old != nil {
			var ok bool
			oldOSC, ok = old.(*extensionsv1alpha1.OperatingSystemConfig)
			if !ok {
				return errors.New("could not cast old object to extensionsv1alpha1.OperatingSystemConfig")
			}
		}

		extensionswebhook.LogMutation(m.logger, x.Kind, x.Namespace, x.Name)

		err := m.mutateOperatingSystemConfig(ctx, gctx, x, oldOSC)
		if err != nil {
			return err
		}
	}

	return m.gardenerMutator.Mutate(ctx, new, old)
}

func (m *mutator) mutateOperatingSystemConfig(ctx context.Context, gctx gcontext.GardenContext, osc, _ *extensionsv1alpha1.OperatingSystemConfig) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return err
	}

	cloudProfileConfig, err := helper.DecodeCloudProfileConfig(cluster.CloudProfile)
	if err != nil {
		return err
	}

	infrastructureConfig := &metalv1alpha1.InfrastructureConfig{}
	err = helper.DecodeRawExtension(cluster.Shoot.Spec.Provider.InfrastructureConfig, infrastructureConfig, m.decoder)
	if err != nil {
		return err
	}

	_, p, err := helper.FindMetalControlPlane(cloudProfileConfig, infrastructureConfig.PartitionID)
	if err != nil {
		return err
	}

	controlPlaneConfig := &metalv1alpha1.ControlPlaneConfig{}
	err = helper.DecodeRawExtension(cluster.Shoot.Spec.Provider.ControlPlaneConfig, controlPlaneConfig, m.decoder)
	if err != nil {
		return err
	}

	if controlPlaneConfig.NetworkAccessType == nil || *controlPlaneConfig.NetworkAccessType == metalv1alpha1.NetworkAccessBaseline {
		// this shoot does not have networkaccesstype restricted or forbidden specified, nothing to do here
		return nil
	}

	if p.NetworkIsolation == nil {
		return nil
	}

	var mirrors []metalv1alpha1.RegistryMirror
	for _, m := range p.NetworkIsolation.RegistryMirrors {
		m := m

		mirrors = append(mirrors, metalv1alpha1.RegistryMirror{
			Name:     m.Name,
			Endpoint: m.Endpoint,
			IP:       m.IP,
			Port:     m.Port,
			MirrorOf: m.MirrorOf,
		})
	}

	encoded, err := helper.EncodeRawExtension(&metalv1alpha1.ImageProviderConfig{
		TypeMeta: v1.TypeMeta{
			Kind:       "ImageProviderConfig",
			APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
		},
		NetworkIsolation: &metalv1alpha1.NetworkIsolation{
			AllowedNetworks: metalv1alpha1.AllowedNetworks{
				Ingress: p.NetworkIsolation.AllowedNetworks.Ingress,
				Egress:  p.NetworkIsolation.AllowedNetworks.Egress,
			},
			DNSServers:      p.NetworkIsolation.DNSServers,
			NTPServers:      p.NetworkIsolation.NTPServers,
			RegistryMirrors: mirrors,
		},
	})
	if err != nil {
		return err
	}

	osc.Spec.ProviderConfig = encoded

	return nil
}
