package v1alpha1

import (
	"github.com/metal-stack/metal-go/api/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureConfig infrastructure configuration resource
type InfrastructureConfig struct {
	metav1.TypeMeta `json:",inline"`
	Firewall        Firewall `json:"firewall"`
	PartitionID     string   `json:"partitionID"`
	ProjectID       string   `json:"projectID"`
}

type Firewall struct {
	Size            string                     `json:"size"`
	Image           string                     `json:"image"`
	Networks        []string                   `json:"networks"`
	MachineNetworks []*models.V1MachineNetwork `json:"machinenetworks"`
	RateLimits      []RateLimit                `json:"ratelimits"`
	EgressRules     []EgressRule               `json:"egressrules"`
}

type RateLimit struct {
	NetworkID string `json:"networkid"`
	RateLimit uint32 `json:"ratelimit"`
}

type EgressRule struct {
	NetworkID string   `json:"networkid"`
	IPs       []string `json:"ips"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`
	Firewall        FirewallStatus `json:"firewall"`
}

type FirewallStatus struct {
	Succeeded bool   `json:"succeeded"`
	MachineID string `json:"machineID"`
}
