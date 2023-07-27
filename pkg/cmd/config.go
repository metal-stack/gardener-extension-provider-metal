package cmd

import (
	"fmt"

	healthcheckconfig "github.com/gardener/gardener/extensions/pkg/apis/config"
	"github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config"
	configloader "github.com/metal-stack/gardener-extension-provider-metal/pkg/apis/config/loader"

	"github.com/spf13/pflag"
)

// ConfigOptions are command line options that can be set for config.ControllerConfiguration.
type ConfigOptions struct {
	// Kubeconfig is the path to a kubeconfig.
	ConfigFilePath string

	config *Config
}

// Config is a completed controller configuration.
type Config struct {
	// Config is the controller configuration.
	Config *config.ControllerConfiguration
}

func (c *ConfigOptions) buildConfig() (*config.ControllerConfiguration, error) {
	if len(c.ConfigFilePath) == 0 {
		return nil, fmt.Errorf("config file path not set")
	}
	return configloader.LoadFromFile(c.ConfigFilePath)
}

// Complete implements RESTCompleter.Complete.
func (c *ConfigOptions) Complete() error {
	config, err := c.buildConfig()
	if err != nil {
		return err
	}

	c.config = &Config{config}
	return nil
}

// Completed returns the completed Config. Only call this if `Complete` was successful.
func (c *ConfigOptions) Completed() *Config {
	return c.config
}

// AddFlags implements Flagger.AddFlags.
func (c *ConfigOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.ConfigFilePath, "config-file", "", "path to the controller manager configuration file")
}

// Apply sets the values of this Config in the given config.ControllerConfiguration.
func (c *Config) Apply(cfg *config.ControllerConfiguration) {
	*cfg = *c.Config
}

// ApplyMachineImages sets the given machine images to those of this Config.
func (c *Config) ApplyMachineImages(machineImages *[]config.MachineImage) {
	*machineImages = c.Config.MachineImages
}

// ApplyETCD sets the given etcd configuration to that of this Config.
func (c *Config) ApplyETCD(etcd *config.ETCD) {
	*etcd = c.Config.ETCD
}

// ApplyControllerConfig sets the given controller configuration to that of this Config.
func (c *Config) ApplyControllerConfig(controllerConfig *config.ControllerConfiguration) {
	*controllerConfig = *c.Config
}

// Options initializes empty config.ControllerConfiguration, applies the set values and returns it.
func (c *Config) Options() config.ControllerConfiguration {
	var cfg config.ControllerConfiguration
	c.Apply(&cfg)
	return cfg
}

// ApplyHealthCheckConfig applies the HealthCheckConfig to the config
func (c *Config) ApplyHealthCheckConfig(config *healthcheckconfig.HealthCheckConfig) {
	if c.Config.HealthCheckConfig != nil {
		*config = *c.Config.HealthCheckConfig
	}
}
