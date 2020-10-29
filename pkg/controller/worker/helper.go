/*
 * Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package worker

import (
	"context"

	"github.com/gardener/gardener/extensions/pkg/controller"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	api "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
)

func (w *workerDelegate) decodeWorkerProviderStatus(ctx context.Context) (*api.WorkerStatus, error) {
	workerStatus := &api.WorkerStatus{}

	if w.worker.Status.ProviderStatus == nil {
		return workerStatus, nil
	}

	if _, _, err := w.decoder.Decode(w.worker.Status.ProviderStatus.Raw, nil, workerStatus); err != nil {
		return nil, errors.Wrapf(err, "could not decode WorkerStatus '%s'", kutil.ObjectName(w.worker))
	}

	return workerStatus, nil
}

func (w *workerDelegate) updateWorkerProviderStatus(ctx context.Context, workerStatus *api.WorkerStatus) error {
	var workerStatusV1alpha1 = &v1alpha1.WorkerStatus{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "WorkerStatus",
		},
	}

	if err := w.scheme.Convert(ctx, workerStatus, workerStatusV1alpha1); err != nil {
		return err
	}

	return controller.TryUpdateStatus(ctx, retry.DefaultBackoff, w.client, w.worker, func() error {
		w.worker.Status.ProviderStatus = &runtime.RawExtension{Object: workerStatusV1alpha1}
		return nil
	})
}
