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

package worker

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

type secretToWorkerMapper struct {
	client     client.Client
	predicates []predicate.Predicate
}

func (m *secretToWorkerMapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.TODO()

	if obj.Object == nil {
		return nil
	}

	secret, ok := obj.Object.(*corev1.Secret)
	if !ok {
		return nil
	}

	workerList := &extensions1alpha1.WorkerList{}
	if err := m.client.List(ctx, client.InNamespace(secret.Namespace), workerList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, worker := range workerList.Items {
		if !extensionscontroller.EvalGenericPredicate(&worker, m.predicates...) {
			continue
		}

		if worker.Spec.SecretRef.Name == secret.Name {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: worker.Namespace,
					Name:      worker.Name,
				},
			})
			continue
		}
	}

	return requests
}

// SecretToWorkerMapper returns a mapper that returns requests for Workers whose
// referenced secrets have been modified.
func SecretToWorkerMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return &secretToWorkerMapper{client, predicates}
}

// ClusterToWorkerMapper returns a mapper that returns requests for Worker whose
// referenced clusters have been modified.
func ClusterToWorkerMapper(client client.Client, predicates []predicate.Predicate) handler.Mapper {
	return extensionscontroller.ClusterToObjectMapper(client, func() runtime.Object { return &extensions1alpha1.WorkerList{} }, predicates)
}
