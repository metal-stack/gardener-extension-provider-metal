package mutator

import (
	"os"
	"strconv"
	"strings"

	calicoextensionv1alpha1 "github.com/gardener/gardener-extension-networking-calico/pkg/apis/calico/v1alpha1"
	ciliumextensionv1alpha1 "github.com/gardener/gardener-extension-networking-cilium/pkg/apis/cilium/v1alpha1"
)

type config struct{}

func (c *config) allowedPrivilegedContainers() bool {
	return c.bool("DEFAULTER_ALLOWEDPRIVILEGEDCONTAINERS", true)
}

func (c *config) maxPods() int32 {
	return c.int32("DEFAULTER_MAXPODS", 250)
}

func (c *config) nodeCIDRMaskSize() int32 {
	return c.int32("DEFAULTER_NODECIDRMASKSIZE", 23)
}

func (c *config) podsCIDR() string {
	return c.string("DEFAULTER_PODSCIDR", "10.240.0.0/13")
}

func (c *config) servicesCIDR() string {
	return c.string("DEFAULTER_SERVICESCIDR", "10.248.0.0/18")
}

func (c *config) networkType() string {
	return c.string("DEFAULTER_NETWORKTYPE", "calico")
}

func (c *config) calicoBackend() calicoextensionv1alpha1.Backend {
	return calicoextensionv1alpha1.Backend(c.string("DEFAULTER_CALICOBACKEND", string(calicoextensionv1alpha1.None)))
}

func (c *config) calicoKubeProxyEnabled() bool {
	return c.bool("DEFAULTER_CALICOKUBEPROXYENABLED", true)
}

func (c *config) calicoPoolMode() calicoextensionv1alpha1.IPv4PoolMode {
	return calicoextensionv1alpha1.IPv4PoolMode(c.string("DEFAULTER_CALICOPOOLMODE", string(calicoextensionv1alpha1.Never)))
}

func (c *config) calicoTyphaEnabled() bool {
	return c.bool("DEFAULTER_CALICOTYPHAENABLED", false)
}

func (c *config) ciliumHubbleEnabled() bool {
	return c.bool("DEFAULTER_CILIUMHUBBLEENABLED", true)
}

func (c *config) ciliumKubeProxyEnabled() bool {
	return c.bool("DEFAULTER_CILIUMKUBEPROXYENABLED", false)
}

func (c *config) ciliumPSPEnabled() bool {
	return c.bool("DEFAULTER_CILIUMPSPENABLED", false)
}

func (c *config) ciliumTunnel() ciliumextensionv1alpha1.TunnelMode {
	return ciliumextensionv1alpha1.TunnelMode(c.string("DEFAULTER_CILIUMTUNNEL", string(ciliumextensionv1alpha1.Disabled)))
}

func (c *config) ciliumDevices() []string {
	return c.slice("DEFAULTER_CILIUMDEVICES", []string{"lan+", "lo"})
}

func (c *config) ciliumDirectRoutingDevice() string {
	return c.string("DEFAULTER_CILIUMDIRECTROUTINGDEVICE", "lo")
}

func (c *config) bgpControlPlaneEnabled() bool {
	return c.bool("DEFAULTER_CILIUMBGPCONTROLPLANE", true)
}

func (c *config) ciliumIPv4NativeRoutingCIDREnabled() bool {
	return c.bool("DEFAULTER_CILIUMIPV4NATIVEROUTINGCIDRENABLED", true)
}

func (c *config) ciliumLoadBalancingMode() ciliumextensionv1alpha1.LoadBalancingMode {
	return ciliumextensionv1alpha1.LoadBalancingMode(c.string("DEFAULTER_CILIUMLOADBALANCINGMODE", string(ciliumextensionv1alpha1.DSR)))
}

func (c *config) ciliumMTU() int {
	return int(c.int32("DEFAULTER_CILIUMMTU", 1440))
}

func (c *config) bool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func (c *config) string(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return value
}

func (c *config) slice(key string, fallback []string) []string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	return strings.Split(value, ",")
}

func (c *config) int32(key string, fallback int32) int32 {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return fallback
	}

	return int32(parsed)
}
