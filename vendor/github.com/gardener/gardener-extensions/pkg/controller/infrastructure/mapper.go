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

package infrastructure

import (
	"context"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"

	extensions1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type secretToInfrastructureMapper struct {
	client     client.Client
	predicates []predicate.Predicate
}

func (m *secretToInfrastructureMapper) Map(obj handler.MapObject) []reconcile.Request {
	if obj.Object == nil {
		return nil
	}

	secret, ok := obj.Object.(*corev1.Secret)
	if !ok {
		return nil
	}

	infrastructureList := &extensions1alpha1.InfrastructureList{}
	if err := m.client.List(context.TODO(), client.InNamespace(secret.Namespace), infrastructureList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, infrastructure := range infrastructureList.Items {
		if !extensionscontroller.EvalGenericPredicate(&infrastructure, m.predicates...) {
			continue
		}

		if infrastructure.Spec.SecretRef.Name == secret.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: infrastructure.Namespace,
					Name:      infrastructure.Name,
				},
			})
		}
	}
	return requests
}

// SecretToInfrastructureMapper returns a mapper that returns requests for Infrastructures whose
// referenced secrets have been modified.
func SecretToInfrastructureMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &secretToInfrastructureMapper{client, predicates}
}

// ClusterToInfrastructureMapper returns a mapper that returns requests for Infrastructures whose
// referenced clusters have been modified.
func ClusterToInfrastructureMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return extensionscontroller.ClusterToObjectMapper(client, func() runtime.Object { return &extensions1alpha1.InfrastructureList{} }, predicates)
}
