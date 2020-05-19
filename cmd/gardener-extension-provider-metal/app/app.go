package app

import (
	"context"
	"fmt"
	"os"

	druidv1alpha1 "github.com/gardener/etcd-druid/api/v1alpha1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metalinstall "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/install"
	metalcmd "github.com/metal-stack/gardener-extension-provider-metal/pkg/cmd"
	metalcontrolplane "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/controlplane"
	metalinfrastructure "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/infrastructure"
	metalworker "github.com/metal-stack/gardener-extension-provider-metal/pkg/controller/worker"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	shootcontrolplanewebhook "github.com/metal-stack/gardener-extension-provider-metal/pkg/webhook/controlplane"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/gardener/gardener/extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/controller/worker"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"

	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewControllerManagerCommand creates a new command for running a Metal provider controller.
func NewControllerManagerCommand(ctx context.Context) *cobra.Command {
	var (
		restOpts = &controllercmd.RESTOptions{}
		mgrOpts  = &controllercmd.ManagerOptions{
			LeaderElection:          true,
			LeaderElectionID:        controllercmd.LeaderElectionNameID(metal.Name),
			LeaderElectionNamespace: os.Getenv("LEADER_ELECTION_NAMESPACE"),
			WebhookServerPort:       443,
		}
		configFileOpts = &metalcmd.ConfigOptions{}

		// options for the controlplane controller
		controlPlaneCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}

		// options for the infrastructure controller
		infraCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		reconcileOpts = &controllercmd.ReconcilerOptions{}

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
		webhookOptions  = webhookcmd.NewAddToManagerOptions(metal.Name, webhookServerOptions, webhookSwitches)

		aggOption = controllercmd.NewOptionAggregator(
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", infraCtrlOpts),
			controllercmd.PrefixOption("worker-", &workerCtrlOptsUnprefixed),
			configFileOpts,
			reconcileOpts,
			controllerSwitches,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("%s-controller-manager", metal.Name),

		Run: func(cmd *cobra.Command, args []string) {
			if err := aggOption.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}

			if workerReconcileOpts.Completed().DeployCRDs {
				if err := worker.ApplyMachineResourcesForConfig(ctx, restOpts.Completed().Config); err != nil {
					controllercmd.LogErrAndExit(err, "Error ensuring the machine CRDs")
				}

				// FIXME this is a copy of the logic of worker.ApplyMachineResourcesForConfig from gardener/gardener-extension
				// because there is currently nothing related to metal implemented, and should not.
				// Refactoring into separate helper required.
				name := "metal"
				kind := "Metal"
				const (
					machineGroup   = "machine.sapcloud.io"
					machineVersion = "v1alpha1"
				)
				var apiextensionsScheme = runtime.NewScheme()

				utilruntime.Must(apiextensionsscheme.AddToScheme(apiextensionsScheme))

				metalCRD := &apiextensionsv1beta1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: name + "machineclasses.machine.sapcloud.io",
					},
					Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
						Group:   machineGroup,
						Version: machineVersion,
						Scope:   apiextensionsv1beta1.NamespaceScoped,
						Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
							Kind:       kind + "MachineClass",
							Plural:     name + "machineclasses",
							Singular:   name + "machineclass",
							ShortNames: []string{name + "cls"},
						},
						Subresources: &apiextensionsv1beta1.CustomResourceSubresources{
							Status: &apiextensionsv1beta1.CustomResourceSubresourceStatus{},
						},
					},
				}

				obj := &apiextensionsv1beta1.CustomResourceDefinition{
					ObjectMeta: metav1.ObjectMeta{
						Name: metalCRD.Name,
					},
				}

				c, err := client.New(restOpts.Completed().Config, client.Options{Scheme: apiextensionsScheme})
				if err != nil {
					controllercmd.LogErrAndExit(err, "Error creating k8s client for CRD deployment")
				}

				if _, err := controllerutil.CreateOrUpdate(ctx, c, obj, func() error {
					existingCRD := obj
					existingCRD.Spec = metalCRD.Spec
					return nil
				}); err != nil {
					controllercmd.LogErrAndExit(err, "Error ensuring the metal machine CRDs")
				}
			}

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not instantiate manager")
			}

			if err := controller.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			if err := metalinstall.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			if err := druidv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			configFileOpts.Completed().ApplyMachineImages(&metalworker.DefaultAddOptions.MachineImages)
			configFileOpts.Completed().ApplyControllerConfig(&metalcontrolplane.DefaultAddOptions.ControllerConfig)
			configFileOpts.Completed().ApplyControllerConfig(&shootcontrolplanewebhook.DefaultAddOptions.ControllerConfig)
			controlPlaneCtrlOpts.Completed().Apply(&metalcontrolplane.DefaultAddOptions.Controller)
			infraCtrlOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.Controller)
			reconcileOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&metalcontrolplane.DefaultAddOptions.IgnoreOperationAnnotation)
			reconcileOpts.Completed().Apply(&metalworker.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&metalworker.DefaultAddOptions.Controller)

			_, shootWebhooks, err := webhookOptions.Completed().AddToManager(mgr)
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not add webhooks to manager")
			}
			metalcontrolplane.DefaultAddOptions.ShootWebhooks = shootWebhooks

			if err := controllerSwitches.Completed().AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add controllers to manager")
			}

			if err := mgr.Start(ctx.Done()); err != nil {
				controllercmd.LogErrAndExit(err, "Error running manager")
			}
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
