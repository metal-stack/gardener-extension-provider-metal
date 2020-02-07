package infrastructure

import (
	"github.com/gardener/gardener-extensions/pkg/controller/infrastructure"
	"github.com/go-logr/logr"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}
)

// AddOptions are options to apply when adding the metal infrastructure controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {
	logr.InfoLogger.Info(log.Log.WithName("infrastructure-actuator"), "Adding infrastructure controller")
	return infrastructure.Add(mgr, infrastructure.AddArgs{
		Actuator:          NewActuator(),
		ControllerOptions: opts.Controller,
		Predicates:        infrastructure.DefaultPredicates(opts.IgnoreOperationAnnotation),
		Type:              metal.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
