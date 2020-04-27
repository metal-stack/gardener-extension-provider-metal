package worker

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/extensions/pkg/util"
	confighelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/helper"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	apismetalhelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
	metalv1alpha1 "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/v1alpha1"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	logger = log.Log.WithName("metal-worker-controller")
)

// GetMachineImages returns the used machine images for the `Worker` resource.
func (w *workerDelegate) GetMachineImages(ctx context.Context) (runtime.Object, error) {
	if w.machineImages == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return nil, err
		}
	}

	var (
		workerStatus = &apismetal.WorkerStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: apismetal.SchemeGroupVersion.String(),
				Kind:       "WorkerStatus",
			},
			MachineImages: w.machineImages,
		}

		workerStatusV1alpha1 = &metalv1alpha1.WorkerStatus{
			TypeMeta: metav1.TypeMeta{
				APIVersion: metalv1alpha1.SchemeGroupVersion.String(),
				Kind:       "WorkerStatus",
			},
		}
	)

	if err := w.scheme.Convert(workerStatus, workerStatusV1alpha1, nil); err != nil {
		return nil, err
	}

	return workerStatusV1alpha1, nil
}

func (w *workerDelegate) findMachineImage(name, version string) (string, error) {
	machineImage, err := confighelper.FindImage(w.machineImageMapping, name, version)
	if err == nil {
		return machineImage, nil
	}

	// Try to look up machine image in worker provider status as it was not found in componentconfig.
	if providerStatus := w.worker.Status.ProviderStatus; providerStatus != nil {
		workerStatus := &apismetal.WorkerStatus{}
		if _, _, err := w.decoder.Decode(providerStatus.Raw, nil, workerStatus); err != nil {
			return "", errors.Wrapf(err, "could not decode worker status of worker '%s'", util.ObjectName(w.worker))
		}

		machineImage, err := apismetalhelper.FindMachineImage(workerStatus.MachineImages, name, version)
		if err != nil {
			return "", errorMachineImageNotFound(name, version)
		}

		return machineImage.Image, nil
	}

	return "", errorMachineImageNotFound(name, version)
}

func errorMachineImageNotFound(name, version string) error {
	return fmt.Errorf("could not find machine image for %s/%s neither in componentconfig nor in worker status", name, version)
}

func appendMachineImage(machineImages []apismetal.MachineImage, machineImage apismetal.MachineImage) []apismetal.MachineImage {
	if _, err := apismetalhelper.FindMachineImage(machineImages, machineImage.Name, machineImage.Version); err != nil {
		return append(machineImages, machineImage)
	}
	return machineImages
}
