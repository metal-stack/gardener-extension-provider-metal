package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	extensionscontextwebhook "github.com/gardener/gardener/extensions/pkg/webhook/context"
	gcontext "github.com/gardener/gardener/extensions/pkg/webhook/context"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"

	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type mutator struct {
	client  client.Client
	decoder runtime.Decoder
	logger  logr.Logger
}

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator(mgr manager.Manager) extensionswebhook.Mutator {
	return &mutator{
		logger:  log.Log.WithName("shoot-mutator"),
		client:  mgr.GetClient(),
		decoder: serializer.NewCodecFactory(mgr.GetScheme(), serializer.EnableStrict).UniversalDecoder(),
	}
}

func (m *mutator) Mutate(ctx context.Context, new, _ client.Object) error {
	acc, err := meta.Accessor(new)
	if err != nil {
		return fmt.Errorf("could not create accessor during webhook %w", err)
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}

	gctx := extensionscontextwebhook.NewGardenContext(m.client, new)

	switch x := new.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		case "vpn-shoot":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateVPNShootDeployment(ctx, x)

		}
	case *appsv1.DaemonSet:
		switch x.Name {
		case "calico-node":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateCalicoNode(ctx, gctx, x)
		}
	}
	return nil
}

func (m *mutator) mutateVPNShootDeployment(_ context.Context, deployment *appsv1.Deployment) error {
	if c := extensionswebhook.ContainerWithName(deployment.Spec.Template.Spec.Containers, "vpn-shoot"); c != nil {
		// fixes a regression from https://github.com/gardener/gardener/pull/4691
		// raising the timeout to 15 minutes leads to additional 15 minutes of provisioning time because
		// the nodes cidr will only be set on next shoot reconcile
		// with the following mutation we can immediately provide the proper nodes cidr and save time
		m.logger.Info("ensuring nodes cidr from shoot-node-cidr configmap in vpn-shoot deployment")
		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name:  "NODE_NETWORK",
			Value: "",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "shoot-info-node-cidr",
					},
					Key: "node-cidr",
				},
			},
		})
	}

	return nil
}

func (m *mutator) mutateCalicoNode(ctx context.Context, gctx gcontext.GardenContext, ds *appsv1.DaemonSet) error {
	cluster, err := gctx.GetCluster(ctx)
	if err != nil {
		return fmt.Errorf("unable to extract cluster from ctx: %w", err)
	}

	networkConfig := &calicoextensionv1alpha1.NetworkConfig{}
	err = helper.DecodeRawExtension(cluster.Shoot.Spec.Networking.ProviderConfig, networkConfig, m.decoder)
	if err != nil {
		return err
	}

	if networkConfig.EbpfDataplane == nil || !networkConfig.EbpfDataplane.Enabled {
		return nil
	}

	if c := extensionswebhook.ContainerWithName(ds.Spec.Template.Spec.Containers, "calico-node"); c != nil {
		m.logger.Info("patching calico-node daemon set due to ebpf dataplane being enabled")

		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name: "FELIX_BPFDATAIFACEPATTERN",
			// including "lan" interface name to default value
			Value: "^((en|wl|ww|sl|ib)[Popsx].*|(lan|eth|wlan|wwan).*|tunl0$|vxlan.calico$|wireguard.cali$|wg-v6.cali$)",
		})

		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name:  "FELIX_BPFEXTERNALSERVICEMODE",
			Value: "DSR",
		})

		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name:  "FELIX_MTUIFACEPATTERN",
			Value: "lan",
		})
	}

	return nil
}
