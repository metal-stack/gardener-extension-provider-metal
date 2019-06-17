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

package controlplane

import (
	"context"

	"github.com/gardener/gardener-extensions/controllers/provider-openstack/pkg/openstack"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"

	"github.com/coreos/go-systemd/unit"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubeletconfigv1beta1 "k8s.io/kubelet/config/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplane ensurer.
func NewEnsurer(logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		logger: logger.WithName("openstack-controlplane-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	client client.Client
	logger logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureKubeAPIServerDeployment ensures that the kube-apiserver deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeAPIServerDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	template := &dep.Spec.Template
	ps := &template.Spec
	if c := controlplane.ContainerWithName(ps.Containers, "kube-apiserver"); c != nil {
		ensureKubeAPIServerCommandLineArgs(c)
		ensureVolumeMounts(c)
	}
	ensureVolumes(ps)
	return e.ensureChecksumAnnotations(ctx, &dep.Spec.Template, dep.Namespace)
}

// EnsureKubeControllerManagerDeployment ensures that the kube-controller-manager deployment conforms to the provider requirements.
func (e *ensurer) EnsureKubeControllerManagerDeployment(ctx context.Context, dep *appsv1.Deployment) error {
	template := &dep.Spec.Template
	ps := &template.Spec
	if c := controlplane.ContainerWithName(ps.Containers, "kube-controller-manager"); c != nil {
		ensureKubeControllerManagerCommandLineArgs(c)
		ensureVolumeMounts(c)
	}
	ensureVolumes(ps)
	return e.ensureChecksumAnnotations(ctx, &dep.Spec.Template, dep.Namespace)
}

func ensureKubeAPIServerCommandLineArgs(c *corev1.Container) {
	c.Command = controlplane.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "openstack")
	c.Command = controlplane.EnsureStringWithPrefix(c.Command, "--cloud-config=",
		"/etc/kubernetes/cloudprovider/cloudprovider.conf")
	c.Command = controlplane.EnsureStringWithPrefixContains(c.Command, "--enable-admission-plugins=",
		"PersistentVolumeLabel", ",")
	c.Command = controlplane.EnsureNoStringWithPrefixContains(c.Command, "--disable-admission-plugins=",
		"PersistentVolumeLabel", ",")
}

func ensureKubeControllerManagerCommandLineArgs(c *corev1.Container) {
	c.Command = controlplane.EnsureStringWithPrefix(c.Command, "--cloud-provider=", "external")
	c.Command = controlplane.EnsureStringWithPrefix(c.Command, "--cloud-config=",
		"/etc/kubernetes/cloudprovider/cloudprovider.conf")
	c.Command = controlplane.EnsureStringWithPrefix(c.Command, "--external-cloud-volume-plugin=", "openstack")
}

var (
	cloudProviderConfigVolumeMount = corev1.VolumeMount{
		Name:      openstack.CloudProviderConfigName,
		MountPath: "/etc/kubernetes/cloudprovider",
	}
	cloudProviderConfigVolume = corev1.Volume{
		Name: openstack.CloudProviderConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: openstack.CloudProviderConfigName},
			},
		},
	}
)

func ensureVolumeMounts(c *corev1.Container) {
	c.VolumeMounts = controlplane.EnsureVolumeMountWithName(c.VolumeMounts, cloudProviderConfigVolumeMount)
}

func ensureVolumes(ps *corev1.PodSpec) {
	ps.Volumes = controlplane.EnsureVolumeWithName(ps.Volumes, cloudProviderConfigVolume)
}

func (e *ensurer) ensureChecksumAnnotations(ctx context.Context, template *corev1.PodTemplateSpec, namespace string) error {
	if err := controlplane.EnsureConfigMapChecksumAnnotation(ctx, template, e.client, namespace, openstack.CloudProviderConfigName); err != nil {
		return err
	}
	return nil
}

// EnsureKubeletServiceUnitOptions ensures that the kubelet.service unit options conform to the provider requirements.
func (e *ensurer) EnsureKubeletServiceUnitOptions(ctx context.Context, opts []*unit.UnitOption) ([]*unit.UnitOption, error) {
	if opt := controlplane.UnitOptionWithSectionAndName(opts, "Service", "ExecStart"); opt != nil {
		command := controlplane.DeserializeCommandLine(opt.Value)
		command = ensureKubeletCommandLineArgs(command)
		opt.Value = controlplane.SerializeCommandLine(command, 1, " \\\n    ")
	}
	return opts, nil
}

func ensureKubeletCommandLineArgs(command []string) []string {
	command = controlplane.EnsureStringWithPrefix(command, "--cloud-provider=", "openstack")
	return command
}

// EnsureKubeletConfiguration ensures that the kubelet configuration conforms to the provider requirements.
func (e *ensurer) EnsureKubeletConfiguration(ctx context.Context, kubeletConfig *kubeletconfigv1beta1.KubeletConfiguration) error {
	// Make sure CSI-related feature gates are not enabled
	// TODO Leaving these enabled shouldn't do any harm, perhaps remove this code when properly tested?
	delete(kubeletConfig.FeatureGates, "VolumeSnapshotDataSource")
	delete(kubeletConfig.FeatureGates, "CSINodeInfo")
	delete(kubeletConfig.FeatureGates, "CSIDriverRegistry")
	return nil
}
