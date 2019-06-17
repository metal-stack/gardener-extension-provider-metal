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
	"encoding/json"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/utils"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ContextFromStopChannel creates a new context from a given stop channel.
func ContextFromStopChannel(stopCh <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		<-stopCh
	}()

	return ctx
}

// ComputeChecksum computes a SHA256 checksum for the give map.
func ComputeChecksum(data interface{}) string {
	jsonString, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return utils.ComputeSHA256Hex(jsonString)
}

// GetSecretByRef reads the secret given by the reference and returns it.
func GetSecretByRef(ctx context.Context, c client.Client, ref corev1.SecretReference) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, kutil.Key(ref.Namespace, ref.Name), secret)
	return secret, err
}

// GetKubeconfigFromSecret gets the Kubeconfig from the passed secret.
func GetKubeconfigFromSecret(secret *corev1.Secret) (*restclient.Config, error) {
	var (
		key            = "kubeconfig"
		kubeconfig, ok = secret.Data[key]
	)
	if !ok {
		return nil, fmt.Errorf("Key %s not available in map", key)
	}
	return clientcmd.RESTConfigFromKubeConfig(kubeconfig)
}

// ObjectName returns the name of the given object in the format <namespace>/<name>
func ObjectName(obj runtime.Object) string {
	k, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return "/"
	}
	return k.String()
}

// WaitUntilResourceDeleted deletes the given resource and then waits until it has been deleted. It respects the
// given interval and timeout.
// TODO: Remove and use https://github.com/gardener/gardener/blob/master/pkg/utils/kubernetes/kubernetes.go#L105 once
// new version of github.com/gardener/gardener can be vendored.
func WaitUntilResourceDeleted(ctx context.Context, c client.Client, obj runtime.Object, interval time.Duration) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}

	return wait.PollImmediateUntil(interval, func() (bool, error) {
		if err := c.Get(ctx, key, obj); err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	}, ctx.Done())
}
