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

package metal

import (
	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControlPlaneConfig contains configuration settings for the control plane.
type ControlPlaneConfig struct {
	metav1.TypeMeta

	// CloudControllerManager contains configuration settings for the cloud-controller-manager.
	// +optional
	CloudControllerManager *CloudControllerManagerConfig

	// IAMConfig contains the config for all AuthN/AuthZ related components
	IAMConfig IAMConfig `json:"iamconfig" optional:"false"`
}

// CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
type CloudControllerManagerConfig struct {
	gardenv1beta1.KubernetesConfig `json:",inline"`
}

// IAMConfig contains the config for all AuthN/AuthZ related components
type IAMConfig struct {
	IssuerConfig *IssuerConfig         `json:"issuerConfig,omitempty"`
	IdmConfig    *IDMConfig            `json:"idmConfig,omitempty"`
	GroupConfig  *NamespaceGroupConfig `json:"groupConfig,omitempty"`
}

// TokenIssuer contains configuration settings for the token issuer.
type IssuerConfig struct {
	Url      string `json:"url,omitempty"`
	ClientId string `json:"clientId,omitempty"`
}

// IDMConfig contains config for the IDM-System that is used as directory for users and groups
type IDMConfig struct {
	Idmtype string `json:"idmtype,omitempty"`

	ConnectorConfig *ConnectorConfig `json:"connectorConfig,omitempty"`
}

// NamespaceGroupConfig for group-rolebinding-controller
type NamespaceGroupConfig struct {
	// no action is taken or any namespace in this list
	// kube-system,kube-public,kube-node-lease,default
	ExcludedNamespaces string `json:"excludedNamespaces,omitempty"`
	// for each element a RoleBinding is created in any Namespace - ClusterRoles are bound with this name
	// admin,edit,view
	ExpectedGroupsList string `json:"expectedGroupsList,omitempty"`
	// Maximum length of namespace-part in clusterGroupname and therefore in the corresponding groupname in the directory.
	// 20 chars für AD, given the FITS-naming-conventions
	NamespaceMaxLength int `json:"namespaceMaxLength,omitempty"`
	// The created RoleBindings will reference this group (from token).
	// oidc:{{ .Namespace }}-{{ .Group }}
	ClusterGroupnameTemplate string `json:"clusterGroupnameTemplate,omitempty"`
	// The RoleBindings will created with this name.
	// oidc-{{ .Namespace }}-{{ .Group }}
	RoleBindingNameTemplate string `json:"roleBindingNameTemplate,omitempty"`
}

// ConnectorConfig optional config for the IDM Webhook - if it should be used to automatically create/delete groups/roles in the tenant IDM
type ConnectorConfig struct {
	IdmApiUrl            string `json:"idmApiUrl,omitempty"`
	IdmApiUser           string `json:"idmApiUser,omitempty"`
	IdmApiPassword       string `json:"idmApiPassword,omitempty"`
	IdmSystemId          string `json:"idmSystemId,omitempty"`
	IdmAccessCode        string `json:"idmAccessCode,omitempty"`
	IdmCustomerId        string `json:"idmCustomerId,omitempty"`
	IdmGroupOU           string `json:"idmGroupOU,omitempty"`
	IdmGroupnameTemplate string `json:"idmGroupnameTemplate,omitempty"`
	IdmDomainName        string `json:"idmDomainName,omitempty"`
	IdmTenantPrefix      string `json:"idmTenantPrefix,omitempty"`
	IdmSubmitter         string `json:"idmSubmitter,omitempty"`
	IdmJobInfo           string `json:"idmJobInfo,omitempty"`
	IdmReqSystem         string `json:"idmReqSystem,omitempty"`
	IdmReqUser           string `json:"idmReqUser,omitempty"`
	IdmReqEMail          string `json:"idmReqEMail,omitempty"`
}
