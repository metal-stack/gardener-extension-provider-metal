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

	"github.com/gardener/gardener-extensions/pkg/util"

	"github.com/gardener/gardener-resource-manager/pkg/manager"
	"github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RenderChartAndCreateManagedResource renders a chart and creates a ManagedResource for the gardener-resource-manager
// out of the results.
func RenderChartAndCreateManagedResource(
	ctx context.Context,
	namespace string,
	name string,
	client client.Client,
	chartRenderer chartrenderer.Interface,
	chart util.Chart,
	values map[string]interface{},
	imageVector imagevector.ImageVector,
	chartNamespace string,
	version string,
	withNoCleanupLabel bool,
) error {
	chartName, data, err := chart.Render(chartRenderer, chartNamespace, imageVector, version, version, values)
	if err != nil {
		return errors.Wrapf(err, "could not render chart")
	}

	// Create or update secret containing the rendered storage class chart
	if err := manager.
		NewSecret(client).
		WithNamespacedName(namespace, name).
		WithKeyValues(map[string][]byte{chartName: data}).
		Reconcile(ctx); err != nil {
		return errors.Wrapf(err, "could not create or update secret '%s/%s' of managed resource containing storage classes chart", namespace, name)
	}

	// Create or update managed resource referencing the previously created secret
	var injectedLabels map[string]string
	if withNoCleanupLabel {
		injectedLabels = map[string]string{ShootNoCleanupLabel: "true"}
	}

	if err := manager.
		NewManagedResource(client).
		WithNamespacedName(namespace, name).
		WithInjectedLabels(injectedLabels).
		WithSecretRef(name).
		Reconcile(ctx); err != nil {
		return errors.Wrapf(err, "could not create or update managed resource '%s/%s' containing storage classes chart", namespace, name)
	}

	return nil
}

// DeleteManagedResource deletes a managed resource and a secret with the given <name>.
func DeleteManagedResource(
	ctx context.Context,
	client client.Client,
	namespace string,
	name string,
) error {
	if err := manager.
		NewManagedResource(client).
		WithNamespacedName(namespace, name).
		Delete(ctx); err != nil {
		return errors.Wrapf(err, "could not delete managed resource '%s/%s'", namespace, name)
	}

	if err := manager.
		NewSecret(client).
		WithNamespacedName(namespace, name).
		Delete(ctx); err != nil {
		return errors.Wrapf(err, "could not delete secret '%s/%s' of managed resource", namespace, name)
	}

	return nil
}
