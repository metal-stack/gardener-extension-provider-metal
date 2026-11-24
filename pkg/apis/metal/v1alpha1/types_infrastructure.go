package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SuricataListenMode sets the listen mode for suricata
type SuricataListenMode string

const (
	// SuricataListenModeAfPacket configures suricata that it is sniffing traffic and emitting alerts (IDS).
	SuricataListenModeAfPacket SuricataListenMode = "af-packet"
	// SuricataListenModeNfqueue configures suricata that it can actively drop traffic (IPS).
	SuricataListenModeNfqueue SuricataListenMode = "nfqueue"
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
	Size                   string       `json:"size"`
	Image                  string       `json:"image"`
	Networks               []string     `json:"networks"`
	RateLimits             []RateLimit  `json:"rateLimits"`
	EgressRules            []EgressRule `json:"egressRules"`
	LogAcceptedConnections bool         `json:"logAcceptedConnections"`
	ControllerVersion      string       `json:"controllerVersion"`
	AutoUpdateMachineImage bool         `json:"autoUpdateMachineImage,omitempty"`
	// FirewallHealthTimeout is the duration after a created firewall not getting ready is considered dead.
	// If set to 0, the timeout is disabled.
	// +optional
	FirewallHealthTimeout *metav1.Duration `json:"firewallHealthTimeout,omitempty"`
	// FirewallCreateTimeout is the duration after which a firewall in the creation phase will be recreated.
	// +optional
	FirewallCreateTimeout *metav1.Duration `json:"firewallCreateTimeout,omitempty"`
	// Suricata configures suricata on the firewall.
	Suricata Suricata `json:"suricata,omitempty"`
}

type RateLimit struct {
	NetworkID string `json:"networkID"`
	RateLimit uint32 `json:"rateLimit"`
}

type EgressRule struct {
	NetworkID string   `json:"networkID"`
	IPs       []string `json:"ips"`
}

type Suricata struct {
	// Enabled enables suricata. Enabled by default.
	Enabled *bool `json:"enabled"`
	// Evebox starts an evebox agent on the firewall that forwards it to an evebox server in the shoot.
	EveboxEnabled bool `json:"eveboxEnabled"`
	// ListenMode configures the suricata listen mode. when set to nfqueue suricata will actively filter packets instead of just sniffing traffic.
	ListenMode *SuricataListenMode `json:"listenMode,omitempty"`
	// DetectionRulesets configures detection rulesets for suricata. The available values can be determined from the FirewallMonitor status. If this slice is nil, the default detection ruleset is used.
	DetectionRulesets []string `json:"detectionRulesets,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InfrastructureStatus contains information about created infrastructure resources.
type InfrastructureStatus struct {
	metav1.TypeMeta `json:",inline"`
	Firewall        FirewallStatus `json:"firewall"`
}

type FirewallStatus struct {
	MachineID string `json:"machineID"`
}
