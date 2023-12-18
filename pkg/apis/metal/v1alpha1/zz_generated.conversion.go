//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
2023 Copyright metal-stack Authors.
*/

// Code generated by conversion-gen. DO NOT EDIT.

package v1alpha1

import (
	unsafe "unsafe"

	metal "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/metal"
	conversion "k8s.io/apimachinery/pkg/conversion"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func init() {
	localSchemeBuilder.Register(RegisterConversions)
}

// RegisterConversions adds conversion functions to the given scheme.
// Public to allow building arbitrary schemes.
func RegisterConversions(s *runtime.Scheme) error {
	if err := s.AddGeneratedConversionFunc((*CloudControllerManagerConfig)(nil), (*metal.CloudControllerManagerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CloudControllerManagerConfig_To_metal_CloudControllerManagerConfig(a.(*CloudControllerManagerConfig), b.(*metal.CloudControllerManagerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.CloudControllerManagerConfig)(nil), (*CloudControllerManagerConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(a.(*metal.CloudControllerManagerConfig), b.(*CloudControllerManagerConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*CloudProfileConfig)(nil), (*metal.CloudProfileConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CloudProfileConfig_To_metal_CloudProfileConfig(a.(*CloudProfileConfig), b.(*metal.CloudProfileConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.CloudProfileConfig)(nil), (*CloudProfileConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(a.(*metal.CloudProfileConfig), b.(*CloudProfileConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ControlPlaneConfig)(nil), (*metal.ControlPlaneConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ControlPlaneConfig_To_metal_ControlPlaneConfig(a.(*ControlPlaneConfig), b.(*metal.ControlPlaneConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.ControlPlaneConfig)(nil), (*ControlPlaneConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(a.(*metal.ControlPlaneConfig), b.(*ControlPlaneConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*ControlPlaneFeatures)(nil), (*metal.ControlPlaneFeatures)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures(a.(*ControlPlaneFeatures), b.(*metal.ControlPlaneFeatures), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.ControlPlaneFeatures)(nil), (*ControlPlaneFeatures)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures(a.(*metal.ControlPlaneFeatures), b.(*ControlPlaneFeatures), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*CustomDefaultStorageClass)(nil), (*metal.CustomDefaultStorageClass)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_CustomDefaultStorageClass_To_metal_CustomDefaultStorageClass(a.(*CustomDefaultStorageClass), b.(*metal.CustomDefaultStorageClass), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.CustomDefaultStorageClass)(nil), (*CustomDefaultStorageClass)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_CustomDefaultStorageClass_To_v1alpha1_CustomDefaultStorageClass(a.(*metal.CustomDefaultStorageClass), b.(*CustomDefaultStorageClass), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*EgressRule)(nil), (*metal.EgressRule)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_EgressRule_To_metal_EgressRule(a.(*EgressRule), b.(*metal.EgressRule), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.EgressRule)(nil), (*EgressRule)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_EgressRule_To_v1alpha1_EgressRule(a.(*metal.EgressRule), b.(*EgressRule), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Firewall)(nil), (*metal.Firewall)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Firewall_To_metal_Firewall(a.(*Firewall), b.(*metal.Firewall), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.Firewall)(nil), (*Firewall)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_Firewall_To_v1alpha1_Firewall(a.(*metal.Firewall), b.(*Firewall), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*FirewallControllerVersion)(nil), (*metal.FirewallControllerVersion)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_FirewallControllerVersion_To_metal_FirewallControllerVersion(a.(*FirewallControllerVersion), b.(*metal.FirewallControllerVersion), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.FirewallControllerVersion)(nil), (*FirewallControllerVersion)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_FirewallControllerVersion_To_v1alpha1_FirewallControllerVersion(a.(*metal.FirewallControllerVersion), b.(*FirewallControllerVersion), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*FirewallStatus)(nil), (*metal.FirewallStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_FirewallStatus_To_metal_FirewallStatus(a.(*FirewallStatus), b.(*metal.FirewallStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.FirewallStatus)(nil), (*FirewallStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_FirewallStatus_To_v1alpha1_FirewallStatus(a.(*metal.FirewallStatus), b.(*FirewallStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*InfrastructureConfig)(nil), (*metal.InfrastructureConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_InfrastructureConfig_To_metal_InfrastructureConfig(a.(*InfrastructureConfig), b.(*metal.InfrastructureConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.InfrastructureConfig)(nil), (*InfrastructureConfig)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(a.(*metal.InfrastructureConfig), b.(*InfrastructureConfig), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*InfrastructureStatus)(nil), (*metal.InfrastructureStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_InfrastructureStatus_To_metal_InfrastructureStatus(a.(*InfrastructureStatus), b.(*metal.InfrastructureStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.InfrastructureStatus)(nil), (*InfrastructureStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(a.(*metal.InfrastructureStatus), b.(*InfrastructureStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MachineImage)(nil), (*metal.MachineImage)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MachineImage_To_metal_MachineImage(a.(*MachineImage), b.(*metal.MachineImage), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.MachineImage)(nil), (*MachineImage)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_MachineImage_To_v1alpha1_MachineImage(a.(*metal.MachineImage), b.(*MachineImage), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*MetalControlPlane)(nil), (*metal.MetalControlPlane)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_MetalControlPlane_To_metal_MetalControlPlane(a.(*MetalControlPlane), b.(*metal.MetalControlPlane), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.MetalControlPlane)(nil), (*MetalControlPlane)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_MetalControlPlane_To_v1alpha1_MetalControlPlane(a.(*metal.MetalControlPlane), b.(*MetalControlPlane), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NetworkIsolation)(nil), (*metal.NetworkIsolation)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NetworkIsolation_To_metal_NetworkIsolation(a.(*NetworkIsolation), b.(*metal.NetworkIsolation), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.NetworkIsolation)(nil), (*NetworkIsolation)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_NetworkIsolation_To_v1alpha1_NetworkIsolation(a.(*metal.NetworkIsolation), b.(*NetworkIsolation), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NetworkServer)(nil), (*metal.NetworkServer)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NetworkServer_To_metal_NetworkServer(a.(*NetworkServer), b.(*metal.NetworkServer), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.NetworkServer)(nil), (*NetworkServer)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_NetworkServer_To_v1alpha1_NetworkServer(a.(*metal.NetworkServer), b.(*NetworkServer), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*NftablesExporter)(nil), (*metal.NftablesExporter)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_NftablesExporter_To_metal_NftablesExporter(a.(*NftablesExporter), b.(*metal.NftablesExporter), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.NftablesExporter)(nil), (*NftablesExporter)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_NftablesExporter_To_v1alpha1_NftablesExporter(a.(*metal.NftablesExporter), b.(*NftablesExporter), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*Partition)(nil), (*metal.Partition)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_Partition_To_metal_Partition(a.(*Partition), b.(*metal.Partition), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.Partition)(nil), (*Partition)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_Partition_To_v1alpha1_Partition(a.(*metal.Partition), b.(*Partition), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*RateLimit)(nil), (*metal.RateLimit)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_RateLimit_To_metal_RateLimit(a.(*RateLimit), b.(*metal.RateLimit), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.RateLimit)(nil), (*RateLimit)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_RateLimit_To_v1alpha1_RateLimit(a.(*metal.RateLimit), b.(*RateLimit), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*WorkerStatus)(nil), (*metal.WorkerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_v1alpha1_WorkerStatus_To_metal_WorkerStatus(a.(*WorkerStatus), b.(*metal.WorkerStatus), scope)
	}); err != nil {
		return err
	}
	if err := s.AddGeneratedConversionFunc((*metal.WorkerStatus)(nil), (*WorkerStatus)(nil), func(a, b interface{}, scope conversion.Scope) error {
		return Convert_metal_WorkerStatus_To_v1alpha1_WorkerStatus(a.(*metal.WorkerStatus), b.(*WorkerStatus), scope)
	}); err != nil {
		return err
	}
	return nil
}

func autoConvert_v1alpha1_CloudControllerManagerConfig_To_metal_CloudControllerManagerConfig(in *CloudControllerManagerConfig, out *metal.CloudControllerManagerConfig, s conversion.Scope) error {
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	out.DefaultExternalNetwork = (*string)(unsafe.Pointer(in.DefaultExternalNetwork))
	return nil
}

// Convert_v1alpha1_CloudControllerManagerConfig_To_metal_CloudControllerManagerConfig is an autogenerated conversion function.
func Convert_v1alpha1_CloudControllerManagerConfig_To_metal_CloudControllerManagerConfig(in *CloudControllerManagerConfig, out *metal.CloudControllerManagerConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_CloudControllerManagerConfig_To_metal_CloudControllerManagerConfig(in, out, s)
}

func autoConvert_metal_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in *metal.CloudControllerManagerConfig, out *CloudControllerManagerConfig, s conversion.Scope) error {
	out.FeatureGates = *(*map[string]bool)(unsafe.Pointer(&in.FeatureGates))
	out.DefaultExternalNetwork = (*string)(unsafe.Pointer(in.DefaultExternalNetwork))
	return nil
}

// Convert_metal_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig is an autogenerated conversion function.
func Convert_metal_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in *metal.CloudControllerManagerConfig, out *CloudControllerManagerConfig, s conversion.Scope) error {
	return autoConvert_metal_CloudControllerManagerConfig_To_v1alpha1_CloudControllerManagerConfig(in, out, s)
}

func autoConvert_v1alpha1_CloudProfileConfig_To_metal_CloudProfileConfig(in *CloudProfileConfig, out *metal.CloudProfileConfig, s conversion.Scope) error {
	out.MetalControlPlanes = *(*map[string]metal.MetalControlPlane)(unsafe.Pointer(&in.MetalControlPlanes))
	return nil
}

// Convert_v1alpha1_CloudProfileConfig_To_metal_CloudProfileConfig is an autogenerated conversion function.
func Convert_v1alpha1_CloudProfileConfig_To_metal_CloudProfileConfig(in *CloudProfileConfig, out *metal.CloudProfileConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_CloudProfileConfig_To_metal_CloudProfileConfig(in, out, s)
}

func autoConvert_metal_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in *metal.CloudProfileConfig, out *CloudProfileConfig, s conversion.Scope) error {
	out.MetalControlPlanes = *(*map[string]MetalControlPlane)(unsafe.Pointer(&in.MetalControlPlanes))
	return nil
}

// Convert_metal_CloudProfileConfig_To_v1alpha1_CloudProfileConfig is an autogenerated conversion function.
func Convert_metal_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in *metal.CloudProfileConfig, out *CloudProfileConfig, s conversion.Scope) error {
	return autoConvert_metal_CloudProfileConfig_To_v1alpha1_CloudProfileConfig(in, out, s)
}

func autoConvert_v1alpha1_ControlPlaneConfig_To_metal_ControlPlaneConfig(in *ControlPlaneConfig, out *metal.ControlPlaneConfig, s conversion.Scope) error {
	out.CloudControllerManager = (*metal.CloudControllerManagerConfig)(unsafe.Pointer(in.CloudControllerManager))
	if err := Convert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures(&in.FeatureGates, &out.FeatureGates, s); err != nil {
		return err
	}
	out.CustomDefaultStorageClass = (*metal.CustomDefaultStorageClass)(unsafe.Pointer(in.CustomDefaultStorageClass))
	out.NetworkAccessType = (*metal.NetworkAccessType)(unsafe.Pointer(in.NetworkAccessType))
	return nil
}

// Convert_v1alpha1_ControlPlaneConfig_To_metal_ControlPlaneConfig is an autogenerated conversion function.
func Convert_v1alpha1_ControlPlaneConfig_To_metal_ControlPlaneConfig(in *ControlPlaneConfig, out *metal.ControlPlaneConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_ControlPlaneConfig_To_metal_ControlPlaneConfig(in, out, s)
}

func autoConvert_metal_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in *metal.ControlPlaneConfig, out *ControlPlaneConfig, s conversion.Scope) error {
	out.CloudControllerManager = (*CloudControllerManagerConfig)(unsafe.Pointer(in.CloudControllerManager))
	if err := Convert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures(&in.FeatureGates, &out.FeatureGates, s); err != nil {
		return err
	}
	out.CustomDefaultStorageClass = (*CustomDefaultStorageClass)(unsafe.Pointer(in.CustomDefaultStorageClass))
	out.NetworkAccessType = (*NetworkAccessType)(unsafe.Pointer(in.NetworkAccessType))
	return nil
}

// Convert_metal_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig is an autogenerated conversion function.
func Convert_metal_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in *metal.ControlPlaneConfig, out *ControlPlaneConfig, s conversion.Scope) error {
	return autoConvert_metal_ControlPlaneConfig_To_v1alpha1_ControlPlaneConfig(in, out, s)
}

func autoConvert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures(in *ControlPlaneFeatures, out *metal.ControlPlaneFeatures, s conversion.Scope) error {
	out.MachineControllerManagerOOT = (*bool)(unsafe.Pointer(in.MachineControllerManagerOOT))
	out.ClusterAudit = (*bool)(unsafe.Pointer(in.ClusterAudit))
	out.AuditToSplunk = (*bool)(unsafe.Pointer(in.AuditToSplunk))
	out.DurosStorageEncryption = (*bool)(unsafe.Pointer(in.DurosStorageEncryption))
	out.RestrictEgress = (*bool)(unsafe.Pointer(in.RestrictEgress))
	return nil
}

// Convert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures is an autogenerated conversion function.
func Convert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures(in *ControlPlaneFeatures, out *metal.ControlPlaneFeatures, s conversion.Scope) error {
	return autoConvert_v1alpha1_ControlPlaneFeatures_To_metal_ControlPlaneFeatures(in, out, s)
}

func autoConvert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures(in *metal.ControlPlaneFeatures, out *ControlPlaneFeatures, s conversion.Scope) error {
	out.MachineControllerManagerOOT = (*bool)(unsafe.Pointer(in.MachineControllerManagerOOT))
	out.ClusterAudit = (*bool)(unsafe.Pointer(in.ClusterAudit))
	out.AuditToSplunk = (*bool)(unsafe.Pointer(in.AuditToSplunk))
	out.DurosStorageEncryption = (*bool)(unsafe.Pointer(in.DurosStorageEncryption))
	out.RestrictEgress = (*bool)(unsafe.Pointer(in.RestrictEgress))
	return nil
}

// Convert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures is an autogenerated conversion function.
func Convert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures(in *metal.ControlPlaneFeatures, out *ControlPlaneFeatures, s conversion.Scope) error {
	return autoConvert_metal_ControlPlaneFeatures_To_v1alpha1_ControlPlaneFeatures(in, out, s)
}

func autoConvert_v1alpha1_CustomDefaultStorageClass_To_metal_CustomDefaultStorageClass(in *CustomDefaultStorageClass, out *metal.CustomDefaultStorageClass, s conversion.Scope) error {
	out.ClassName = in.ClassName
	return nil
}

// Convert_v1alpha1_CustomDefaultStorageClass_To_metal_CustomDefaultStorageClass is an autogenerated conversion function.
func Convert_v1alpha1_CustomDefaultStorageClass_To_metal_CustomDefaultStorageClass(in *CustomDefaultStorageClass, out *metal.CustomDefaultStorageClass, s conversion.Scope) error {
	return autoConvert_v1alpha1_CustomDefaultStorageClass_To_metal_CustomDefaultStorageClass(in, out, s)
}

func autoConvert_metal_CustomDefaultStorageClass_To_v1alpha1_CustomDefaultStorageClass(in *metal.CustomDefaultStorageClass, out *CustomDefaultStorageClass, s conversion.Scope) error {
	out.ClassName = in.ClassName
	return nil
}

// Convert_metal_CustomDefaultStorageClass_To_v1alpha1_CustomDefaultStorageClass is an autogenerated conversion function.
func Convert_metal_CustomDefaultStorageClass_To_v1alpha1_CustomDefaultStorageClass(in *metal.CustomDefaultStorageClass, out *CustomDefaultStorageClass, s conversion.Scope) error {
	return autoConvert_metal_CustomDefaultStorageClass_To_v1alpha1_CustomDefaultStorageClass(in, out, s)
}

func autoConvert_v1alpha1_EgressRule_To_metal_EgressRule(in *EgressRule, out *metal.EgressRule, s conversion.Scope) error {
	out.NetworkID = in.NetworkID
	out.IPs = *(*[]string)(unsafe.Pointer(&in.IPs))
	return nil
}

// Convert_v1alpha1_EgressRule_To_metal_EgressRule is an autogenerated conversion function.
func Convert_v1alpha1_EgressRule_To_metal_EgressRule(in *EgressRule, out *metal.EgressRule, s conversion.Scope) error {
	return autoConvert_v1alpha1_EgressRule_To_metal_EgressRule(in, out, s)
}

func autoConvert_metal_EgressRule_To_v1alpha1_EgressRule(in *metal.EgressRule, out *EgressRule, s conversion.Scope) error {
	out.NetworkID = in.NetworkID
	out.IPs = *(*[]string)(unsafe.Pointer(&in.IPs))
	return nil
}

// Convert_metal_EgressRule_To_v1alpha1_EgressRule is an autogenerated conversion function.
func Convert_metal_EgressRule_To_v1alpha1_EgressRule(in *metal.EgressRule, out *EgressRule, s conversion.Scope) error {
	return autoConvert_metal_EgressRule_To_v1alpha1_EgressRule(in, out, s)
}

func autoConvert_v1alpha1_Firewall_To_metal_Firewall(in *Firewall, out *metal.Firewall, s conversion.Scope) error {
	out.Size = in.Size
	out.Image = in.Image
	out.Networks = *(*[]string)(unsafe.Pointer(&in.Networks))
	out.RateLimits = *(*[]metal.RateLimit)(unsafe.Pointer(&in.RateLimits))
	out.EgressRules = *(*[]metal.EgressRule)(unsafe.Pointer(&in.EgressRules))
	out.LogAcceptedConnections = in.LogAcceptedConnections
	out.ControllerVersion = in.ControllerVersion
	return nil
}

// Convert_v1alpha1_Firewall_To_metal_Firewall is an autogenerated conversion function.
func Convert_v1alpha1_Firewall_To_metal_Firewall(in *Firewall, out *metal.Firewall, s conversion.Scope) error {
	return autoConvert_v1alpha1_Firewall_To_metal_Firewall(in, out, s)
}

func autoConvert_metal_Firewall_To_v1alpha1_Firewall(in *metal.Firewall, out *Firewall, s conversion.Scope) error {
	out.Size = in.Size
	out.Image = in.Image
	out.Networks = *(*[]string)(unsafe.Pointer(&in.Networks))
	out.RateLimits = *(*[]RateLimit)(unsafe.Pointer(&in.RateLimits))
	out.EgressRules = *(*[]EgressRule)(unsafe.Pointer(&in.EgressRules))
	out.LogAcceptedConnections = in.LogAcceptedConnections
	out.ControllerVersion = in.ControllerVersion
	return nil
}

// Convert_metal_Firewall_To_v1alpha1_Firewall is an autogenerated conversion function.
func Convert_metal_Firewall_To_v1alpha1_Firewall(in *metal.Firewall, out *Firewall, s conversion.Scope) error {
	return autoConvert_metal_Firewall_To_v1alpha1_Firewall(in, out, s)
}

func autoConvert_v1alpha1_FirewallControllerVersion_To_metal_FirewallControllerVersion(in *FirewallControllerVersion, out *metal.FirewallControllerVersion, s conversion.Scope) error {
	out.Version = in.Version
	out.URL = in.URL
	out.Classification = (*metal.VersionClassification)(unsafe.Pointer(in.Classification))
	return nil
}

// Convert_v1alpha1_FirewallControllerVersion_To_metal_FirewallControllerVersion is an autogenerated conversion function.
func Convert_v1alpha1_FirewallControllerVersion_To_metal_FirewallControllerVersion(in *FirewallControllerVersion, out *metal.FirewallControllerVersion, s conversion.Scope) error {
	return autoConvert_v1alpha1_FirewallControllerVersion_To_metal_FirewallControllerVersion(in, out, s)
}

func autoConvert_metal_FirewallControllerVersion_To_v1alpha1_FirewallControllerVersion(in *metal.FirewallControllerVersion, out *FirewallControllerVersion, s conversion.Scope) error {
	out.Version = in.Version
	out.URL = in.URL
	out.Classification = (*VersionClassification)(unsafe.Pointer(in.Classification))
	return nil
}

// Convert_metal_FirewallControllerVersion_To_v1alpha1_FirewallControllerVersion is an autogenerated conversion function.
func Convert_metal_FirewallControllerVersion_To_v1alpha1_FirewallControllerVersion(in *metal.FirewallControllerVersion, out *FirewallControllerVersion, s conversion.Scope) error {
	return autoConvert_metal_FirewallControllerVersion_To_v1alpha1_FirewallControllerVersion(in, out, s)
}

func autoConvert_v1alpha1_FirewallStatus_To_metal_FirewallStatus(in *FirewallStatus, out *metal.FirewallStatus, s conversion.Scope) error {
	out.MachineID = in.MachineID
	return nil
}

// Convert_v1alpha1_FirewallStatus_To_metal_FirewallStatus is an autogenerated conversion function.
func Convert_v1alpha1_FirewallStatus_To_metal_FirewallStatus(in *FirewallStatus, out *metal.FirewallStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_FirewallStatus_To_metal_FirewallStatus(in, out, s)
}

func autoConvert_metal_FirewallStatus_To_v1alpha1_FirewallStatus(in *metal.FirewallStatus, out *FirewallStatus, s conversion.Scope) error {
	out.MachineID = in.MachineID
	return nil
}

// Convert_metal_FirewallStatus_To_v1alpha1_FirewallStatus is an autogenerated conversion function.
func Convert_metal_FirewallStatus_To_v1alpha1_FirewallStatus(in *metal.FirewallStatus, out *FirewallStatus, s conversion.Scope) error {
	return autoConvert_metal_FirewallStatus_To_v1alpha1_FirewallStatus(in, out, s)
}

func autoConvert_v1alpha1_InfrastructureConfig_To_metal_InfrastructureConfig(in *InfrastructureConfig, out *metal.InfrastructureConfig, s conversion.Scope) error {
	if err := Convert_v1alpha1_Firewall_To_metal_Firewall(&in.Firewall, &out.Firewall, s); err != nil {
		return err
	}
	out.PartitionID = in.PartitionID
	out.ProjectID = in.ProjectID
	return nil
}

// Convert_v1alpha1_InfrastructureConfig_To_metal_InfrastructureConfig is an autogenerated conversion function.
func Convert_v1alpha1_InfrastructureConfig_To_metal_InfrastructureConfig(in *InfrastructureConfig, out *metal.InfrastructureConfig, s conversion.Scope) error {
	return autoConvert_v1alpha1_InfrastructureConfig_To_metal_InfrastructureConfig(in, out, s)
}

func autoConvert_metal_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in *metal.InfrastructureConfig, out *InfrastructureConfig, s conversion.Scope) error {
	if err := Convert_metal_Firewall_To_v1alpha1_Firewall(&in.Firewall, &out.Firewall, s); err != nil {
		return err
	}
	out.PartitionID = in.PartitionID
	out.ProjectID = in.ProjectID
	return nil
}

// Convert_metal_InfrastructureConfig_To_v1alpha1_InfrastructureConfig is an autogenerated conversion function.
func Convert_metal_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in *metal.InfrastructureConfig, out *InfrastructureConfig, s conversion.Scope) error {
	return autoConvert_metal_InfrastructureConfig_To_v1alpha1_InfrastructureConfig(in, out, s)
}

func autoConvert_v1alpha1_InfrastructureStatus_To_metal_InfrastructureStatus(in *InfrastructureStatus, out *metal.InfrastructureStatus, s conversion.Scope) error {
	if err := Convert_v1alpha1_FirewallStatus_To_metal_FirewallStatus(&in.Firewall, &out.Firewall, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_InfrastructureStatus_To_metal_InfrastructureStatus is an autogenerated conversion function.
func Convert_v1alpha1_InfrastructureStatus_To_metal_InfrastructureStatus(in *InfrastructureStatus, out *metal.InfrastructureStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_InfrastructureStatus_To_metal_InfrastructureStatus(in, out, s)
}

func autoConvert_metal_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in *metal.InfrastructureStatus, out *InfrastructureStatus, s conversion.Scope) error {
	if err := Convert_metal_FirewallStatus_To_v1alpha1_FirewallStatus(&in.Firewall, &out.Firewall, s); err != nil {
		return err
	}
	return nil
}

// Convert_metal_InfrastructureStatus_To_v1alpha1_InfrastructureStatus is an autogenerated conversion function.
func Convert_metal_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in *metal.InfrastructureStatus, out *InfrastructureStatus, s conversion.Scope) error {
	return autoConvert_metal_InfrastructureStatus_To_v1alpha1_InfrastructureStatus(in, out, s)
}

func autoConvert_v1alpha1_MachineImage_To_metal_MachineImage(in *MachineImage, out *metal.MachineImage, s conversion.Scope) error {
	out.Name = in.Name
	out.Version = in.Version
	out.Image = in.Image
	return nil
}

// Convert_v1alpha1_MachineImage_To_metal_MachineImage is an autogenerated conversion function.
func Convert_v1alpha1_MachineImage_To_metal_MachineImage(in *MachineImage, out *metal.MachineImage, s conversion.Scope) error {
	return autoConvert_v1alpha1_MachineImage_To_metal_MachineImage(in, out, s)
}

func autoConvert_metal_MachineImage_To_v1alpha1_MachineImage(in *metal.MachineImage, out *MachineImage, s conversion.Scope) error {
	out.Name = in.Name
	out.Version = in.Version
	out.Image = in.Image
	return nil
}

// Convert_metal_MachineImage_To_v1alpha1_MachineImage is an autogenerated conversion function.
func Convert_metal_MachineImage_To_v1alpha1_MachineImage(in *metal.MachineImage, out *MachineImage, s conversion.Scope) error {
	return autoConvert_metal_MachineImage_To_v1alpha1_MachineImage(in, out, s)
}

func autoConvert_v1alpha1_MetalControlPlane_To_metal_MetalControlPlane(in *MetalControlPlane, out *metal.MetalControlPlane, s conversion.Scope) error {
	out.Endpoint = in.Endpoint
	out.Partitions = *(*map[string]metal.Partition)(unsafe.Pointer(&in.Partitions))
	out.FirewallImages = *(*[]string)(unsafe.Pointer(&in.FirewallImages))
	out.FirewallControllerVersions = *(*[]metal.FirewallControllerVersion)(unsafe.Pointer(&in.FirewallControllerVersions))
	if err := Convert_v1alpha1_NftablesExporter_To_metal_NftablesExporter(&in.NftablesExporter, &out.NftablesExporter, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_MetalControlPlane_To_metal_MetalControlPlane is an autogenerated conversion function.
func Convert_v1alpha1_MetalControlPlane_To_metal_MetalControlPlane(in *MetalControlPlane, out *metal.MetalControlPlane, s conversion.Scope) error {
	return autoConvert_v1alpha1_MetalControlPlane_To_metal_MetalControlPlane(in, out, s)
}

func autoConvert_metal_MetalControlPlane_To_v1alpha1_MetalControlPlane(in *metal.MetalControlPlane, out *MetalControlPlane, s conversion.Scope) error {
	out.Endpoint = in.Endpoint
	out.Partitions = *(*map[string]Partition)(unsafe.Pointer(&in.Partitions))
	out.FirewallImages = *(*[]string)(unsafe.Pointer(&in.FirewallImages))
	out.FirewallControllerVersions = *(*[]FirewallControllerVersion)(unsafe.Pointer(&in.FirewallControllerVersions))
	if err := Convert_metal_NftablesExporter_To_v1alpha1_NftablesExporter(&in.NftablesExporter, &out.NftablesExporter, s); err != nil {
		return err
	}
	return nil
}

// Convert_metal_MetalControlPlane_To_v1alpha1_MetalControlPlane is an autogenerated conversion function.
func Convert_metal_MetalControlPlane_To_v1alpha1_MetalControlPlane(in *metal.MetalControlPlane, out *MetalControlPlane, s conversion.Scope) error {
	return autoConvert_metal_MetalControlPlane_To_v1alpha1_MetalControlPlane(in, out, s)
}

func autoConvert_v1alpha1_NetworkIsolation_To_metal_NetworkIsolation(in *NetworkIsolation, out *metal.NetworkIsolation, s conversion.Scope) error {
	out.AllowedNetworks = *(*[]string)(unsafe.Pointer(&in.AllowedNetworks))
	out.DNSServers = *(*[]string)(unsafe.Pointer(&in.DNSServers))
	out.NTPServers = *(*[]string)(unsafe.Pointer(&in.NTPServers))
	if err := Convert_v1alpha1_NetworkServer_To_metal_NetworkServer(&in.Registry, &out.Registry, s); err != nil {
		return err
	}
	return nil
}

// Convert_v1alpha1_NetworkIsolation_To_metal_NetworkIsolation is an autogenerated conversion function.
func Convert_v1alpha1_NetworkIsolation_To_metal_NetworkIsolation(in *NetworkIsolation, out *metal.NetworkIsolation, s conversion.Scope) error {
	return autoConvert_v1alpha1_NetworkIsolation_To_metal_NetworkIsolation(in, out, s)
}

func autoConvert_metal_NetworkIsolation_To_v1alpha1_NetworkIsolation(in *metal.NetworkIsolation, out *NetworkIsolation, s conversion.Scope) error {
	out.AllowedNetworks = *(*[]string)(unsafe.Pointer(&in.AllowedNetworks))
	out.DNSServers = *(*[]string)(unsafe.Pointer(&in.DNSServers))
	out.NTPServers = *(*[]string)(unsafe.Pointer(&in.NTPServers))
	if err := Convert_metal_NetworkServer_To_v1alpha1_NetworkServer(&in.Registry, &out.Registry, s); err != nil {
		return err
	}
	return nil
}

// Convert_metal_NetworkIsolation_To_v1alpha1_NetworkIsolation is an autogenerated conversion function.
func Convert_metal_NetworkIsolation_To_v1alpha1_NetworkIsolation(in *metal.NetworkIsolation, out *NetworkIsolation, s conversion.Scope) error {
	return autoConvert_metal_NetworkIsolation_To_v1alpha1_NetworkIsolation(in, out, s)
}

func autoConvert_v1alpha1_NetworkServer_To_metal_NetworkServer(in *NetworkServer, out *metal.NetworkServer, s conversion.Scope) error {
	out.Name = in.Name
	out.Hostname = in.Hostname
	out.IP = in.IP
	out.Port = in.Port
	return nil
}

// Convert_v1alpha1_NetworkServer_To_metal_NetworkServer is an autogenerated conversion function.
func Convert_v1alpha1_NetworkServer_To_metal_NetworkServer(in *NetworkServer, out *metal.NetworkServer, s conversion.Scope) error {
	return autoConvert_v1alpha1_NetworkServer_To_metal_NetworkServer(in, out, s)
}

func autoConvert_metal_NetworkServer_To_v1alpha1_NetworkServer(in *metal.NetworkServer, out *NetworkServer, s conversion.Scope) error {
	out.Name = in.Name
	out.Hostname = in.Hostname
	out.IP = in.IP
	out.Port = in.Port
	return nil
}

// Convert_metal_NetworkServer_To_v1alpha1_NetworkServer is an autogenerated conversion function.
func Convert_metal_NetworkServer_To_v1alpha1_NetworkServer(in *metal.NetworkServer, out *NetworkServer, s conversion.Scope) error {
	return autoConvert_metal_NetworkServer_To_v1alpha1_NetworkServer(in, out, s)
}

func autoConvert_v1alpha1_NftablesExporter_To_metal_NftablesExporter(in *NftablesExporter, out *metal.NftablesExporter, s conversion.Scope) error {
	out.Version = in.Version
	out.URL = in.URL
	return nil
}

// Convert_v1alpha1_NftablesExporter_To_metal_NftablesExporter is an autogenerated conversion function.
func Convert_v1alpha1_NftablesExporter_To_metal_NftablesExporter(in *NftablesExporter, out *metal.NftablesExporter, s conversion.Scope) error {
	return autoConvert_v1alpha1_NftablesExporter_To_metal_NftablesExporter(in, out, s)
}

func autoConvert_metal_NftablesExporter_To_v1alpha1_NftablesExporter(in *metal.NftablesExporter, out *NftablesExporter, s conversion.Scope) error {
	out.Version = in.Version
	out.URL = in.URL
	return nil
}

// Convert_metal_NftablesExporter_To_v1alpha1_NftablesExporter is an autogenerated conversion function.
func Convert_metal_NftablesExporter_To_v1alpha1_NftablesExporter(in *metal.NftablesExporter, out *NftablesExporter, s conversion.Scope) error {
	return autoConvert_metal_NftablesExporter_To_v1alpha1_NftablesExporter(in, out, s)
}

func autoConvert_v1alpha1_Partition_To_metal_Partition(in *Partition, out *metal.Partition, s conversion.Scope) error {
	out.FirewallTypes = *(*[]string)(unsafe.Pointer(&in.FirewallTypes))
	out.NetworkIsolation = (*metal.NetworkIsolation)(unsafe.Pointer(in.NetworkIsolation))
	return nil
}

// Convert_v1alpha1_Partition_To_metal_Partition is an autogenerated conversion function.
func Convert_v1alpha1_Partition_To_metal_Partition(in *Partition, out *metal.Partition, s conversion.Scope) error {
	return autoConvert_v1alpha1_Partition_To_metal_Partition(in, out, s)
}

func autoConvert_metal_Partition_To_v1alpha1_Partition(in *metal.Partition, out *Partition, s conversion.Scope) error {
	out.FirewallTypes = *(*[]string)(unsafe.Pointer(&in.FirewallTypes))
	out.NetworkIsolation = (*NetworkIsolation)(unsafe.Pointer(in.NetworkIsolation))
	return nil
}

// Convert_metal_Partition_To_v1alpha1_Partition is an autogenerated conversion function.
func Convert_metal_Partition_To_v1alpha1_Partition(in *metal.Partition, out *Partition, s conversion.Scope) error {
	return autoConvert_metal_Partition_To_v1alpha1_Partition(in, out, s)
}

func autoConvert_v1alpha1_RateLimit_To_metal_RateLimit(in *RateLimit, out *metal.RateLimit, s conversion.Scope) error {
	out.NetworkID = in.NetworkID
	out.RateLimit = in.RateLimit
	return nil
}

// Convert_v1alpha1_RateLimit_To_metal_RateLimit is an autogenerated conversion function.
func Convert_v1alpha1_RateLimit_To_metal_RateLimit(in *RateLimit, out *metal.RateLimit, s conversion.Scope) error {
	return autoConvert_v1alpha1_RateLimit_To_metal_RateLimit(in, out, s)
}

func autoConvert_metal_RateLimit_To_v1alpha1_RateLimit(in *metal.RateLimit, out *RateLimit, s conversion.Scope) error {
	out.NetworkID = in.NetworkID
	out.RateLimit = in.RateLimit
	return nil
}

// Convert_metal_RateLimit_To_v1alpha1_RateLimit is an autogenerated conversion function.
func Convert_metal_RateLimit_To_v1alpha1_RateLimit(in *metal.RateLimit, out *RateLimit, s conversion.Scope) error {
	return autoConvert_metal_RateLimit_To_v1alpha1_RateLimit(in, out, s)
}

func autoConvert_v1alpha1_WorkerStatus_To_metal_WorkerStatus(in *WorkerStatus, out *metal.WorkerStatus, s conversion.Scope) error {
	out.MachineImages = *(*[]metal.MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_v1alpha1_WorkerStatus_To_metal_WorkerStatus is an autogenerated conversion function.
func Convert_v1alpha1_WorkerStatus_To_metal_WorkerStatus(in *WorkerStatus, out *metal.WorkerStatus, s conversion.Scope) error {
	return autoConvert_v1alpha1_WorkerStatus_To_metal_WorkerStatus(in, out, s)
}

func autoConvert_metal_WorkerStatus_To_v1alpha1_WorkerStatus(in *metal.WorkerStatus, out *WorkerStatus, s conversion.Scope) error {
	out.MachineImages = *(*[]MachineImage)(unsafe.Pointer(&in.MachineImages))
	return nil
}

// Convert_metal_WorkerStatus_To_v1alpha1_WorkerStatus is an autogenerated conversion function.
func Convert_metal_WorkerStatus_To_v1alpha1_WorkerStatus(in *metal.WorkerStatus, out *WorkerStatus, s conversion.Scope) error {
	return autoConvert_metal_WorkerStatus_To_v1alpha1_WorkerStatus(in, out, s)
}
