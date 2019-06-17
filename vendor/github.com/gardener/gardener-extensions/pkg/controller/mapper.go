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

package controller

import (
	"context"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ObjectNameToExtensionTypeMapper returns a `ToRequestsFunc` that gets the Extension resource of type certificate-service
// in the namespace that is named after the incoming object's name.
func ObjectNameToExtensionTypeMapper(cl client.Client, extensionType string) handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		geList := extensionsv1alpha1.ExtensionList{}
		if err := cl.List(context.TODO(), client.InNamespace(object.Meta.GetName()), &geList); err != nil {
			return nil
		}

		for _, ge := range geList.Items {
			if EvalGenericPredicate(&ge, TypePredicate(extensionType)) {
				// There is only one extension object per type.
				return []reconcile.Request{
					reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      ge.GetName(),
							Namespace: object.Meta.GetName(),
						},
					}}
			}
		}
		return nil
	}
}

// TypeMapperWithinNamespace returns a `ToRequestsFunc` that maps the incoming object
// to the certificate service extension object in the same namespace.
func TypeMapperWithinNamespace(cl client.Client, extensionType string) handler.ToRequestsFunc {
	return func(object handler.MapObject) []reconcile.Request {
		geList := extensionsv1alpha1.ExtensionList{}
		if err := cl.List(context.TODO(), client.InNamespace(object.Meta.GetNamespace()), &geList); err != nil {
			return nil
		}

		for _, ge := range geList.Items {
			if EvalGenericPredicate(&ge, TypePredicate(extensionType)) {
				// There is only one extension object per type.
				return []reconcile.Request{
					reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      ge.GetName(),
							Namespace: object.Meta.GetNamespace(),
						},
					}}
			}
		}

		return nil
	}
}

type clusterToObjectMapper struct {
	client         client.Client
	newObjListFunc func() runtime.Object
	predicates     []predicate.Predicate
}

func (m *clusterToObjectMapper) Map(obj handler.MapObject) []reconcile.Request {
	ctx := context.TODO()

	if obj.Object == nil {
		return nil
	}

	cluster, ok := obj.Object.(*extensionsv1alpha1.Cluster)
	if !ok {
		return nil
	}

	objList := m.newObjListFunc()
	if err := m.client.List(ctx, client.InNamespace(cluster.Name), objList); err != nil {
		return nil
	}

	var requests []reconcile.Request

	utilruntime.HandleError(meta.EachListItem(objList, func(obj runtime.Object) error {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return err
		}

		if !EvalGenericPredicate(obj, m.predicates...) {
			return nil
		}

		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: accessor.GetNamespace(),
				Name:      accessor.GetName(),
			},
		})
		return nil
	}))

	return requests
}

// ClusterToObjectMapper returns a mapper that returns requests for objects whose
// referenced clusters have been modified.
func ClusterToObjectMapper(client client.Client, newObjListFunc func() runtime.Object, predicates []predicate.Predicate) handler.Mapper {
	return &clusterToObjectMapper{client, newObjListFunc, predicates}
}
