// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package app

import (
	"context"
	"fmt"
	"os"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	metalinstall "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/install"
	metalcmd "github.com/metal-pod/gardener-extension-provider-metal/pkg/cmd"
	metalcontrolplane "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/controlplane"
	metalinfrastructure "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/infrastructure"
	metalworker "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/worker"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	"k8s.io/apimachinery/pkg/runtime"
	// metalcontrolplanebackup "github.com/metal-pod/gardener-extension-provider-metal/pkg/webhook/controlplanebackup"
	// metalcontrolplaneexposure "github.com/metal-pod/gardener-extension-provider-metal/pkg/webhook/controlplaneexposure"
	"github.com/gardener/gardener-extensions/pkg/controller"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
	"github.com/gardener/gardener-extensions/pkg/controller/infrastructure"
	"github.com/gardener/gardener-extensions/pkg/controller/worker"
	webhookcmd "github.com/gardener/gardener-extensions/pkg/webhook/cmd"

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
		infraReconcileOpts = &infrastructure.ReconcilerOptions{
			IgnoreOperationAnnotation: true,
		}
		infraCtrlOptsUnprefixed = controllercmd.NewOptionAggregator(infraCtrlOpts, infraReconcileOpts)

		// options for the worker controller
		workerCtrlOpts = &controllercmd.ControllerOptions{
			MaxConcurrentReconciles: 5,
		}
		workerReconcileOpts = &worker.Options{
			DeployCRDs: true,
		}
		workerCtrlOptsUnprefixed = controllercmd.NewOptionAggregator(workerCtrlOpts, workerReconcileOpts)

		controllerSwitches   = metalcmd.ControllerSwitchOptions()
		webhookSwitches      = metalcmd.WebhookSwitchOptions()
		webhookServerOptions = &webhookcmd.ServerOptions{
			Port:             7890,
			CertDir:          "/tmp/cert",
			Mode:             webhookcmd.ServiceMode,
			Name:             "webhooks",
			Namespace:        os.Getenv("WEBHOOK_CONFIG_NAMESPACE"),
			ServiceSelectors: "{}",
			Host:             "localhost",
		}
		webhookOptions = webhookcmd.NewAddToManagerOptions(metal.Name, webhookServerOptions, webhookSwitches)

		aggOption = controllercmd.NewOptionAggregator(
			restOpts,
			mgrOpts,
			controllercmd.PrefixOption("controlplane-", controlPlaneCtrlOpts),
			controllercmd.PrefixOption("infrastructure-", &infraCtrlOptsUnprefixed),
			controllercmd.PrefixOption("worker-", &workerCtrlOptsUnprefixed),
			configFileOpts,
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

				if _, err := controllerutil.CreateOrUpdate(ctx, c, obj, func(existing runtime.Object) error {
					existingCRD := existing.(*apiextensionsv1beta1.CustomResourceDefinition)
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

			configFileOpts.Completed().ApplyMachineImages(&metalworker.DefaultAddOptions.MachineImages)
			// configFileOpts.Completed().ApplyETCDStorage(&metalcontrolplaneexposure.DefaultAddOptions.ETCDStorage)
			// configFileOpts.Completed().ApplyETCDBackup(&metalcontrolplanebackup.DefaultAddOptions.ETCDBackup)
			controlPlaneCtrlOpts.Completed().Apply(&metalcontrolplane.Options)
			infraCtrlOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.Controller)
			infraReconcileOpts.Completed().Apply(&metalinfrastructure.DefaultAddOptions.IgnoreOperationAnnotation)
			workerCtrlOpts.Completed().Apply(&metalworker.DefaultAddOptions.Controller)

			if err := controllerSwitches.Completed().AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add controllers to manager")
			}

			if err := webhookOptions.Completed().AddToManager(mgr); err != nil {
				controllercmd.LogErrAndExit(err, "Could not add webhooks to manager")
			}

			if err := mgr.Start(ctx.Done()); err != nil {
				controllercmd.LogErrAndExit(err, "Error running manager")
			}
		},
	}

	aggOption.AddFlags(cmd.Flags())

	return cmd
}
