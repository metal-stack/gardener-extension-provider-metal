package controlplane

import (
	"context"
	"fmt"
	"slices"
	"strings"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
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
	grmv1alpha1 "github.com/gardener/gardener/pkg/resourcemanager/apis/config/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"sigs.k8s.io/yaml"
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
			{Obj: &corev1.ConfigMap{}},
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

	scheme := mgr.GetScheme()
	err := grmv1alpha1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}

	return &mutator{
		logger:          logger,
		gardenerMutator: gardenerMutator,
		client:          mgr.GetClient(),
		decoder:         serializer.NewCodecFactory(scheme).UniversalDecoder(),
	}
}

func (m *mutator) Mutate(ctx context.Context, new, old client.Object) error {
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if new.GetDeletionTimestamp() != nil {
		return nil
	}
	gctx := gcontext.NewGardenContext(m.client, new)

	switch x := new.(type) {
	case *extensionsv1alpha1.OperatingSystemConfig:
		extensionswebhook.LogMutation(m.logger, x.Kind, x.Namespace, x.Name)

		o, _ := old.(*extensionsv1alpha1.OperatingSystemConfig)

		err := m.mutateOperatingSystemConfig(ctx, gctx, x, o)
		if err != nil {
			return err
		}
	case *corev1.ConfigMap:
		if strings.HasPrefix(x.Name, "gardener-resource-manager-") {
			// hopefully this whole mutation can be removed in a future version of Gardener where
			// the namespaces are not hard-coded for the GRM
			extensionswebhook.LogMutation(m.logger, x.Kind, x.Namespace, x.Name)

			err := m.mutateResourceManagerConfigMap(ctx, gctx, x)
			if err != nil {
				return err
			}
		}
	}

	return m.gardenerMutator.Mutate(ctx, new, old)
}

func (m *mutator) mutateOperatingSystemConfig(ctx context.Context, gctx gcontext.GardenContext, osc, oldOSC *extensionsv1alpha1.OperatingSystemConfig) error {
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

	var dnsServers, ntpServers []string
	if oldOSC != nil {
		// this is required for backwards-compatibility before we started to create worker machines with DNS and NTP configuration through metal-stack
		// otherwise existing machines would lose connectivity because the GNA cleans up the dns and ntp definitions
		// references https://github.com/metal-stack/gardener-extension-provider-metal/issues/433
		//
		// can potentially be cleaned up as soon as there are no worker nodes of isolated clusters anymore that were created without dns and ntp configuration
		// ideally a point in time should be defined when we add the dns and ntp to the worker hashes to enforce the setting
		for _, path := range []string{
			"/etc/systemd/resolved.conf.d/dns.conf",
			"/etc/resolv.conf",
			"/etc/systemd/timesyncd.conf",
		} {
			if idx := slices.IndexFunc(oldOSC.Status.ExtensionFiles, func(f extensionsv1alpha1.File) bool {
				return f.Path == path
			}); idx >= 0 {
				dnsServers = p.NetworkIsolation.DNSServers
				ntpServers = p.NetworkIsolation.NTPServers
				break
			}
		}
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
			DNSServers:      dnsServers,
			NTPServers:      ntpServers,
			RegistryMirrors: mirrors,
		},
	})
	if err != nil {
		return err
	}

	osc.Spec.ProviderConfig = encoded

	return nil
}

func (m *mutator) mutateResourceManagerConfigMap(_ context.Context, _ gcontext.GardenContext, cm *corev1.ConfigMap) error {
	const configKey = "config.yaml"

	raw, ok := cm.Data[configKey]
	if !ok {
		return fmt.Errorf("gardener-resource-manager config map does not contain config.yaml key")
	}

	config := &grmv1alpha1.ResourceManagerConfiguration{}

	_, _, err := m.decoder.Decode([]byte(raw), nil, config)
	if err != nil {
		return fmt.Errorf("unable to decode gardener-resource-manager configuration: %w", err)
	}

	// TODO: audit is actually used by the gardener-extension-audit but this extension can be toggled on and off
	// so actually need to resolve the problem differently: https://github.com/metal-stack/gardener-extension-audit/issues/24
	config.TargetClientConnection.Namespaces = append(config.TargetClientConnection.Namespaces, "firewall", "metallb-system", "csi-lvm", "audit")

	encoded, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("unable to encode gardener-resource-manager configuration: %w", err)
	}

	cm.Data[configKey] = string(encoded)

	return nil
}
