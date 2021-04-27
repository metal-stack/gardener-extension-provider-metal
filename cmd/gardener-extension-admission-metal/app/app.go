package app

import (
	"context"
	"fmt"

	admissioncmd "github.com/metal-stack/gardener-extension-provider-metal/pkg/admission/cmd"
	metalinstall "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/install"
	providermetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"

	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	"github.com/gardener/gardener/extensions/pkg/util"
	webhookcmd "github.com/gardener/gardener/extensions/pkg/webhook/cmd"
	"github.com/gardener/gardener/pkg/apis/core/install"
	"github.com/spf13/cobra"
	componentbaseconfig "k8s.io/component-base/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logf.Log.WithName("gardener-extension-admission-metal")

// NewAdmissionCommand creates a new command for running an metal admission webhook.
func NewAdmissionCommand(ctx context.Context) *cobra.Command {
	var (
		restOpts = &controllercmd.RESTOptions{}
		mgrOpts  = &controllercmd.ManagerOptions{
			WebhookServerPort: 443,
		}
		webhookSwitches = admissioncmd.GardenWebhookSwitchOptions()
		webhookOptions  = webhookcmd.NewAddToManagerSimpleOptions(webhookSwitches)

		aggOption = controllercmd.NewOptionAggregator(
			restOpts,
			mgrOpts,
			webhookOptions,
		)
	)

	cmd := &cobra.Command{
		Use: fmt.Sprintf("admission-%s", providermetal.Type),

		RunE: func(cmd *cobra.Command, args []string) error {
			if err := aggOption.Complete(); err != nil {
				controllercmd.LogErrAndExit(err, "Error completing options")
			}

			util.ApplyClientConnectionConfigurationToRESTConfig(&componentbaseconfig.ClientConnectionConfiguration{
				QPS:   100.0,
				Burst: 130,
			}, restOpts.Completed().Config)

			mgr, err := manager.New(restOpts.Completed().Config, mgrOpts.Completed().Options())
			if err != nil {
				controllercmd.LogErrAndExit(err, "Could not instantiate manager")
			}

			install.Install(mgr.GetScheme())

			if err := metalinstall.AddToScheme(mgr.GetScheme()); err != nil {
				controllercmd.LogErrAndExit(err, "Could not update manager scheme")
			}

			log.Info("Setting up webhook server")
			if err := webhookOptions.Completed().AddToManager(mgr); err != nil {
				return err
			}

			return mgr.Start(ctx)
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
