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

package config

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ControllerConfiguration defines the configuration for the metal provider.
type ControllerConfiguration struct {
	metav1.TypeMeta

	// MachineImages is the list of machine images that are understood by the controller. It maps
	// logical names and versions to metal-specific identifiers, i.e. AMIs.
	MachineImages []MachineImage

	// ETCD is the etcd configuration.
	ETCD ETCD

	// Auth is configuration for metal stack specific user authentication in the cluster.
	Auth Auth

	// SplunkAudit is configuration for auditlogging the kube-apiserver to splunk endpoint via webhook.
	SplunkAudit SplunkAudit

	// AccountingExporter is the configuration for the accounting exporter
	AccountingExporter AccountingExporterConfiguration
}

// MachineImage is a mapping from logical names and versions to GCP-specific identifiers.
type MachineImage struct {
	// Name is the logical name of the machine image.
	Name string
	// Version is the logical version of the machine image.
	Version string
	// Image is the path to the image.
	Image string
}

// ETCD is an etcd configuration.
type ETCD struct {
	// ETCDStorage is the etcd storage configuration.
	Storage ETCDStorage
	// ETCDBackup is the etcd backup configuration.
	Backup ETCDBackup
}

// ETCDStorage is an etcd storage configuration.
type ETCDStorage struct {
	// ClassName is the name of the storage class used in etcd-main volume claims.
	ClassName *string
	// Capacity is the storage capacity used in etcd-main volume claims.
	Capacity *resource.Quantity
}

// ETCDBackup is an etcd backup configuration.
type ETCDBackup struct {
	// Schedule is the etcd backup schedule.
	Schedule *string
}

// Auth contains the configuration for metal stack specific user authentication in the cluster.
type Auth struct {
	// Enabled enables the deployment of metal stack specific cluster authentication when set to true.
	Enabled bool
	// ProviderTenant is the name of the provider tenant who has special privileges.
	ProviderTenant string
}

// SplunkAudit contains the configuration for auditlogging the kube-apiserver to splunk endpoint via webhook.
type SplunkAudit struct {
	// Enabled enables the deployment of splunk auditlog webhook when set to true.
	Enabled bool
}

// AccountingExporterConfiguration contains the configuration for the accounting exporter.
type AccountingExporterConfiguration struct {
	// Enabled enables the deployment of the accounting exporter when set to true.
	Enabled bool
	// NetworkTraffic contains the configuration for accounting network traffic
	NetworkTraffic AccountingExporterNetworkTrafficConfiguration
	// Client contains the configuration for the accounting exporter client.
	Client AccountingExporterClientConfiguration
}

// AccountingExporterClientConfiguration contains the configuration for the network traffic accounting.
type AccountingExporterNetworkTrafficConfiguration struct {
	// Enabled enables network traffic accounting of the accounting exporter when set to true.
	Enabled bool
	// InternalNetworks defines the networks for the firewall that are considered internal (which can be accounted differently)
	InternalNetworks []string
}

// AccountingExporterClientConfiguration contains the configuration for the accounting exporter client
type AccountingExporterClientConfiguration struct {
	// Hostname is the hostname of the accounting api
	Hostname string
	// Port is the port of the accounting api
	Port int
	// CA is the ca certificate used for communicating with the accounting api
	CA string
	// Cert is the client certificate used for communicating with the accounting api
	Cert string
	// CertKey is the client certificate key used for communicating with the accounting api
	CertKey string
}
