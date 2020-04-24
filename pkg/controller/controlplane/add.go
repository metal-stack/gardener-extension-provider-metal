package controlplane

import (
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	"github.com/spf13/pflag"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// DefaultAddOptions are the default AddOptions for AddToManager.
	DefaultAddOptions = AddOptions{}

	logger = log.Log.WithName("metal-controlplane-controller")
)

type AuthOptions struct {
	ProviderTenant string

	config *AuthConfig
}

// AddFlags implements Flagger.AddFlags.
func (a *AuthOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.ProviderTenant, "provider-tenant", a.ProviderTenant, "The name of the provider tenant for authentication, who will have extended privileges.")
}

func (a *AuthOptions) Complete() error {
	a.config = &AuthConfig{
		ProviderTenant: a.ProviderTenant,
	}
	return nil
}

func (a *AuthOptions) Completed() *AuthConfig {
	return a.config
}

type AuthConfig struct {
	ProviderTenant string
}

func (a *AuthConfig) Apply(authOpt *AuthOptions) {
	a.ProviderTenant = authOpt.ProviderTenant
}

// Options initializes empty controller.Options, applies the set values and returns it.
func (a *AuthConfig) Options() AuthOptions {
	var opts AuthOptions
	a.Apply(&opts)
	return opts
}

var AuthOpts = AuthOptions{}

// AddOptions are options to apply when adding the Packet controlplane controller to the manager.
type AddOptions struct {
	AccountingExporterConfig config.AccountingExporterConfiguration
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ShootWebhooks specifies the list of desired shoot webhooks.
	ShootWebhooks []admissionregistrationv1beta1.MutatingWebhook
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {

	return controlplane.Add(mgr, controlplane.AddArgs{
		Actuator: genericactuator.NewActuator(metal.Name, controlPlaneSecrets, nil, configChart, controlPlaneChart, cpShootChart,
			storageClassChart, nil, NewValuesProvider(mgr, logger, opts.AccountingExporterConfig, *AuthOpts.config), extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
			imagevector.ImageVector(), "", opts.ShootWebhooks, mgr.GetWebhookServer().Port, logger),
		ControllerOptions: opts.Controller,
		Predicates:        controlplane.DefaultPredicates(opts.IgnoreOperationAnnotation),
		Type:              metal.Type,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, DefaultAddOptions)
}
