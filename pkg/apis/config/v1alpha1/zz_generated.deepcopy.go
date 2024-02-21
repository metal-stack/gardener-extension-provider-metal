//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
2024 Copyright metal-stack Authors.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	apisconfigv1alpha1 "github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

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
	if in.ClientConnection != nil {
		in, out := &in.ClientConnection, &out.ClientConnection
		*out = new(configv1alpha1.ClientConnectionConfiguration)
		**out = **in
	}
	if in.AdditionalPodLabels != nil {
		in, out := &in.AdditionalPodLabels, &out.AdditionalPodLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.MachineImages != nil {
		in, out := &in.MachineImages, &out.MachineImages
		*out = make([]MachineImage, len(*in))
		copy(*out, *in)
	}
	if in.FirewallInternalPrefixes != nil {
		in, out := &in.FirewallInternalPrefixes, &out.FirewallInternalPrefixes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.ETCD.DeepCopyInto(&out.ETCD)
	out.ClusterAudit = in.ClusterAudit
	out.AuditToSplunk = in.AuditToSplunk
	if in.HealthCheckConfig != nil {
		in, out := &in.HealthCheckConfig, &out.HealthCheckConfig
		*out = new(apisconfigv1alpha1.HealthCheckConfig)
		(*in).DeepCopyInto(*out)
	}
	in.Storage.DeepCopyInto(&out.Storage)
	if in.ImagePullSecret != nil {
		in, out := &in.ImagePullSecret, &out.ImagePullSecret
		*out = new(ImagePullSecret)
		**out = **in
	}
	if in.EgressDestinations != nil {
		in, out := &in.EgressDestinations, &out.EgressDestinations
		*out = make([]EgressDest, len(*in))
		copy(*out, *in)
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
func (in *EgressDest) DeepCopyInto(out *EgressDest) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EgressDest.
func (in *EgressDest) DeepCopy() *EgressDest {
	if in == nil {
		return nil
	}
	out := new(EgressDest)
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
