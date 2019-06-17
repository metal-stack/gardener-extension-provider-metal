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
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
)

// Actuator acts upon Infrastructure resources.
type Actuator interface {
	// Reconcile the Infrastructure config.
	Reconcile(context.Context, *extensionsv1alpha1.Infrastructure, *extensionscontroller.Cluster) error
	// Delete the Infrastructure config.
	Delete(context.Context, *extensionsv1alpha1.Infrastructure, *extensionscontroller.Cluster) error
}

type operationAnnotationWrapper struct {
	Actuator
	client client.Client
}

// OperationAnnotationWrapper is a wrapper for an actuator that, after a successful reconcile,
// removes the Gardener operation annotation.
//
// This is useful in conjunction with the OperationAnnotationPredicate.
func OperationAnnotationWrapper(actuator Actuator) Actuator {
	return &operationAnnotationWrapper{Actuator: actuator}
}

// InjectClient implements inject.Client.
func (o *operationAnnotationWrapper) InjectClient(client client.Client) error {
	o.client = client
	return nil
}

// InjectClient implements inject.Func.
func (o *operationAnnotationWrapper) InjectFunc(f inject.Func) error {
	return f(o.Actuator)
}

// Reconcile implements Actuator.
func (o *operationAnnotationWrapper) Reconcile(ctx context.Context, infra *extensionsv1alpha1.Infrastructure, cluster *extensionscontroller.Cluster) error {
	if kutil.HasMetaDataAnnotation(&infra.ObjectMeta, gardencorev1alpha1.GardenerOperation, gardencorev1alpha1.GardenerOperationReconcile) {
		delete(infra.Annotations, gardencorev1alpha1.GardenerOperation)
		if err := o.client.Update(ctx, infra); err != nil {
			return err
		}
	}

	return o.Actuator.Reconcile(ctx, infra, cluster)
}
