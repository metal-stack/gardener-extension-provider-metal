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

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// FinalizerName is the controlplane controller finalizer.
	FinalizerName = "extensions.gardener.cloud/controlplane"
	// ControllerName is the name of the controller
	ControllerName = "controlplane-controller"
)

// AddArgs are arguments for adding an controlplane controller to a manager.
type AddArgs struct {
	// Actuator is an controlplane actuator.
	Actuator Actuator
	// Type is the controlplane type the actuator supports.
	Type string
	// ControllerOptions are the controller options used for creating a controller.
	// The options.Reconciler is always overridden with a reconciler created from the
	// given actuator.
	ControllerOptions controller.Options
	// Predicates are the predicates to use.
	// If unset, GenerationChangedPredicate will be used.
	Predicates []predicate.Predicate
}

// DefaultPredicates returns the default predicates for a controlplane reconciler.
func DefaultPredicates(mgr manager.Manager) []predicate.Predicate {
	return []predicate.Predicate{
		extensionscontroller.ShootFailedPredicate(mgr.GetClient()),
		extensionscontroller.GenerationChangedPredicate(),
	}
}

// Add creates a new ControlPlane Controller and adds it to the Manager.
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, args AddArgs) error {
	args.ControllerOptions.Reconciler = NewReconciler(mgr, args.Actuator)
	return add(mgr, args.Type, args.ControllerOptions, args.Predicates)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, typeName string, options controller.Options, predicates []predicate.Predicate) error {
	ctrl, err := controller.New(ControllerName, mgr, options)
	if err != nil {
		return err
	}

	if predicates == nil {
		predicates = DefaultPredicates(mgr)
	}
	predicates = append(predicates, extensionscontroller.TypePredicate(typeName))

	if err := ctrl.Watch(&source.Kind{Type: &extensionsv1alpha1.ControlPlane{}}, &handler.EnqueueRequestForObject{}, predicates...); err != nil {
		return err
	}
	if err := ctrl.Watch(&source.Kind{Type: &corev1.Secret{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: SecretToControlPlaneMapper(mgr.GetClient(), predicates)}); err != nil {
		return err
	}
	if err := ctrl.Watch(&source.Kind{Type: &extensionsv1alpha1.Cluster{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: ClusterToControlPlaneMapper(mgr.GetClient(), predicates)}); err != nil {
		return err
	}
	return nil
}
