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

package webhook

import (
	"github.com/pkg/errors"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

const (
	// NameSuffix is a common suffix for all webhook names.
	NameSuffix = "extensions.gardener.cloud"

	// SeedProviderLabel is a label on shoot namespaces in the seed cluster that identifies the Seed provider.
	// TODO Move this constant to gardener/gardener
	SeedProviderLabel = "seed.gardener.cloud/provider"
	// ShootProviderLabel is a label on shoot namespaces in the seed cluster that identifies the Shoot provider.
	// TODO Move this constant to gardener/gardener
	ShootProviderLabel = "shoot.gardener.cloud/provider"
	// BackupProviderLabel is a label on shoot namespaces in the seed cluster that identifies the Backup provider.
	// This provider can be different from both the Seed or the Shoot provider, see https://github.com/gardener/gardener/blob/master/docs/proposals/02-backupinfra.md.
	// TODO Move this constant to gardener/gardener
	BackupProviderLabel = "backup.gardener.cloud/provider"
)

// Kind is a type for webhook kinds.
type Kind string

// Webhook kinds.
const (
	// A seed webhook is applied only to those shoot namespaces that have the correct Seed provider label.
	SeedKind Kind = "seed"
	// A shoot webhook is applied only to those shoot namespaces that have the correct Shoot provider label.
	ShootKind Kind = "shoot"
	// A backup webhook is applied only to those shoot namespaces that have the correct Backup provider label.
	BackupKind Kind = "backup"
)

// FactoryAggregator aggregates various Factory functions.
type FactoryAggregator []func(manager.Manager) (webhook.Webhook, error)

// NewFactoryAggregator creates a new FactoryAggregator and registers the given functions.
func NewFactoryAggregator(funcs ...func(manager.Manager) (webhook.Webhook, error)) FactoryAggregator {
	var builder FactoryAggregator
	builder.Register(funcs...)
	return builder
}

// Register registers the given functions in this builder.
func (a *FactoryAggregator) Register(funcs ...func(manager.Manager) (webhook.Webhook, error)) {
	*a = append(*a, funcs...)
}

// Webhooks calls all factories with the given managers and returns all created webhooks.
// As soon as there is an error creating a webhook, the error is returned immediately.
func (a *FactoryAggregator) Webhooks(mgr manager.Manager) ([]webhook.Webhook, error) {
	var webhooks []webhook.Webhook
	for _, f := range *a {
		wh, err := f(mgr)
		if err != nil {
			return nil, err
		}

		webhooks = append(webhooks, wh)
	}

	return webhooks, nil
}

// ServerBuilder is a builder to build a webhook server.
type ServerBuilder struct {
	Name     string
	Options  webhook.ServerOptions
	Webhooks []webhook.Webhook
}

// NewServerBuilder instantiates a new ServerBuilder with the given name, options and initial set of webhooks.
func NewServerBuilder(name string, options webhook.ServerOptions, webhooks ...webhook.Webhook) *ServerBuilder {
	return &ServerBuilder{name, options, webhooks}
}

// Register registers the given Webhooks in this ServerBuilder.
func (s *ServerBuilder) Register(webhooks ...webhook.Webhook) {
	s.Webhooks = append(s.Webhooks, webhooks...)
}

// AddToManager creates and adds the webhook server to the manager if there are any webhooks.
// If there are no webhooks, this is a no-op.
func (s *ServerBuilder) AddToManager(mgr manager.Manager) error {
	if len(s.Webhooks) == 0 {
		return nil
	}

	srv, err := webhook.NewServer(s.Name, mgr, s.Options)
	if err != nil {
		return errors.Wrapf(err, "could not create webhook server %s", s.Name)
	}

	if err := srv.Register(s.Webhooks...); err != nil {
		return errors.Wrapf(err, "could not register webhooks in server %s", s.Name)
	}

	return nil
}

// NewWebhook creates a new mutating webhook for create and update operations
// with the given kind, provider, and name, applicable to objects of all given types,
// executing the given handler, and bound to the given manager.
func NewWebhook(mgr manager.Manager, kind Kind, provider, name string, types []runtime.Object, handler admission.Handler) (*admission.Webhook, error) {
	// Build namespace selector from the webhook kind and provider
	namespaceSelector, err := buildSelector(kind, provider)
	if err != nil {
		return nil, err
	}

	// Build rules for all object types
	var rules []admissionregistrationv1beta1.RuleWithOperations
	for _, t := range types {
		rule, err := buildRule(mgr, t)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	// Build webhook
	return builder.NewWebhookBuilder().
		Name(name + "." + provider + "." + NameSuffix).
		Path("/" + name).
		Mutating().
		FailurePolicy(admissionregistrationv1beta1.Fail).
		NamespaceSelector(namespaceSelector).
		Rules(rules...).
		Handlers(handler).
		WithManager(mgr).
		Build()
}

// buildSelector creates and returns a LabelSelector for the given webhook kind and provider.
func buildSelector(kind Kind, provider string) (*metav1.LabelSelector, error) {
	// Determine label selector key from the kind
	var key string
	switch kind {
	case SeedKind:
		key = SeedProviderLabel
	case ShootKind:
		key = ShootProviderLabel
	case BackupKind:
		key = BackupProviderLabel
	default:
		return nil, errors.Errorf("invalid webhook kind '%s'", kind)
	}

	// Create and return LabelSelector
	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: key, Operator: metav1.LabelSelectorOpIn, Values: []string{provider}},
		},
	}, nil
}

// buildRule creates and returns a RuleWithOperations for the given object type.
func buildRule(mgr manager.Manager, t runtime.Object) (*admissionregistrationv1beta1.RuleWithOperations, error) {
	// Get GVK from the type
	gvk, err := apiutil.GVKForObject(t, mgr.GetScheme())
	if err != nil {
		return nil, errors.Wrapf(err, "could not get GroupVersionKind from object %v", t)
	}

	// Get REST mapping from GVK
	mapping, err := mgr.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get REST mapping from GroupVersionKind '%s'", gvk.String())
	}

	// Create and return RuleWithOperations
	return &admissionregistrationv1beta1.RuleWithOperations{
		Operations: []admissionregistrationv1beta1.OperationType{
			admissionregistrationv1beta1.Create,
			admissionregistrationv1beta1.Update,
		},
		Rule: admissionregistrationv1beta1.Rule{
			APIGroups:   []string{gvk.Group},
			APIVersions: []string{gvk.Version},
			Resources:   []string{mapping.Resource.Resource},
		},
	}, nil
}
