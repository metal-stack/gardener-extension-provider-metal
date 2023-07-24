package worker

import (
	"context"
	"fmt"

	api "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (w *workerDelegate) decodeWorkerProviderStatus() (*api.WorkerStatus, error) {
	workerStatus := &api.WorkerStatus{}

	if w.worker.Status.ProviderStatus == nil {
		return workerStatus, nil
	}

	if _, _, err := w.decoder.Decode(w.worker.Status.ProviderStatus.Raw, nil, workerStatus); err != nil {
		return nil, fmt.Errorf("could not decode WorkerStatus '%s' %w", kutil.ObjectName(w.worker), err)
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

	if err := w.scheme.Convert(workerStatus, workerStatusV1alpha1, nil); err != nil {
		return err
	}
	patch := cclient.MergeFrom(w.worker.DeepCopy())
	w.worker.Status.ProviderStatus = &runtime.RawExtension{Object: workerStatusV1alpha1}
	return w.client.Status().Patch(ctx, w.worker, patch)
}
