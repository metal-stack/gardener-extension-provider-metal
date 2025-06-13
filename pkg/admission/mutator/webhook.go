package mutator

import (
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	Name = "mutator"
)

var logger = log.Log.WithName("metal-mutator-webhook")

// New creates a new webhook that mutates Shoot resources.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Setting up webhook", "name", Name)

	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider: metal.Type,
		Name:     Name,
		Path:     "/webhooks/mutate",
		Mutators: map[extensionswebhook.Mutator][]extensionswebhook.Type{
			NewShootMutator(mgr): {{Obj: &gardencorev1beta1.Shoot{}}},
		},
		Target: extensionswebhook.TargetSeed,
		ObjectSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"provider.extensions.gardener.cloud/metal": "true"},
		},
	})
}
