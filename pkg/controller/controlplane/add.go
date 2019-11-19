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

package controlplane

import (
	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
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

type AccountingOptions struct {
	AccountingSinkUrl  string
	AccountingSinkHmac string

	config *AccountingConfig
}

// AddFlags implements Flagger.AddFlags.
func (a *AccountingOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.AccountingSinkUrl, "url", a.AccountingSinkUrl, "Url of the accounting sink API.")
	fs.StringVar(&a.AccountingSinkHmac, "hmac", a.AccountingSinkHmac, "HMAC for the accounting sink API.")
}

func (a *AccountingOptions) Complete() error {
	a.config = &AccountingConfig{
		AccountingSinkUrl:  a.AccountingSinkUrl,
		AccountingSinkHmac: a.AccountingSinkHmac,
	}
	return nil
}

func (a *AccountingOptions) Completed() *AccountingConfig {
	return a.config
}

type AccountingConfig struct {
	AccountingSinkUrl  string
	AccountingSinkHmac string
}

func (a *AccountingConfig) Apply(accOpt *AccountingOptions) {
	a.AccountingSinkUrl = accOpt.AccountingSinkUrl
	a.AccountingSinkHmac = accOpt.AccountingSinkHmac
}

// Options initializes empty controller.Options, applies the set values and returns it.
func (a *AccountingConfig) Options() AccountingOptions {
	var opts AccountingOptions
	a.Apply(&opts)
	return opts
}

var AccOpts = AccountingOptions{}

// AddOptions are options to apply when adding the Packet controlplane controller to the manager.
type AddOptions struct {
	// Controller are the controller.Options.
	Controller controller.Options
	// IgnoreOperationAnnotation specifies whether to ignore the operation annotation or not.
	IgnoreOperationAnnotation bool
	// ShootWebhooks specifies the list of desired shoot webhooks.
	ShootWebhooks []admissionregistrationv1beta1.Webhook
}

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts AddOptions) error {

	return controlplane.Add(mgr, controlplane.AddArgs{
		Actuator: genericactuator.NewActuator(metal.Name, controlPlaneSecrets, nil, configChart, controlPlaneChart, cpShootChart,
			storageClassChart, nil, NewValuesProvider(mgr, logger, *AccOpts.config), extensionscontroller.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
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
