package metal

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta
	Firewall    Firewall
	PartitionID string
	ProjectID   string
}

type Firewall struct {
	Size              string
	Image             string
	Networks          []string
	RateLimits        []RateLimit
	EgressRules       []EgressRule
	ControllerVersion string
}

type RateLimit struct {
	NetworkID string
	RateLimit uint32
}

type EgressRule struct {
	NetworkID string
	IPs       []string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta
	Firewall FirewallStatus
}

type FirewallStatus struct {
	MachineID string
}
