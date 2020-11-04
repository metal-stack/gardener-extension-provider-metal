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
	Size            string
	Image           string
	Networks        []string
	MachineNetworks []MachineNetwork
	RateLimits      []RateLimit
	EgressRules     []EgressRule
}

type MachineNetwork struct {
	Asn                 *int64   `json:"asn"`
	Destinationprefixes []string `json:"destinationprefixes"`
	Ips                 []string `json:"ips"`
	Nat                 *bool    `json:"nat"`
	Networkid           *string  `json:"networkid"`
	Networktype         *string  `json:"networktype"`
	Prefixes            []string `json:"prefixes"`
	Private             *bool    `json:"private"`
	Underlay            *bool    `json:"underlay"`
	Vrf                 *int64   `json:"vrf"`
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
	Succeeded bool
	MachineID string
}
