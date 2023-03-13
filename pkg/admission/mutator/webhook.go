package mutator

import (
	"github.com/gardener/gardener-extension-networking-calico/pkg/calico"
	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	Name = "mutator"
)

var logger = log.Log.WithName("metal-mutator-webhook")

// New creates a new webhook that mutates Shoot resources.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Setting up webhook", "name", Name)

	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider:   metal.Type,
		Name:       Name,
		Path:       "/webhooks/mutate",
		Predicates: []predicate.Predicate{extensionspredicate.GardenCoreProviderType(metal.Type), createMetalPredicate()},
		Mutators: map[extensionswebhook.Mutator][]extensionswebhook.Type{
			NewShootMutator(): {{Obj: &gardencorev1beta1.Shoot{}}},
		},
	})
}

func createMetalPredicate() predicate.Funcs {
	f := func(obj client.Object) bool {
		if obj == nil {
			return false
		}

		shoot, ok := obj.(*gardencorev1beta1.Shoot)
		if !ok {
			return false
		}

		return shoot.Spec.Networking.Type == calico.ReleaseName
	}

	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return f(event.Object)
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return f(event.ObjectNew)
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return f(event.Object)
		},
		DeleteFunc: func(event event.DeleteEvent) bool {
			return f(event.Object)
		},
	}
}
