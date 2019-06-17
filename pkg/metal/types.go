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

import "path/filepath"

const (
	// Name is the name of the AWS provider.
	Name = "provider-metal"
	// StorageProviderName is the name of the AWS storage provider.
	StorageProviderName = "S3"

	// MachineControllerManagerImageName is the name of the MachineControllerManager image.
	MachineControllerManagerImageName = "machine-controller-manager"
	// HyperkubeImageName is the name of the hyperkube image.
	HyperkubeImageName = "hyperkube"
	// ETCDBackupRestoreImageName is the name of the etcd backup and restore image.
	ETCDBackupRestoreImageName = "etcd-backup-restore"
	// AccessKeyID is a constant for the key in a cloud provider secret and backup secret that holds the AWS access key id.
	MetalAPIKey = "metalAPIKey"
	// SecretAccessKey is a constant for the key in a cloud provider secret and backup secret that holds the AWS secret access key.
	MetalAPIHMac = "metalAPIHMac"
	// Region is a constant for the key in a backup secret that holds the AWS region.
	Region = "region"
	// BucketName is a constant for the key in a backup secret that holds the bucket name.
	// The bucket name is written to the backup secret by Gardener as a temporary solution.
	// TODO In the future, the bucket name should come from a BackupBucket resource (see https://github.com/gardener/gardener/blob/master/docs/proposals/02-backupinfra.md)
	BucketName = "bucketName"
	// SSHKeyName key for accessing SSH key name from outputs in terraform
	SSHKeyName = "keyName"

	// MachineControllerManagerName is a constant for the name of the machine-controller-manager.
	MachineControllerManagerName = "machine-controller-manager"
	// BackupSecretName is the name of the secret containing the credentials for storing the backups of Shoot clusters.
	BackupSecretName = "etcd-backup"
)

var (
	// ChartsPath is the path to the charts
	ChartsPath = filepath.Join("controllers", Name, "charts")
	// InternalChartsPath is the path to the internal charts
	InternalChartsPath = filepath.Join(ChartsPath, "internal")
)

// Credentials stores AWS credentials.
type Credentials struct {
	MetalAPIKey  []byte
	MetalAPIHMac []byte
}
