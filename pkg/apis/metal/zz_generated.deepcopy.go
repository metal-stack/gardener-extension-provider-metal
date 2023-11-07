//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
2023 Copyright metal-stack Authors.
*/

// Code generated by deepcopy-gen. DO NOT EDIT.

package metal

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudControllerManagerConfig) DeepCopyInto(out *CloudControllerManagerConfig) {
	*out = *in
	if in.FeatureGates != nil {
		in, out := &in.FeatureGates, &out.FeatureGates
		*out = make(map[string]bool, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.DefaultExternalNetwork != nil {
		in, out := &in.DefaultExternalNetwork, &out.DefaultExternalNetwork
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudControllerManagerConfig.
func (in *CloudControllerManagerConfig) DeepCopy() *CloudControllerManagerConfig {
	if in == nil {
		return nil
	}
	out := new(CloudControllerManagerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CloudProfileConfig) DeepCopyInto(out *CloudProfileConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.MetalControlPlanes != nil {
		in, out := &in.MetalControlPlanes, &out.MetalControlPlanes
		*out = make(map[string]MetalControlPlane, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CloudProfileConfig.
func (in *CloudProfileConfig) DeepCopy() *CloudProfileConfig {
	if in == nil {
		return nil
	}
	out := new(CloudProfileConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CloudProfileConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControlPlaneConfig) DeepCopyInto(out *ControlPlaneConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.CloudControllerManager != nil {
		in, out := &in.CloudControllerManager, &out.CloudControllerManager
		*out = new(CloudControllerManagerConfig)
		(*in).DeepCopyInto(*out)
	}
	in.FeatureGates.DeepCopyInto(&out.FeatureGates)
	if in.CustomDefaultStorageClass != nil {
		in, out := &in.CustomDefaultStorageClass, &out.CustomDefaultStorageClass
		*out = new(CustomDefaultStorageClass)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControlPlaneConfig.
func (in *ControlPlaneConfig) DeepCopy() *ControlPlaneConfig {
	if in == nil {
		return nil
	}
	out := new(ControlPlaneConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ControlPlaneConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ControlPlaneFeatures) DeepCopyInto(out *ControlPlaneFeatures) {
	*out = *in
	if in.MachineControllerManagerOOT != nil {
		in, out := &in.MachineControllerManagerOOT, &out.MachineControllerManagerOOT
		*out = new(bool)
		**out = **in
	}
	if in.ClusterAudit != nil {
		in, out := &in.ClusterAudit, &out.ClusterAudit
		*out = new(bool)
		**out = **in
	}
	if in.AuditToSplunk != nil {
		in, out := &in.AuditToSplunk, &out.AuditToSplunk
		*out = new(bool)
		**out = **in
	}
	if in.DurosStorageEncryption != nil {
		in, out := &in.DurosStorageEncryption, &out.DurosStorageEncryption
		*out = new(bool)
		**out = **in
	}
	if in.RestrictEgress != nil {
		in, out := &in.RestrictEgress, &out.RestrictEgress
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ControlPlaneFeatures.
func (in *ControlPlaneFeatures) DeepCopy() *ControlPlaneFeatures {
	if in == nil {
		return nil
	}
	out := new(ControlPlaneFeatures)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CustomDefaultStorageClass) DeepCopyInto(out *CustomDefaultStorageClass) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CustomDefaultStorageClass.
func (in *CustomDefaultStorageClass) DeepCopy() *CustomDefaultStorageClass {
	if in == nil {
		return nil
	}
	out := new(CustomDefaultStorageClass)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *EgressRule) DeepCopyInto(out *EgressRule) {
	*out = *in
	if in.IPs != nil {
		in, out := &in.IPs, &out.IPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new EgressRule.
func (in *EgressRule) DeepCopy() *EgressRule {
	if in == nil {
		return nil
	}
	out := new(EgressRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Firewall) DeepCopyInto(out *Firewall) {
	*out = *in
	if in.Networks != nil {
		in, out := &in.Networks, &out.Networks
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.RateLimits != nil {
		in, out := &in.RateLimits, &out.RateLimits
		*out = make([]RateLimit, len(*in))
		copy(*out, *in)
	}
	if in.EgressRules != nil {
		in, out := &in.EgressRules, &out.EgressRules
		*out = make([]EgressRule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Firewall.
func (in *Firewall) DeepCopy() *Firewall {
	if in == nil {
		return nil
	}
	out := new(Firewall)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FirewallControllerVersion) DeepCopyInto(out *FirewallControllerVersion) {
	*out = *in
	if in.Classification != nil {
		in, out := &in.Classification, &out.Classification
		*out = new(VersionClassification)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FirewallControllerVersion.
func (in *FirewallControllerVersion) DeepCopy() *FirewallControllerVersion {
	if in == nil {
		return nil
	}
	out := new(FirewallControllerVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *FirewallStatus) DeepCopyInto(out *FirewallStatus) {
	*out = *in
	if in.ExternalEgressIPs != nil {
		in, out := &in.ExternalEgressIPs, &out.ExternalEgressIPs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new FirewallStatus.
func (in *FirewallStatus) DeepCopy() *FirewallStatus {
	if in == nil {
		return nil
	}
	out := new(FirewallStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InfrastructureConfig) DeepCopyInto(out *InfrastructureConfig) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Firewall.DeepCopyInto(&out.Firewall)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InfrastructureConfig.
func (in *InfrastructureConfig) DeepCopy() *InfrastructureConfig {
	if in == nil {
		return nil
	}
	out := new(InfrastructureConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InfrastructureConfig) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InfrastructureStatus) DeepCopyInto(out *InfrastructureStatus) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.Firewall.DeepCopyInto(&out.Firewall)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InfrastructureStatus.
func (in *InfrastructureStatus) DeepCopy() *InfrastructureStatus {
	if in == nil {
		return nil
	}
	out := new(InfrastructureStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *InfrastructureStatus) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
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
func (in *MetalControlPlane) DeepCopyInto(out *MetalControlPlane) {
	*out = *in
	if in.Partitions != nil {
		in, out := &in.Partitions, &out.Partitions
		*out = make(map[string]Partition, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.FirewallImages != nil {
		in, out := &in.FirewallImages, &out.FirewallImages
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.FirewallControllerVersions != nil {
		in, out := &in.FirewallControllerVersions, &out.FirewallControllerVersions
		*out = make([]FirewallControllerVersion, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	out.NftablesExporter = in.NftablesExporter
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MetalControlPlane.
func (in *MetalControlPlane) DeepCopy() *MetalControlPlane {
	if in == nil {
		return nil
	}
	out := new(MetalControlPlane)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NftablesExporter) DeepCopyInto(out *NftablesExporter) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NftablesExporter.
func (in *NftablesExporter) DeepCopy() *NftablesExporter {
	if in == nil {
		return nil
	}
	out := new(NftablesExporter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Partition) DeepCopyInto(out *Partition) {
	*out = *in
	if in.FirewallTypes != nil {
		in, out := &in.FirewallTypes, &out.FirewallTypes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Partition.
func (in *Partition) DeepCopy() *Partition {
	if in == nil {
		return nil
	}
	out := new(Partition)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RateLimit) DeepCopyInto(out *RateLimit) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RateLimit.
func (in *RateLimit) DeepCopy() *RateLimit {
	if in == nil {
		return nil
	}
	out := new(RateLimit)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkerStatus) DeepCopyInto(out *WorkerStatus) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	if in.MachineImages != nil {
		in, out := &in.MachineImages, &out.MachineImages
		*out = make([]MachineImage, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkerStatus.
func (in *WorkerStatus) DeepCopy() *WorkerStatus {
	if in == nil {
		return nil
	}
	out := new(WorkerStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkerStatus) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}
