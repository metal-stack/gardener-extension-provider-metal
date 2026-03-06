package shoot

import (
	"context"
	"fmt"
	"slices"
	"strconv"

	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"

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

	switch x := new.(type) {
	case *appsv1.DaemonSet:
		switch x.Name {
		case "calico-node":
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateCalicoNode(ctx, x)
		}
	}
	return nil
}

func (m *mutator) mutateCalicoNode(_ context.Context, ds *appsv1.DaemonSet) error {
	if c := extensionswebhook.ContainerWithName(ds.Spec.Template.Spec.Containers, "calico-node"); c != nil {
		ebpfEnabled := slices.ContainsFunc(c.Env, func(e corev1.EnvVar) bool {
			if e.Name != "FELIX_BPFENABLED" {
				return false
			}

			enabled, _ := strconv.ParseBool(e.Value)

			return enabled
		})

		if !ebpfEnabled {
			return nil
		}

		m.logger.Info("patching calico-node daemon set due to ebpf dataplane being enabled")

		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, corev1.EnvVar{
			Name: "FELIX_BPFDATAIFACEPATTERN",
			// including "lan" interface name to default value
			// (see https://github.com/projectcalico/calico/blob/3f7fe4d290541bbdd73c97bdc89a29a29855a48a/felix/config/config_params.go#L180)
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
