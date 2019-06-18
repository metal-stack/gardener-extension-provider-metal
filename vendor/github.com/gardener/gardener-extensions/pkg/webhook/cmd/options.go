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
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	extensionwebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	// PortFlag is the name of the command line flag to specify the webhook server port.
	PortFlag = "webhook-server-port"
	// CertDirFlag is the name of the command line flag to specify the directory that contains the webhook server key and certificate.
	CertDirFlag = "webhook-server-cert-dir"
	// ModeFlag is the name of the command line flag to specify the webhook config mode, either 'service' or 'url'.
	ModeFlag = "webhook-config-mode"
	// NameFlag is the name of the command line flag to specify the webhook config name.
	NameFlag = "webhook-config-name"
	// NamespaceFlag is the name of the command line flag to specify the webhook config namespace for 'service' mode.
	NamespaceFlag = "webhook-config-namespace"
	// ServiceSelectorsFlag is the name of the command line flag to specify the webhook config service selectors as JSON for 'service' mode.
	ServiceSelectorsFlag = "webhook-config-service-selectors"
	// HostFlag is the name of the command line flag to specify the webhook config host for 'url' mode.
	HostFlag = "webhook-config-host"

	// DisableFlag is the name of the command line flag to disable individual webhooks.
	DisableFlag = "disable-webhooks"
)

// Webhook config modes
const (
	ServiceMode = "service"
	URLMode     = "url"
)

// ServerOptions are command line options that can be set for ServerConfig.
type ServerOptions struct {
	// Port is the webhook server port.
	Port int32
	// CertDir is the directory that contains the webhook server key and certificate.
	CertDir string
	// Mode is the webhook config mode, either 'service' or 'url'
	Mode string
	// Name is the webhook config name.
	Name string
	// Namespace is the webhook config namespace for 'service' mode.
	Namespace string
	// ServiceSelectors is the webhook config service selectors as JSON for 'service' mode.
	ServiceSelectors string
	// Host is the webhook config host for 'url' mode.
	Host string

	config *ServerConfig
}

// ServerConfig is a completed webhook server configuration.
type ServerConfig struct {
	// Port is the webhook server port.
	Port int32
	// CertDir is the directory that contains the webhook server key and certificate.
	CertDir string
	// BootstrapOptions contains the options for bootstrapping the webhook server.
	BootstrapOptions *webhook.BootstrapOptions
}

// Complete implements Completer.Complete.
func (w *ServerOptions) Complete() error {
	bootstrapOptions, err := w.buildBootstrapOptions()
	if err != nil {
		return err
	}

	w.config = &ServerConfig{
		Port:             w.Port,
		CertDir:          w.CertDir,
		BootstrapOptions: bootstrapOptions,
	}
	return nil
}

// Completed returns the completed ServerConfig. Only call this if `Complete` was successful.
func (w *ServerOptions) Completed() *ServerConfig {
	return w.config
}

// Options returns the webhook.ServerOptions of this ServerConfig.
func (w *ServerConfig) Options() webhook.ServerOptions {
	return webhook.ServerOptions{
		Port:             w.Port,
		CertDir:          w.CertDir,
		BootstrapOptions: w.BootstrapOptions,
	}
}

// AddFlags implements Flagger.AddFlags.
func (w *ServerOptions) AddFlags(fs *pflag.FlagSet) {
	fs.Int32Var(&w.Port, PortFlag, w.Port, "The webhook server port.")
	fs.StringVar(&w.CertDir, CertDirFlag, w.CertDir, "The directory that contains the webhook server key and certificate.")
	fs.StringVar(&w.Mode, ModeFlag, w.Mode, "The webhook config mode, either 'service' or 'url'.")
	fs.StringVar(&w.Name, NameFlag, w.Name, "The webhook config name.")
	fs.StringVar(&w.Namespace, NamespaceFlag, w.Namespace, "The webhook config namespace for 'service' mode.")
	fs.StringVar(&w.ServiceSelectors, ServiceSelectorsFlag, w.ServiceSelectors, "The webhook config service selectors as JSON for 'service' mode.")
	fs.StringVar(&w.Host, HostFlag, w.Host, "The webhook config host for 'url' mode.")
}

func (w *ServerOptions) buildBootstrapOptions() (*webhook.BootstrapOptions, error) {
	switch w.Mode {
	case ServiceMode:
		serviceSelectors := make(map[string]string)
		if err := json.Unmarshal([]byte(w.ServiceSelectors), &serviceSelectors); err != nil {
			return nil, errors.Wrap(err, "could not unmarshal webhook config service selectors from JSON")
		}

		return &webhook.BootstrapOptions{
			MutatingWebhookConfigName: w.Name,
			Service: &webhook.Service{
				Name:      w.Name,
				Namespace: w.Namespace,
				Selectors: serviceSelectors,
			},
		}, nil

	case URLMode:
		return &webhook.BootstrapOptions{
			MutatingWebhookConfigName: w.Name,
			Host:                      &w.Host,
		}, nil

	default:
		return nil, errors.Errorf("invalid webhook config mode '%s'", w.Mode)
	}
}

// NameToFactory binds a specific name to a webhook's factory function.
type NameToFactory struct {
	Name string
	Func func(manager.Manager) (webhook.Webhook, error)
}

// SwitchOptions are options to build an AddToManager function that filters the disabled webhooks.
type SwitchOptions struct {
	Disabled []string

	nameToWebhookFactory     map[string]func(manager.Manager) (webhook.Webhook, error)
	webhookFactoryAggregator extensionwebhook.FactoryAggregator
}

// Register registers the given NameToWebhookFuncs in the options.
func (w *SwitchOptions) Register(pairs ...NameToFactory) {
	for _, pair := range pairs {
		w.nameToWebhookFactory[pair.Name] = pair.Func
	}
}

// AddFlags implements Option.
func (w *SwitchOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&w.Disabled, DisableFlag, w.Disabled, "List of webhooks to disable")
}

// Complete implements Option.
func (w *SwitchOptions) Complete() error {
	disabled := sets.NewString()
	for _, disabledName := range w.Disabled {
		if _, ok := w.nameToWebhookFactory[disabledName]; !ok {
			return fmt.Errorf("cannot disable unknown webhook %q", disabledName)
		}
		disabled.Insert(disabledName)
	}

	for name, addToManager := range w.nameToWebhookFactory {
		if !disabled.Has(name) {
			w.webhookFactoryAggregator.Register(addToManager)
		}
	}
	return nil
}

// Completed returns the completed SwitchConfig. Call this only after successfully calling `Completed`.
func (w *SwitchOptions) Completed() *SwitchConfig {
	return &SwitchConfig{WebhooksFactory: w.webhookFactoryAggregator.Webhooks}
}

// SwitchConfig is the completed configuration of SwitchOptions.
type SwitchConfig struct {
	WebhooksFactory func(manager.Manager) ([]webhook.Webhook, error)
}

// Switch binds the given name to the given AddToManager function.
func Switch(name string, f func(manager.Manager) (webhook.Webhook, error)) NameToFactory {
	return NameToFactory{
		Name: name,
		Func: f,
	}
}

// NewSwitchOptions creates new SwitchOptions with the given initial pairs.
func NewSwitchOptions(pairs ...NameToFactory) *SwitchOptions {
	opts := SwitchOptions{nameToWebhookFactory: make(map[string]func(manager.Manager) (webhook.Webhook, error))}
	opts.Register(pairs...)
	return &opts
}

// AddToManagerOptions are options to create an `AddToManager` function from ServerOptions and SwitchOptions.
type AddToManagerOptions struct {
	serverName string
	Server     ServerOptions
	Switch     SwitchOptions
}

// NewAddToManagerOptions creates new AddToManagerOptions with the given server name, server, and switch options.
func NewAddToManagerOptions(serverName string, serverOpts *ServerOptions, switchOpts *SwitchOptions) *AddToManagerOptions {
	return &AddToManagerOptions{
		serverName: serverName,
		Server:     *serverOpts,
		Switch:     *switchOpts,
	}
}

// AddFlags implements Option.
func (c *AddToManagerOptions) AddFlags(fs *pflag.FlagSet) {
	c.Switch.AddFlags(fs)
	c.Server.AddFlags(fs)
}

// Complete implements Option.
func (c *AddToManagerOptions) Complete() error {
	if err := c.Switch.Complete(); err != nil {
		return err
	}

	return c.Server.Complete()
}

// Compoleted returns the completed AddToManagerConfig. Only call this if a previous call to `Complete` succeeded.
func (c *AddToManagerOptions) Completed() *AddToManagerConfig {
	return &AddToManagerConfig{
		serverName: c.serverName,
		Server:     *c.Server.Completed(),
		Switch:     *c.Switch.Completed(),
	}
}

// AddToManagerConfig is a completed AddToManager configuration.
type AddToManagerConfig struct {
	serverName string
	Server     ServerConfig
	Switch     SwitchConfig
}

// AddToManager instantiates all webhooks of this configuration. If there are any webhooks, it creates a
// webhook server, registers the webhooks and adds the server to the manager. Otherwise, it is a no-op.
func (c *AddToManagerConfig) AddToManager(mgr manager.Manager) error {
	webhooks, err := c.Switch.WebhooksFactory(mgr)
	if err != nil {
		return errors.Wrapf(err, "could not create webhooks")
	}

	return extensionwebhook.NewServerBuilder(c.serverName, c.Server.Options(), webhooks...).AddToManager(mgr)
}
