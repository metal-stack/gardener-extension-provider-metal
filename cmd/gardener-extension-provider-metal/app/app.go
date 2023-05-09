package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	gardenerhealthz "github.com/gardener/gardener/pkg/healthz"
	metalinstall "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/install"
	metalcmd "github.com/metal-stack/gardener-extension-provider-metal/pkg/cmd"
	metalcontrolplane "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/controlplane"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/healthcheck"
	metalinfrastructure "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/infrastructure"
	metalworker "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/worker"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	shootcontrolplanewebhook "github.com/metal-stack/gardener-extension-provider-metal/pkg/webhook/controlplane"
	metalcontrolplaneexposure "github.com/metal-stack/gardener-extension-provider-metal/pkg/webhook/controlplaneexposure"

	"github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	"github.com/gardener/gardener/extensions/pkg/util"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/gardener/gardener/pkg/client/kubernetes"

	"github.com/spf13/cobra"
	"k8s.io/component-base/version/verflag"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewControllerManagerCommand creates a new command for running a Metal provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		generalOpts = &controllercmd.GeneralOptions{}
		restOpts    = &controllercmd.RESTOptions{}
		mgrOpts     = &controllercmd.ManagerOptions{
			LeaderElection:             true,
			LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
			LeaderElectionID:           controllercmd.LeaderElectionNameID(metal.Name),
			LeaderElectionNamespace:    os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:          443,
			WebhookCertDir:             "/tmp/gardener-extensions-cert",
			HealthBindAddress:          ":8081",
		}
		configFileOpts = &metalcmd.ConfigOptions{}

		// options for the health care controller
		healthCheckCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

		// options for the controlplane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		workerReconcileOpts = &worker.Options{
			DeployCRDs: true,
		}
		workerCtrlOptsUnprefixed = controllercmd.NewOptionAggregator(workerCtrlOpts, workerReconcileOpts)

		controllerSwitches   = metalcmd.ControllerSwitchOptions()
		webhookServerOptions = &webhookcmd.ServerOptions{
			Namespace: os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
		}
		webhookSwitches = metalcmd.WebhookSwitchOptions()
		webhookOptions  = webhookcmd.NewAddToManagerOptions(
			metal.Name,
			genericactuator.ShootWebhooksResourceName,
			genericactuator.ShootWebhookNamespaceSelector(metal.Type),
			webhookServerOptions,
			webhookSwitches,
		)

		aggOption = controllercmd.NewOptionAggregator(
			generalOpts,
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("healthcheck-", healthCheckCtrlOpts),
			controllercmd.PrefixOption("worker-", &workerCtrlOptsUnprefixed),
			configFileOpts,
			reconcileOpts,
			controllerSwitches,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-controller-manager", metal.Name),

		RunE: func(cmd *cobra.Command, args []string) error {
			verflag.PrintAndExitIfRequested()

			if err := aggOption.Complete(); err != nil {
				return fmt.Errorf("error completing options: %w", err)
			}

			util.ApplyClientConnectionConfigurationToRESTConfig(configFileOpts.Completed().Config.ClientConnection, restOpts.Completed().Config)

			if workerReconcileOpts.Completed().DeployCRDs {
				ca, err := kubernetes.NewChartApplierForConfig(restOpts.Completed().Config)
				if err != nil {
					return fmt.Errorf("error creating chart renderer: %w", err)
				}

				err = ca.Apply(ctx, filepath.Join(metal.InternalChartsPath, "metal-crds"), "", "metal-crds")
				if err != nil {
					return fmt.Errorf("error applying metal-crds chart: %w", err)
				}

				c, err := client.New(restOpts.Completed().Config, client.Options{})
				if err != nil {
					return fmt.Errorf("error creating k8s client for firewall namespace deployment: %w", err)
				}

				// the firewall namespace needs to exist in order to be able to deploy the control plane chart properly
				namespace := v1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "firewall",
					},
				}

				if _, err := controllerutil.CreateOrUpdate(ctx, c, &namespace, func() error {
					return nil
				}); err != nil {
					return fmt.Errorf("error ensuring the firewall namespace: %w", err)
				}
			}

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				return fmt.Errorf("could not instantiate manager: %w", err)
			}

			scheme := mgr.GetScheme()
			if err := controller.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			if err := metalinstall.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			if err := druidv1alpha1.AddToScheme(scheme); err != nil {
				return fmt.Errorf("could not update manager scheme: %w", err)
			}

			configFileOpts.Completed().ApplyETCD(&metalcontrolplaneexposure.DefaultAddOptions.ETCD)
			configFileOpts.Completed().ApplyMachineImages(&metalworker.DefaultAddOptions.MachineImages)
			configFileOpts.Completed().ApplyControllerConfig(&metalcontrolplane.DefaultAddOptions.ControllerConfig)
			configFileOpts.Completed().ApplyControllerConfig(&shootcontrolplanewebhook.DefaultAddOptions.ControllerConfig)
			configFileOpts.Completed().ApplyControllerConfig(&healthcheck.DefaultAddOptions.ControllerConfig)
			configFileOpts.Completed().ApplyControllerConfig(&metalworker.DefaultAddOptions.ControllerConfig)
			configFileOpts.Completed().ApplyHealthCheckConfig(&healthcheck.DefaultAddOptions.HealthCheckDefaults.HealthCheckConfig)
			controlPlaneCtrlOpts.Completed().Apply(&metalcontrolplane.DefaultAddOptions.Controller)
			infraCtrlOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.Controller)
			healthCheckCtrlOpts.Completed().Apply(&healthcheck.DefaultAddOptions.HealthCheckDefaults.Controller)
			reconcileOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&metalcontrolplane.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&metalworker.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&metalworker.DefaultAddOptions.Controller)

			atomicShootWebhookConfig, err := webhookOptions.Completed().AddToManager(ctx, mgr)
			if err != nil {
				return fmt.Errorf("could not add webhooks to manager: %w", err)
			}
			metalcontrolplane.DefaultAddOptions.ShootWebhooks = atomicShootWebhookConfig

			if err := controllerSwitches.Completed().AddToManager(mgr); err != nil {
				return fmt.Errorf("could not add controllers to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("informer-sync", gardenerhealthz.NewCacheSyncHealthz(mgr.GetCache())); err != nil {
				return fmt.Errorf("could not add readycheck for informers: %w", err)
			}

			if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
				return fmt.Errorf("could not add health check to manager: %w", err)
			}

			if err := mgr.AddReadyzCheck("webhook-server", mgr.GetWebhookServer().StartedChecker()); err != nil {
				return fmt.Errorf("could not add ready check for webhook server to manager: %w", err)
			}

			if err := mgr.Start(ctx); err != nil {
				return fmt.Errorf("error running manager: %w", err)
			}

			return nil
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
