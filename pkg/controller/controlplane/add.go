package controlplane

import (
	"context"
	"sync/atomic"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/util"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}

	logger = log.Log.WithName("metal-controlplane-controller")
)

// AddOptions are options to apply when adding the Packet controlplane controller to the manager.
type AddOptions struct {
	ControllerConfig config.ControllerConfiguration
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ShootWebhookConfig specifies the desired Shoot MutatingWebhooksConfiguration.
	ShootWebhookConfig *atomic.Value
	// WebhookServerNamespace is the namespace in which the webhook server runs.
	WebhookServerNamespace string
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(ctx context.Context, mgr manager.Manager, opts AddOptions) error {
	actuator, err := genericactuator.NewActuator(mgr, metal.Name,
		secretConfigsFunc, shootAccessSecretsFunc, nil, nil,
		nil, controlPlaneChart, cpShootChart, nil, storageClassChart, nil,
		NewValuesProvider(mgr, opts.ControllerConfig), extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
		imagevector.ImageVector(), "", opts.ShootWebhookConfig, opts.WebhookServerNamespace,
	)
	if err != nil {
		return err
	}

	return controlplane.Add(ctx, mgr, controlplane.AddArgs{
		Actuator:          actuator,
		ControllerOptions: opts.Controller,
		Predicates:        controlplane.DefaultPredicates(ctx, mgr, opts.IgnoreOperationAnnotation),
		Type:              metal.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(ctx context.Context, mgr manager.Manager) error {
	return AddToManagerWithOptions(ctx, mgr, DefaultAddOptions)
}
