package validator

import (
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	extensionspredicate "github.com/gardener/gardener/extensions/pkg/predicate"
	extensionswebhook "github.com/gardener/gardener/extensions/pkg/webhook"
	"github.com/gardener/gardener/pkg/apis/core"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var logger = log.Log.WithName("metal-validator-webhook")

// New creates a new webhook that validates Shoot and Cloduprofile resources.
func New(mgr manager.Manager) (*extensionswebhook.Webhook, error) {
	logger.Info("Setting up webhook", "name", extensionswebhook.ValidatorName)

	return extensionswebhook.New(mgr, extensionswebhook.Args{
		Provider:   metal.Type,
		Name:       extensionswebhook.ValidatorName,
		Path:       extensionswebhook.ValidatorPath,
		Predicates: []predicate.Predicate{extensionspredicate.GardenCoreProviderType(metal.Type)},
		Validators: map[extensionswebhook.Validator][]client.Object{
			NewShootValidator():        {&core.Shoot{}},
			NewCloudProfileValidator(): {&core.CloudProfile{}},
		},
	})
}
