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

package cmd

import (
	controlplanecontroller "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/controlplane"
	infrastructurecontroller "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/infrastructure"
	workercontroller "github.com/metal-pod/gardener-extension-provider-metal/pkg/controller/worker"
	controlplanewebhook "github.com/metal-pod/gardener-extension-provider-metal/pkg/webhook/controlplane"
	controlplanebackupwebhook "github.com/metal-pod/gardener-extension-provider-metal/pkg/webhook/controlplanebackup"
	controlplaneexposurewebhook "github.com/metal-pod/gardener-extension-provider-metal/pkg/webhook/controlplaneexposure"
	controllercmd "github.com/gardener/gardener-extensions/pkg/controller/cmd"
	extensionscontrolplanecontroller "github.com/gardener/gardener-extensions/pkg/controller/controlplane"
	extensionsinfrastructurecontroller "github.com/gardener/gardener-extensions/pkg/controller/infrastructure"
	extensionsworkercontroller "github.com/gardener/gardener-extensions/pkg/controller/worker"
	webhookcmd "github.com/gardener/gardener-extensions/pkg/webhook/cmd"
	extensioncontrolplanewebhook "github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
)

// ControllerSwitchOptions are the controllercmd.SwitchOptions for the provider controllers.
func ControllerSwitchOptions() *controllercmd.SwitchOptions {
	return controllercmd.NewSwitchOptions(
		controllercmd.Switch(extensionsinfrastructurecontroller.ControllerName, infrastructurecontroller.AddToManager),
		controllercmd.Switch(extensionscontrolplanecontroller.ControllerName, controlplanecontroller.AddToManager),
		controllercmd.Switch(extensionsworkercontroller.ControllerName, workercontroller.AddToManager),
	)
}

WebhookSwitchOptions are the webhookcmd.SwitchOptions for the provider webhooks.
func WebhookSwitchOptions() *webhookcmd.SwitchOptions {
	return webhookcmd.NewSwitchOptions(
		webhookcmd.Switch(extensioncontrolplanewebhook.WebhookName, controlplanewebhook.AddToManager),
		webhookcmd.Switch(extensioncontrolplanewebhook.ExposureWebhookName, controlplaneexposurewebhook.AddToManager),
		webhookcmd.Switch(extensioncontrolplanewebhook.BackupWebhookName, controlplanebackupwebhook.AddToManager),
	)
}
