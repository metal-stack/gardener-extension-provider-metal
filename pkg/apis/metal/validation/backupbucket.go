package validation

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var (
	secretGVK = corev1.SchemeGroupVersion.WithKind("Secret")

	allowedGVKs = sets.New(secretGVK)
	validGVKs   = []string{secretGVK.String()}
)

// ValidateBackupBucketCredentialsRef validates credentialsRef is set to supported kind of credentials.
func ValidateBackupBucketCredentialsRef(credentialsRef *corev1.ObjectReference, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if credentialsRef == nil {
		return append(allErrs, field.Required(fldPath, "must be set"))
	}

	if !allowedGVKs.Has(credentialsRef.GroupVersionKind()) {
		allErrs = append(allErrs, field.NotSupported(fldPath, credentialsRef.String(), validGVKs))
	}

	return allErrs
}
