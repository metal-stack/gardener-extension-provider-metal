package validation

import (
	"fmt"
	"sort"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateWorkers validates the workers of a Shoot.
func ValidateWorkers(workers []core.Worker, cloudProfile *gardencorev1beta1.CloudProfile, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	availableImages := sets.NewString()
	for _, image := range cloudProfile.Spec.MachineImages {
		for _, version := range image.Versions {
			availableImages.Insert(image.Name + "-" + version.Version)
		}
	}

	for i, worker := range workers {
		if worker.Volume != nil {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("volume"), "volumes are not yet supported and must be nil"))
		}

		if len(worker.Zones) != 0 {
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("zones"), "zone spreading is not supported, specify partition via infrastructure config"))
		}

		wantedImage := worker.Machine.Image.Name + "-" + worker.Machine.Image.Version
		if !availableImages.Has(wantedImage) {
			sortedImages := availableImages.List()
			sort.Strings(sortedImages)
			allErrs = append(allErrs, field.Required(fldPath.Index(i).Child("machine.image"), fmt.Sprintf("machine image %s in version %s not offered, available images are: %v", worker.Machine.Image.Name, worker.Machine.Image.Version, sortedImages)))
		}
	}

	return allErrs
}
