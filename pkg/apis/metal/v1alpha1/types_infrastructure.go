package v1alpha1

import (
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
	Size            string           `json:"size"`
	Image           string           `json:"image"`
	Networks        []string         `json:"networks"`
	MachineNetworks []MachineNetwork `json:"machinenetworks"`
	RateLimits      []RateLimit      `json:"ratelimits"`
	EgressRules     []EgressRule     `json:"egressrules"`
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
