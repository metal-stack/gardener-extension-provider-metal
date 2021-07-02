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

	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config"
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

	// ClusterAudit is the configuration for cluster auditing.
	ClusterAudit ClusterAudit

	// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
	AuditToSplunk AuditToSplunk

	// Auth is the configuration for metal stack specific user authentication in the cluster.
	Auth Auth

	// AccountingExporter is the configuration for the accounting exporter
	AccountingExporter AccountingExporterConfiguration

	// HealthCheckConfig is the config for the health check controller
	HealthCheckConfig *healthcheckconfig.HealthCheckConfig

	// Storage is the configuration for storage.
	Storage StorageConfiguration

	// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
	ImagePullSecret *ImagePullSecret
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
	// DeltaSnapshotPeriod is the time for delta snapshots to be made
	DeltaSnapshotPeriod *string
}

// ClusterAudit is the configuration for cluster auditing.
type ClusterAudit struct {
	// Enabled enables collecting of the kube-apiserver auditlog.
	Enabled bool
}

// AuditToSplunk is the configuration for forwarding audit (and firewall) logs to Splunk.
type AuditToSplunk struct {
	// Enabled enables collecting of the kube-apiserver auditlog.
	Enabled    bool
	HECToken   string
	Index      string
	HECHost    string
	HECPort    int
	TLSEnabled bool
	HECCAFile  string
}

// Auth contains the configuration for metal stack specific user authentication in the cluster.
type Auth struct {
	// Enabled enables the deployment of metal stack specific cluster authentication when set to true.
	Enabled bool
	// ProviderTenant is the name of the provider tenant who has special privileges.
	ProviderTenant string
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

// StorageConfiguration contains the configuration for provider specfic storage solutions.
type StorageConfiguration struct {
	// Duros contains the configuration for duros cloud storage
	Duros DurosConfiguration
}

// DurosConfiguration contains the configuration for lightbits duros storage.
type DurosConfiguration struct {
	// Enabled enables duros storage when set to true.
	Enabled bool
	// SeedConfig is a map of a seed name to the duros seed configuration
	SeedConfig map[string]DurosSeedConfiguration
}

// DurosSeedConfiguration is the configuration for duros for a particular seed
type DurosSeedConfiguration struct {
	// Endpoints is the list of endpoints of the duros API
	Endpoints []string
	// AdminKey is the key used for generating storage credentials
	AdminKey string
	// AdminToken is the token used by the duros-controller to authenticate against the duros API
	AdminToken string
	// StorageClasses contain information on the storage classes that the duros-controller creates in the shoot cluster
	StorageClasses []DurosSeedStorageClass
}

type DurosSeedStorageClass struct {
	// Name is the name of the storage class
	Name string
	// ReplicaCount is the amount of replicas in the storage backend for this storage class
	ReplicaCount int
	// Compression enables compression for this storage class
	Compression bool
}

// ImagePullSecret provides an opportunity to inject an image pull secret into the resource deployments
type ImagePullSecret struct {
	// DockerConfigJSON contains the already base64 encoded JSON content for the image pull secret
	DockerConfigJSON string
}
