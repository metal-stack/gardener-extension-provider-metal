package validator

import (
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	// Name is a name for a validation webhook.
	Name = "validator"
)

var logger = log.Log.WithName("metal-validator-webhook")

// New creates a new webhook that validates Shoot and Cloduprofile resources.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Setting up webhook", "name", Name)

	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider:   metal.Type,
		Name:       Name,
		Path:       "/webhooks/validate",
		Predicates: []predicate.Predicate{extensionspredicate.GardenCoreProviderType(metal.Type)},
		Validators: map[extensionswebhook.Validator][]extensionswebhook.Type{
			NewShootValidator():        {{Obj: &core.Shoot{}}},
			NewCloudProfileValidator(): {{Obj: &core.CloudProfile{}}},
		},
	})
}
