//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package config

import (
	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/controller/healthcheck/config"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AccountingExporterClientConfiguration) DeepCopyInto(out *AccountingExporterClientConfiguration) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AccountingExporterClientConfiguration.
func (in *AccountingExporterClientConfiguration) DeepCopy() *AccountingExporterClientConfiguration {
	if in == nil {
		return nil
	}
	out := new(AccountingExporterClientConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AccountingExporterConfiguration) DeepCopyInto(out *AccountingExporterConfiguration) {
	*out = *in
	in.NetworkTraffic.DeepCopyInto(&out.NetworkTraffic)
	out.Client = in.Client
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AccountingExporterConfiguration.
func (in *AccountingExporterConfiguration) DeepCopy() *AccountingExporterConfiguration {
	if in == nil {
		return nil
	}
	out := new(AccountingExporterConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AccountingExporterNetworkTrafficConfiguration) DeepCopyInto(out *AccountingExporterNetworkTrafficConfiguration) {
	*out = *in
	if in.InternalNetworks != nil {
		in, out := &in.InternalNetworks, &out.InternalNetworks
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AccountingExporterNetworkTrafficConfiguration.
func (in *AccountingExporterNetworkTrafficConfiguration) DeepCopy() *AccountingExporterNetworkTrafficConfiguration {
	if in == nil {
		return nil
	}
	out := new(AccountingExporterNetworkTrafficConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AuditToSplunk) DeepCopyInto(out *AuditToSplunk) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AuditToSplunk.
func (in *AuditToSplunk) DeepCopy() *AuditToSplunk {
	if in == nil {
		return nil
	}
	out := new(AuditToSplunk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Auth) DeepCopyInto(out *Auth) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Auth.
func (in *Auth) DeepCopy() *Auth {
	if in == nil {
		return nil
	}
	out := new(Auth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterAudit) DeepCopyInto(out *ClusterAudit) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterAudit.
func (in *ClusterAudit) DeepCopy() *ClusterAudit {
	if in == nil {
		return nil
	}
	out := new(ClusterAudit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControllerConfiguration) DeepCopyInto(out *ControllerConfiguration) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.MachineImages != nil {
		in, out := &in.MachineImages, &out.MachineImages
		*out = make([]MachineImage, len(*in))
		copy(*out, *in)
	}
	in.ETCD.DeepCopyInto(&out.ETCD)
	out.ClusterAudit = in.ClusterAudit
	out.AuditToSplunk = in.AuditToSplunk
	out.Auth = in.Auth
	in.AccountingExporter.DeepCopyInto(&out.AccountingExporter)
	if in.HealthCheckConfig != nil {
		in, out := &in.HealthCheckConfig, &out.HealthCheckConfig
		*out = new(healthcheckconfig.HealthCheckConfig)
		**out = **in
	}
	in.Storage.DeepCopyInto(&out.Storage)
	if in.ImagePullSecret != nil {
		in, out := &in.ImagePullSecret, &out.ImagePullSecret
		*out = new(ImagePullSecret)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControllerConfiguration.
func (in *ControllerConfiguration) DeepCopy() *ControllerConfiguration {
	if in == nil {
		return nil
	}
	out := new(ControllerConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControllerConfiguration) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DurosConfiguration) DeepCopyInto(out *DurosConfiguration) {
	*out = *in
	if in.PartitionConfig != nil {
		in, out := &in.PartitionConfig, &out.PartitionConfig
		*out = make(map[string]DurosPartitionConfiguration, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DurosConfiguration.
func (in *DurosConfiguration) DeepCopy() *DurosConfiguration {
	if in == nil {
		return nil
	}
	out := new(DurosConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DurosPartitionConfiguration) DeepCopyInto(out *DurosPartitionConfiguration) {
	*out = *in
	if in.Endpoints != nil {
		in, out := &in.Endpoints, &out.Endpoints
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.StorageClasses != nil {
		in, out := &in.StorageClasses, &out.StorageClasses
		*out = make([]DurosSeedStorageClass, len(*in))
		copy(*out, *in)
	}
	if in.APIEndpoint != nil {
		in, out := &in.APIEndpoint, &out.APIEndpoint
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DurosPartitionConfiguration.
func (in *DurosPartitionConfiguration) DeepCopy() *DurosPartitionConfiguration {
	if in == nil {
		return nil
	}
	out := new(DurosPartitionConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DurosSeedStorageClass) DeepCopyInto(out *DurosSeedStorageClass) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DurosSeedStorageClass.
func (in *DurosSeedStorageClass) DeepCopy() *DurosSeedStorageClass {
	if in == nil {
		return nil
	}
	out := new(DurosSeedStorageClass)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ETCD) DeepCopyInto(out *ETCD) {
	*out = *in
	in.Storage.DeepCopyInto(&out.Storage)
	in.Backup.DeepCopyInto(&out.Backup)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ETCD.
func (in *ETCD) DeepCopy() *ETCD {
	if in == nil {
		return nil
	}
	out := new(ETCD)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ETCDBackup) DeepCopyInto(out *ETCDBackup) {
	*out = *in
	if in.Schedule != nil {
		in, out := &in.Schedule, &out.Schedule
		*out = new(string)
		**out = **in
	}
	if in.DeltaSnapshotPeriod != nil {
		in, out := &in.DeltaSnapshotPeriod, &out.DeltaSnapshotPeriod
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ETCDBackup.
func (in *ETCDBackup) DeepCopy() *ETCDBackup {
	if in == nil {
		return nil
	}
	out := new(ETCDBackup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ETCDStorage) DeepCopyInto(out *ETCDStorage) {
	*out = *in
	if in.ClassName != nil {
		in, out := &in.ClassName, &out.ClassName
		*out = new(string)
		**out = **in
	}
	if in.Capacity != nil {
		in, out := &in.Capacity, &out.Capacity
		x := (*in).DeepCopy()
		*out = &x
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ETCDStorage.
func (in *ETCDStorage) DeepCopy() *ETCDStorage {
	if in == nil {
		return nil
	}
	out := new(ETCDStorage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImagePullSecret) DeepCopyInto(out *ImagePullSecret) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImagePullSecret.
func (in *ImagePullSecret) DeepCopy() *ImagePullSecret {
	if in == nil {
		return nil
	}
	out := new(ImagePullSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachineImage) DeepCopyInto(out *MachineImage) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachineImage.
func (in *MachineImage) DeepCopy() *MachineImage {
	if in == nil {
		return nil
	}
	out := new(MachineImage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageConfiguration) DeepCopyInto(out *StorageConfiguration) {
	*out = *in
	in.Duros.DeepCopyInto(&out.Duros)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageConfiguration.
func (in *StorageConfiguration) DeepCopy() *StorageConfiguration {
	if in == nil {
		return nil
	}
	out := new(StorageConfiguration)
	in.DeepCopyInto(out)
	return out
}
