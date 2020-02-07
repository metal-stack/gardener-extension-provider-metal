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
	"strings"

	extensionscontroller "github.com/gardener/gardener-extensions/pkg/controller"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
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

// GetPrivateNetworksFromNodeNetwork returns the private network that belongs to the given node network cidr and project.
func GetPrivateNetworksFromNodeNetwork(client *metalgo.Driver, projectID string, nodeNetworkCIDR string) ([]*models.V1NetworkResponse, error) {
	if nodeNetworkCIDR == "" {
		return nil, fmt.Errorf("node network cidr is empty")
	}

	networkFindRequest := metalgo.NetworkFindRequest{
		ProjectID: &projectID,
		Prefixes:  []string{nodeNetworkCIDR},
	}
	networkFindResponse, err := client.NetworkFind(&networkFindRequest)
	if err != nil {
		return nil, err
	}
	return networkFindResponse.Networks, nil
}

// GetPrivateNetworkFromNodeNetwork returns the private network that belongs to the given node network cidr and project.
func GetPrivateNetworkFromNodeNetwork(client *metalgo.Driver, projectID string, nodeNetworkCIDR string) (*models.V1NetworkResponse, error) {
	privateNetworks, err := GetPrivateNetworksFromNodeNetwork(client, projectID, nodeNetworkCIDR)
	if err != nil {
		return nil, err
	}
	if len(privateNetworks) != 1 {
		return nil, fmt.Errorf("no distinct private network for project id %q and prefix %s found", projectID, nodeNetworkCIDR)
	}
	return privateNetworks[0], nil
}

// GetEphemeralIPsFromCluster return all ephemeral IPs for given project and cluster
func GetEphemeralIPsFromCluster(client *metalgo.Driver, projectID, clusterID string) ([]*models.V1IPResponse, []*models.V1IPResponse, error) {
	ephemeral := metalgo.IPTypeEphemeral
	ipFindRequest := metalgo.IPFindRequest{
		ProjectID: &projectID,
		Type:      &ephemeral,
	}
	ipFindResponse, err := client.IPFind(&ipFindRequest)
	if err != nil {
		return nil, nil, err
	}
	// only these who are member of one cluster are freed
	ipsToFree := []*models.V1IPResponse{}
	// those who are member of more clusters must be updated and the tags which references this cluster must be removed.
	ipsToUpdate := []*models.V1IPResponse{}
	for _, ip := range ipFindResponse.IPs {
		clusterCount := 0
		for _, t := range ip.Tags {
			if metalgo.TagIsMemberOfCluster(t, clusterID) {
				clusterCount++
			}
		}
		if clusterCount == 1 {
			ipsToFree = append(ipsToFree, ip)
			continue
		}
		// IPs which are used in more than one cluster must be updated to get the tags with this clusterid removed
		ipsToUpdate = append(ipsToUpdate, ip)
	}
	return ipsToFree, ipsToUpdate, nil
}

// UpdateIPInCluster update the IP in the cluster to have only these tags left which are not from this cluster
func UpdateIPInCluster(client *metalgo.Driver, ip *models.V1IPResponse, clusterID string) error {
	clusterTag := metalgo.BuildServiceTagClusterPrefix(clusterID)

	var newTags []string
	for _, t := range ip.Tags {
		if strings.HasPrefix(t, clusterTag) {
			continue
		}
		newTags = append(newTags, t)
	}
	iur := &metalgo.IPUpdateRequest{
		IPAddress: *ip.Ipaddress,
		Tags:      newTags,
	}
	_, err := client.IPUpdate(iur)
	if err != nil {
		return err
	}
	return nil
}
