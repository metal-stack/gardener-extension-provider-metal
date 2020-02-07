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
	cloudgo "github.com/metal-stack/cloud-go"
	cloudclient "github.com/metal-stack/cloud-go/api/client"
	"github.com/metal-stack/cloud-go/api/client/project"
	"github.com/metal-stack/cloud-go/api/models"

	"github.com/metal-stack/gardener-extension-provider-metal/pkg/metal"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewCloudClient returns a new cloud client with the provider credentials from a given secret reference.
func NewCloudClient(ctx context.Context, k8sClient client.Client, secretRef *corev1.SecretReference) (*cloudclient.Cloud, error) {
	credentials, err := ReadCredentialsFromSecretRef(ctx, k8sClient, secretRef)
	if err != nil {
		return nil, err
	}

	return NewCloudClientFromCredentials(credentials)
}

// NewCloudClientFromCredentials returns a new cloud client with the client constructed from the given credentials.
func NewCloudClientFromCredentials(credentials *metal.Credentials) (*cloudclient.Cloud, error) {
	client, err := cloudgo.NewClient(credentials.CloudAPIURL, credentials.CloudAPIKey, credentials.CloudAPIHMac)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// GetProjectByID returns a project by a given ID
func GetProjectByID(client *cloudclient.Cloud, projectID string) (*models.V1Project, error) {
	params := project.NewFindProjectParams().WithID(projectID)
	resp, err := client.Project.FindProject(params, nil)
	if err != nil {
		switch e := err.(type) {
		case *project.FindProjectDefault:
			return nil, e
		}
		return nil, err
	}
	return resp.Payload.Project, nil
}
