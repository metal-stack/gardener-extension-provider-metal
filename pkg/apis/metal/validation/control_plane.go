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

package validation

import (
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateControlPlaneConfig validates a ControlPlaneConfig object.
func ValidateControlPlaneConfig(controlPlaneConfig *apismetal.ControlPlaneConfig, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	iam := controlPlaneConfig.IAMConfig
	iamPath := fldPath.Child("iamconfig")

	issuer := iam.IssuerConfig
	issuerPath := iamPath.Child("issuerConfig")
	if issuer == nil {
		allErrs = append(allErrs, field.Required(issuerPath, "issuer config must be specified"))
	} else {
		if issuer.Url == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("url"), "url must be specified"))
		}
		if issuer.ClientId == "" {
			allErrs = append(allErrs, field.Required(issuerPath.Child("clientId"), "clientId must be specified"))
		}
	}

	idmConfig := iam.IdmConfig
	idmConfigPath := iamPath.Child("idmConfig")
	if idmConfig == nil {
		allErrs = append(allErrs, field.Required(idmConfigPath, "idm config must be specified"))
	} else {
		if idmConfig.Idmtype == "" {
			allErrs = append(allErrs, field.Required(idmConfigPath.Child("idmtype"), "idmtype must be specified"))
		}
	}

	groupConfig := iam.GroupConfig
	groupConfigPath := iamPath.Child("groupConfig")
	if groupConfig != nil {
		if groupConfig.NamespaceMaxLength <= 0 {
			allErrs = append(allErrs, field.Required(groupConfigPath.Child("namespaceMaxLength"), "namespaceMaxLength must be a positive integer"))
		}
	}

	return allErrs
}
