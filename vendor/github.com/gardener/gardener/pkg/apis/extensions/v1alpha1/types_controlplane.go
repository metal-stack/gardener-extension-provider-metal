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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ControlPlaneResource is a constant for the name of the ControlPlane resource.
const ControlPlaneResource = "ControlPlane"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControlPlane is a specification for a ControlPlane resource.
type ControlPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ControlPlaneSpec   `json:"spec"`
	Status ControlPlaneStatus `json:"status"`
}

// GetExtensionType returns the type of this ControlPlane resource.
func (cp *ControlPlane) GetExtensionType() string {
	return cp.Spec.Type
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControlPlaneList is a list of ControlPlane resources.
type ControlPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of ControlPlanes.
	Items []ControlPlane `json:"items"`
}

// ControlPlaneSpec is the spec of a ControlPlane resource.
type ControlPlaneSpec struct {
	// DefaultSpec is a structure containing common fields used by all extension resources.
	DefaultSpec `json:",inline"`

	// ProviderConfig contains provider-specific configuration for this control plane.
	// +optional
	ProviderConfig *runtime.RawExtension `json:"providerConfig,omitempty"`
	// InfrastructureProviderStatus contains the provider status that has
	// been generated by the controller responsible for the `Infrastructure` resource.
	// +optional
	InfrastructureProviderStatus *runtime.RawExtension `json:"infrastructureProviderStatus,omitempty"`
	// Region is the region of this control plane.
	Region string `json:"region"`
	// SecretRef is a reference to a secret that contains the cloud provider specific credentials.
	SecretRef corev1.SecretReference `json:"secretRef"`
}

// ControlPlaneStatus is the status of a ControlPlane resource.
type ControlPlaneStatus struct {
	// DefaultStatus is a structure containing common fields used by all extension resources.
	DefaultStatus `json:",inline"`

	// ProviderStatus contains provider-specific output for this control plane.
	// +optional
	ProviderStatus *runtime.RawExtension `json:"providerStatus,omitempty"`
}
