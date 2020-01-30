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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
// resource.
type CloudProfileConfig struct {
	metav1.TypeMeta
	// FirewallImages is a list of available firewall images.
	FirewallImages []string
	// FirewallNetworks contains a list of available networks withing a partition
	FirewallNetworks map[string][]string
	// IAMConfig contains the config for all AuthN/AuthZ related components, can be overriden in shoots control plane config
	IAMConfig *IAMConfig
}

// IAMConfig contains the config for all AuthN/AuthZ related components
type IAMConfig struct {
	IssuerConfig *IssuerConfig
	IdmConfig    *IDMConfig
	GroupConfig  *NamespaceGroupConfig
}

// IssuerConfig contains configuration settings for the token issuer.
type IssuerConfig struct {
	Url      string
	ClientId string
}

// IDMConfig contains config for the IDM-System that is used as directory for users and groups
type IDMConfig struct {
	Idmtype string

	ConnectorConfig *ConnectorConfig
}

// NamespaceGroupConfig for group-rolebinding-controller
type NamespaceGroupConfig struct {
	// no action is taken or any namespace in this list
	// kube-system,kube-public,kube-node-lease,default
	ExcludedNamespaces string
	// for each element a RoleBinding is created in any Namespace - ClusterRoles are bound with this name
	// admin,edit,view
	ExpectedGroupsList string
	// Maximum length of namespace-part in clusterGroupname and therefore in the corresponding groupname in the directory.
	// 20 chars für AD, given the FITS-naming-conventions
	NamespaceMaxLength int
	// The created RoleBindings will reference this group (from token).
	// oidc:{{ .Namespace }}-{{ .Group }}
	ClusterGroupnameTemplate string
	// The RoleBindings will created with this name.
	// oidc-{{ .Namespace }}-{{ .Group }}
	RoleBindingNameTemplate string
}

// ConnectorConfig optional config for the IDM Webhook - if it should be used to automatically create/delete groups/roles in the tenant IDM
type ConnectorConfig struct {
	IdmApiUrl            string
	IdmApiUser           string
	IdmApiPassword       string
	IdmSystemId          string
	IdmAccessCode        string
	IdmCustomerId        string
	IdmGroupOU           string
	IdmGroupnameTemplate string
	IdmDomainName        string
	IdmTenantPrefix      string
	IdmSubmitter         string
	IdmJobInfo           string
	IdmReqSystem         string
	IdmReqUser           string
	IdmReqEMail          string
}
