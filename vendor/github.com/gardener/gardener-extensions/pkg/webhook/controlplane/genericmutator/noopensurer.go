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

package genericmutator

import (
	"context"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"

	"github.com/coreos/go-systemd/unit"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
)

// NoopEnsurer provides no-op implementation of Ensurer. This can be anonymously composed by actual Ensurers for convenience.
type NoopEnsurer struct{}

// EnsureKubeAPIServerService ensures that the kube-apiserver service conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeAPIServerService(context.Context, *corev1.Service) error {
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeAPIServerDeployment(context.Context, *appsv1.Deployment) error {
	return nil
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeControllerManagerDeployment(context.Context, *appsv1.Deployment) error {
	return nil
}

// EnsureKubeSchedulerDeployment ensures that the kube-scheduler deployment conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeSchedulerDeployment(context.Context, *appsv1.Deployment) error {
	return nil
}

// EnsureETCDStatefulSet ensures that the etcd stateful sets conform to the provider requirements.
func (e *NoopEnsurer) EnsureETCDStatefulSet(context.Context, *appsv1.StatefulSet, *extensionscontroller.Cluster) error {
	return nil
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *NoopEnsurer) EnsureKubeletServiceUnitOptions(_ context.Context, opts []*unit.UnitOption) ([]*unit.UnitOption, error) {
	return opts, nil
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubeletConfiguration(context.Context, *kubeletconfigv1beta1.KubeletConfiguration) error {
	return nil
}

// EnsureKubernetesGeneralConfiguration ensures that the kubernetes general configuration conforms to the provider requirements.
func (e *NoopEnsurer) EnsureKubernetesGeneralConfiguration(context.Context, *string) error {
	return nil
}
