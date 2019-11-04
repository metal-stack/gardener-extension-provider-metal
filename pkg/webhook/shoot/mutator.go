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

package shoot

import (
	"context"
	"fmt"

	extensionswebhook "github.com/gardener/gardener-extensions/pkg/webhook"
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardenerkubernetes "github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"
	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type mutator struct {
	client client.Client
	logger logr.Logger
}

const (
	droptailerNamespace        = "droptailer"
	droptailerDeploymentName   = "droptailer"
	droptailerClientSecretName = "droptailer-client"
	droptailerServerSecretName = "droptailer-server"
)

// NewMutator creates a new Mutator that mutates resources in the shoot cluster.
func NewMutator(logger logr.Logger) extensionswebhook.MutatorWithShootClient {
	return &mutator{
		logger: logger,
	}
}

func (m *mutator) Mutate(ctx context.Context, obj runtime.Object, shootClient client.Client) error {
	acc, err := meta.Accessor(obj)
	if err != nil {
		return errors.Wrapf(err, "could not create accessor during webhook")
	}
	// If the object does have a deletion timestamp then we don't want to mutate anything.
	if acc.GetDeletionTimestamp() != nil {
		return nil
	}

	switch x := obj.(type) {
	case *appsv1.Deployment:
		switch x.Name {
		case droptailerDeploymentName:
			extensionswebhook.LogMutation(logger, x.Kind, x.Namespace, x.Name)
			return m.mutateDroptailerDeployment(ctx, shootClient, x)
		}
	}
	return nil
}

func (m *mutator) mutateDroptailerDeployment(ctx context.Context, shootClient client.Client, d *appsv1.Deployment) error {
	wanted := &secrets.Secrets{
		CertificateSecretConfigs: map[string]*secrets.CertificateSecretConfig{
			gardencorev1alpha1.SecretNameCACluster: {
				Name:       gardencorev1alpha1.SecretNameCACluster,
				CommonName: "kubernetes",
				CertType:   secrets.CACert,
			},
		},
		SecretConfigsFunc: func(cas map[string]*secrets.Certificate, clusterName string) []secrets.ConfigInterface {
			return []secrets.ConfigInterface{
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         droptailerClientSecretName,
						CommonName:   "system:droptailer-client",
						Organization: []string{"droptailer-client"},
						CertType:     secrets.ClientCert,
						SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
					},
				},
				&secrets.ControlPlaneSecretConfig{
					CertificateSecretConfig: &secrets.CertificateSecretConfig{
						Name:         droptailerServerSecretName,
						CommonName:   "system:droptailer-server",
						Organization: []string{"droptailer-server"},
						CertType:     secrets.ServerCert,
						SigningCA:    cas[gardencorev1alpha1.SecretNameCACluster],
					},
				},
			}
		},
	}
	_, err := wanted.Deploy(ctx, shootClient.(kubernetes.Interface), shootClient.(gardenerkubernetes.Interface), droptailerNamespace)
	if err != nil {
		return fmt.Errorf("could not deploy droptailer secrets to shoot cluster; err: %w", err)
	}
	return nil
}
