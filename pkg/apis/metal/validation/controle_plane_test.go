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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ControlPlaneconfig validation", func() {
	var (
		controlPlaneConfig *apismetal.ControlPlaneConfig
	)

	BeforeEach(func() {
		controlPlaneConfig = &apismetal.ControlPlaneConfig{}
	})

	Describe("#ValidateControlPlaneConfig", func() {
		It("should return no errors for an unchanged config", func() {
			Expect(ValidateControlPlaneConfig(controlPlaneConfig, field.NewPath("spec"))).To(BeEmpty())
		})

		// It("should forbid empty firewall image", func() {
		// 	infrastructureConfig.Firewall.Image = ""

		// 	errorList := ValidateInfrastructureConfig(infrastructureConfig)

		// 	Expect(errorList).To(ConsistOfFields(Fields{
		// 		"Type":   Equal(field.ErrorTypeRequired),
		// 		"Field":  Equal("firewall.image"),
		// 		"Detail": Equal("firewall image must be specified"),
		// 	}))
		// })
	})
})
