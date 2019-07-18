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
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane"
	"github.com/gardener/gardener-extensions/pkg/controller/controlplane/genericactuator"
	"github.com/gardener/gardener-extensions/pkg/util"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/imagevector"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	// Options are the default controller.Options for AddToManager.
	Options = controller.Options{}

	logger = log.Log.WithName("metal-controlplane-controller")
)

// AddToManagerWithOptions adds a controller with the given Options to the given manager.
// The opts.Reconciler is being set with a newly instantiated actuator.
func AddToManagerWithOptions(mgr manager.Manager, opts controller.Options) error {
	return controlplane.Add(mgr, controlplane.AddArgs{
		Actuator: genericactuator.NewActuator(controlPlaneSecrets, nil, ccmChart, ccmShootChart,
			NewValuesProvider(logger), genericactuator.ChartRendererFactoryFunc(util.NewChartRendererForShoot),
			imagevector.ImageVector(), "", logger),
		Type:              metal.Type,
		ControllerOptions: opts,
	})
}

// AddToManager adds a controller with the default Options.
func AddToManager(mgr manager.Manager) error {
	return AddToManagerWithOptions(mgr, Options)
}
