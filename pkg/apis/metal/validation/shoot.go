package validation

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/gardener/gardener/pkg/apis/core"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateClusterName validates the cluster name of the given shoot.
func ValidateClusterName(shoot *core.Shoot) field.ErrorList {
	var (
		clusterNameRegex      = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
		validClusterNameRegex = regexp.MustCompile(clusterNameRegex)
		reservedClusterName   = "all"
	)

	allErrs := field.ErrorList{}

	f := field.NewPath("metadata", "clusterName")
	clusterName := shoot.ClusterName
	if clusterName == "" {
		clusterName = shoot.ObjectMeta.Name
		f = field.NewPath("metadata", "name")
	} else if shoot.ObjectMeta.Name != "" && clusterName != shoot.ObjectMeta.Name {
		allErrs = append(allErrs, field.Required(f, "cluster name differs from shoot name"))
	}
	if clusterName == "" {
		allErrs = append(allErrs, field.Required(f, "cluster name must not be empty"))
	}
	if len(clusterName) > 10 {
		allErrs = append(allErrs, field.Required(f, "cluster name length must be <= 10 chars"))
	}
	if !validClusterNameRegex.MatchString(clusterName) {
		allErrs = append(allErrs, field.Required(f, fmt.Sprintf("cluster name must comply with regex: %s", clusterNameRegex)))
	}
	if clusterName == reservedClusterName {
		allErrs = append(allErrs, field.Required(f, fmt.Sprintf("cluster name must not be reserved word %q", reservedClusterName)))
	}

	return allErrs
}

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
