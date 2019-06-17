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

package util

import (
	"context"
	"fmt"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/operation/common"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"
	"github.com/gardener/gardener/pkg/utils/secrets"

	"github.com/Masterminds/semver"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CAChecksumAnnotation is a resource annotation used to store the checksum of a certificate authority.
const CAChecksumAnnotation = "checksum/ca"

// GetGardenerSecret gets the secret from the given namespace which contains certificate information
// as well as the Kubeconfig for a Shoot cluster.
func GetGardenerSecret(ctx context.Context, client client.Client, namespace string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, kutil.Key(namespace, gardencorev1alpha1.SecretNameGardener), secret); err != nil {
		return nil, err
	}
	return secret, nil
}

// GetOrCreateShootKubeconfig gets or creates a Kubeconfig for a Shoot cluster which has a running control plane in the given `namespace`.
// If the CA of an existing Kubeconfig has changed, it creates a new Kubeconfig.
// Newly generated Kubeconfigs are applied with the given `client` to the given `namespace`.
func GetOrCreateShootKubeconfig(ctx context.Context, client client.Client, certificateConfig secrets.CertificateSecretConfig, namespace string) (*corev1.Secret, error) {
	caSecret, ca, err := secrets.LoadCAFromSecret(client, namespace, gardencorev1alpha1.SecretNameCACluster)
	if err != nil {
		return nil, fmt.Errorf("error fetching CA secret %s/%s: %v", namespace, gardencorev1alpha1.SecretNameCACluster, err)
	}

	var (
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: make(map[string]string),
				Name:        certificateConfig.Name,
				Namespace:   namespace,
			},
		}
		key = types.NamespacedName{
			Name:      certificateConfig.Name,
			Namespace: namespace,
		}
	)
	if err := client.Get(ctx, key, &secret); err != nil && !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("error preparing kubeconfig: %v", err)
	}

	var (
		computedChecksum   = ComputeChecksum(caSecret.Data)
		storedChecksum, ok = secret.Annotations[CAChecksumAnnotation]
	)
	if ok && computedChecksum == storedChecksum {
		return &secret, nil
	}

	certificateConfig.SigningCA = ca
	certificateConfig.CertType = secrets.ClientCert

	config := secrets.ControlPlaneSecretConfig{
		CertificateSecretConfig: &certificateConfig,

		KubeConfigRequest: &secrets.KubeConfigRequest{
			ClusterName:  namespace,
			APIServerURL: kubeAPIServerServiceDNS(namespace),
		},
	}

	controlPlane, err := config.GenerateControlPlane()
	if err != nil {
		return nil, fmt.Errorf("error creating kubeconfig: %v", err)
	}

	return &secret, kutil.CreateOrUpdate(ctx, client, &secret, func() error {
		secret.Data = controlPlane.SecretData()
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		secret.Annotations[CAChecksumAnnotation] = computedChecksum
		return nil
	})
}

// KubeAPIServerServiceDNS returns a domain name which can be used to contact
// the Kube-Apiserver deployment of a Shoot within the Seed cluster.
// e.g. kube-apiserver.shoot--project--prod.svc.cluster.local.
func kubeAPIServerServiceDNS(namespace string) string {
	return fmt.Sprintf("%s.%s", common.KubeAPIServerDeploymentName, namespace)
}

// GetReplicaCount returns the given replica count base on the hibernation status of the shoot.
func GetReplicaCount(shoot *gardenv1beta1.Shoot, count int) int {
	if shoot.Spec.Hibernation != nil && shoot.Spec.Hibernation.Enabled {
		return 0
	}
	return count
}

// VersionMajorMinor extracts and returns the major and the minor part of the given version (input must be a semantic version).
func VersionMajorMinor(version string) (string, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d", v.Major(), v.Minor()), nil
}
