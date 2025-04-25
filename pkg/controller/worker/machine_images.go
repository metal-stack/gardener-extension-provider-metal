package worker

import (
	"context"
	"fmt"

	confighelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/helper"
	apismetal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	apismetalhelper "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal/helper"
)

// UpdateMachineImagesStatus implements genericactuator.WorkerDelegate.
func (w *workerDelegate) UpdateMachineImagesStatus(ctx context.Context) error {
	if w.machineImages == nil {
		if err := w.generateMachineConfig(ctx); err != nil {
			return fmt.Errorf("unable to generate the machine config %w", err)
		}
	}

	workerStatus, err := w.decodeWorkerProviderStatus()
	if err != nil {
		return fmt.Errorf("unable to decode the worker provider status %w", err)
	}

	workerStatus.MachineImages = w.machineImages
	if err := w.updateWorkerProviderStatus(ctx, workerStatus); err != nil {
		return fmt.Errorf("unable to update worker provider status %w", err)
	}

	return nil
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
			return "", fmt.Errorf("could not decode worker status of worker '%s' %w", w.worker.Name, err)
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
