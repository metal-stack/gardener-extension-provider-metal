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

package validation_test

import (
	apismetal "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal"
	. "github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/metal/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	. "github.com/gardener/gardener/pkg/utils/validation/gomega"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("ControlPlaneconfig validation", func() {
	var (
		controlPlaneConfig *apismetal.ControlPlaneConfig
	)

	BeforeEach(func() {
		controlPlaneConfig = &apismetal.ControlPlaneConfig{
			IAMConfig: apismetal.IAMConfig{
				IssuerConfig: &apismetal.IssuerConfig{
					Url:      "https://somewhere",
					ClientId: "abc",
				},
				IdmConfig: &apismetal.IDMConfig{
					Idmtype: "UX",
				},
			},
		}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))).To(BeEmpty())
		})

		It("should forbid empty issuer url", func() {
			controlPlaneConfig.IAMConfig.IssuerConfig.Url = ""

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.issuerConfig.url"),
				"Detail": Equal("url must be specified"),
			}))
		})

		It("should forbid empty client id", func() {
			controlPlaneConfig.IAMConfig.IssuerConfig.ClientId = ""

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.issuerConfig.clientId"),
				"Detail": Equal("clientId must be specified"),
			}))
		})

		It("should forbid empty idm type", func() {
			controlPlaneConfig.IAMConfig.IdmConfig.Idmtype = ""

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.idmConfig.idmtype"),
				"Detail": Equal("idmtype must be specified"),
			}))
		})

		It("should forbid group namespace length of zero", func() {
			controlPlaneConfig.IAMConfig.GroupConfig = &apismetal.NamespaceGroupConfig{NamespaceMaxLength: 0}

			errorList := ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))

			Expect(errorList).To(ConsistOfFields(Fields{
				"Type":   Equal(field.ErrorTypeRequired),
				"Field":  Equal("spec.iamconfig.groupConfig.namespaceMaxLength"),
				"Detail": Equal("namespaceMaxLength must be a positive integer"),
			}))
		})
	})
})
