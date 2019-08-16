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
	"fmt"
	"path"

	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EtcdMainVolumeClaimTemplateName is the name of the volume claim template in the etcd-main StatefulSet. It uses a
// different naming scheme because Gardener was using HDD-based volumes for etcd in the past and did migrate to fast
// SSD volumes recently. Due to the migration of the data of the old volume to the new one the PVC name is now different.
const EtcdMainVolumeClaimTemplateName = "main-etcd"

// GetBackupRestoreContainer returns an etcd backup-restore container with the given name, schedule, provider, image,
// and additional provider-specific command line args and env variables.
func GetBackupRestoreContainer(
	name, volumeClaimTemplateName, schedule, provider, prefix, image string,
	args map[string]string,
	env []corev1.EnvVar,
	volumeMounts []corev1.VolumeMount,
) *corev1.Container {
	c := &corev1.Container{
		Name: "backup-restore",
		Command: []string{
			"etcdbrctl",
			"server",
			fmt.Sprintf("--schedule=%s", schedule),
			"--data-dir=/var/etcd/data/new.etcd",
			fmt.Sprintf("--storage-provider=%s", provider),
			fmt.Sprintf("--store-prefix=%s", path.Join(prefix, name)),
			"--cert=/var/etcd/ssl/client/tls.crt",
			"--key=/var/etcd/ssl/client/tls.key",
			"--cacert=/var/etcd/ssl/ca/ca.crt",
			"--insecure-transport=false",
			"--insecure-skip-tls-verify=false",
			fmt.Sprintf("--endpoints=https://%s-0:2379", name),
			"--etcd-connection-timeout=300",
			"--delta-snapshot-period-seconds=300",
			"--delta-snapshot-memory-limit=104857600", // 100MB
			"--garbage-collection-period-seconds=43200",
			"--snapstore-temp-directory=/var/etcd/data/temp",
		},
		Env:             []corev1.EnvVar{},
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{
				Name:          "server",
				ContainerPort: 8080,
				Protocol:      corev1.ProtocolTCP,
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("23m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("2Gi"),
			},
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeClaimTemplateName,
				MountPath: "/var/etcd/data",
			},
			{
				Name:      "ca-etcd",
				MountPath: "/var/etcd/ssl/ca",
			},
			{
				Name:      "etcd-client-tls",
				MountPath: "/var/etcd/ssl/client",
			},
		},
	}

	// Ensure additional command line args
	for k, v := range args {
		c.Command = extensionswebhook.EnsureStringWithPrefix(c.Command, fmt.Sprintf("--%s=", k), v)
	}

	// Ensure additional env variables
	for _, envVar := range env {
		c.Env = extensionswebhook.EnsureEnvVarWithName(c.Env, envVar)
	}

	// Ensure additional volume mounts
	for _, volumeMount := range volumeMounts {
		c.VolumeMounts = extensionswebhook.EnsureVolumeMountWithName(c.VolumeMounts, volumeMount)
	}

	return c
}

func GetETCDVolumeClaimTemplate(name string, storageClassName *string, storageCapacity *resource.Quantity) *corev1.PersistentVolumeClaim {
	// Determine the storage capacity
	// A non-default storage capacity is used only if it's configured
	capacity := resource.MustParse("10Gi")
	if storageCapacity != nil {
		capacity = *storageCapacity
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: capacity,
				},
			},
		},
	}
}
