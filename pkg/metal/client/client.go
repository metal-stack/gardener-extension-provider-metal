// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package client

import (
	"context"
	"fmt"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	gardencorev1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/metal-pod/gardener-extension-provider-metal/pkg/metal"
	metalgo "github.com/metal-pod/metal-go"
	"github.com/metal-pod/metal-go/api/models"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClient returns a new metal client with the provider credentials from a given secret reference.
func NewClient(ctx context.Context, k8sClient client.Client, secretRef *corev1.SecretReference) (*metalgo.Driver, error) {
	credentials, err := ReadCredentialsFromSecretRef(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, err
	}

	return NewClientFromCredentials(credentials)
}

// NewClientFromCredentials returns a new metal client with the client constructed from the given credentials.
func NewClientFromCredentials(credentials *metal.Credentials) (*metalgo.Driver, error) {
	client, err := metalgo.NewDriver(credentials.MetalAPIURL, credentials.MetalAPIKey, credentials.MetalAPIHMac)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ReadCredentialsFromSecretRef returns metal credentials from the provider credentials from a given secret reference.
func ReadCredentialsFromSecretRef(ctx context.Context, k8sClient client.Client, secretRef *corev1.SecretReference) (*metal.Credentials, error) {
	providerSecret, err := extensionscontroller.GetSecretByReference(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, err
	}

	credentials, err := metal.ReadCredentialsSecret(providerSecret)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// GetPrivateNetworkFromNodeNetwork returns the private network that belongs to the given node network cidr and project.
func GetPrivateNetworkFromNodeNetwork(client *metalgo.Driver, projectID string, nodeNetworkCIDR *gardencorev1.CIDR) (*models.V1NetworkResponse, error) {
	if nodeNetworkCIDR == nil {
		return nil, fmt.Errorf("node network cidr is nil")
	}
	prefix := string(*nodeNetworkCIDR)

	networkFindRequest := metalgo.NetworkFindRequest{
		ProjectID: &projectID,
		Prefixes:  []string{prefix},
	}
	networkFindResponse, err := client.NetworkFind(&networkFindRequest)
	if err != nil {
		return nil, err
	}
	if len(networkFindResponse.Networks) != 1 {
		return nil, fmt.Errorf("no distinct private network for project id %q and prefix %s found", projectID, prefix)
	}
	return networkFindResponse.Networks[0], nil
}
