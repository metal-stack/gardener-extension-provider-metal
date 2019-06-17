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
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// OperationAnnotationPredicate is a predicate for the operation annotation.
func OperationAnnotationPredicate() predicate.Predicate {
	annotationExists := func(obj runtime.Object) bool {
		infrastructure, ok := obj.(*extensionsv1alpha1.Infrastructure)
		if !ok {
			return false
		}
		return mayReconcile(infrastructure)
	}

	return predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return annotationExists(event.Object)
		},
		UpdateFunc: func(event event.UpdateEvent) bool {
			return annotationExists(event.ObjectNew)
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return annotationExists(event.Object)
		},
	}
}

func mayReconcile(infrastructure *extensionsv1alpha1.Infrastructure) bool {
	return infrastructure.DeletionTimestamp != nil ||
		infrastructure.Generation != infrastructure.Status.ObservedGeneration ||
		infrastructure.Status.LastOperation == nil ||
		infrastructure.Status.LastOperation.Type == gardencorev1alpha1.LastOperationTypeCreate ||
		infrastructure.Status.LastOperation.Type == gardencorev1alpha1.LastOperationTypeDelete ||
		infrastructure.Status.LastOperation.State != gardencorev1alpha1.LastOperationStateSucceeded ||
		kutil.HasMetaDataAnnotation(&infrastructure.ObjectMeta, gardencorev1alpha1.GardenerOperation, gardencorev1alpha1.GardenerOperationReconcile)
}
