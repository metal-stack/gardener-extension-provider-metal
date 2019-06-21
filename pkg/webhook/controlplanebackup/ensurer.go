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

package controlplanebackup

import (
	"context"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane"
	"github.com/gardener/gardener-extensions/pkg/webhook/controlplane/genericmutator"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/apis/config"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"

	"github.com/gardener/gardener/pkg/operation/common"
	"github.com/gardener/gardener/pkg/utils/imagevector"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewEnsurer creates a new controlplaneexposure ensurer.
func NewEnsurer(etcdBackup *config.ETCDBackup, imageVector imagevector.ImageVector, logger logr.Logger) genericmutator.Ensurer {
	return &ensurer{
		etcdBackup:  etcdBackup,
		imageVector: imageVector,
		logger:      logger.WithName("metal-controlplanebackup-ensurer"),
	}
}

type ensurer struct {
	genericmutator.NoopEnsurer
	etcdBackup  *config.ETCDBackup
	imageVector imagevector.ImageVector
	client      client.Client
	logger      logr.Logger
}

// InjectClient injects the given client into the ensurer.
func (e *ensurer) InjectClient(client client.Client) error {
	e.client = client
	return nil
}

// EnsureETCDStatefulSet ensures that the etcd stateful sets conform to the provider requirements.
func (e *ensurer) EnsureETCDStatefulSet(ctx context.Context, ss *appsv1.StatefulSet, cluster *extensionscontroller.Cluster) error {
	if err := e.ensureContainers(&ss.Spec.Template.Spec, ss.Name, cluster); err != nil {
		return err
	}
	return e.ensureChecksumAnnotations(ctx, &ss.Spec.Template, ss.Namespace, ss.Name)
}

func (e *ensurer) ensureContainers(ps *corev1.PodSpec, name string, cluster *extensionscontroller.Cluster) error {
	// FIXME remove next line, is only here for debugging purpose.
	return nil
	c, err := e.getBackupRestoreContainer(name, cluster)
	if err != nil {
		return err
	}

	ps.Containers = controlplane.EnsureContainerWithName(ps.Containers, *c)
	return nil
}

func (e *ensurer) ensureChecksumAnnotations(ctx context.Context, template *corev1.PodTemplateSpec, namespace, name string) error {
	if name == common.EtcdMainStatefulSetName {
		return controlplane.EnsureSecretChecksumAnnotation(ctx, template, e.client, namespace, metal.BackupSecretName)
	}
	return nil
}

func (e *ensurer) getBackupRestoreContainer(name string, cluster *extensionscontroller.Cluster) (*corev1.Container, error) {
	// Find etcd-backup-restore image
	// TODO Get seed version from clientset when it's possible to inject it
	image, err := e.imageVector.FindImage(metal.ETCDBackupRestoreImageName, "", cluster.Shoot.Spec.Kubernetes.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "could not find image %s", metal.ETCDBackupRestoreImageName)
	}

	return controlplane.GetBackupRestoreContainer(name, name, "", "", image.String(), nil, nil, nil), nil
}
