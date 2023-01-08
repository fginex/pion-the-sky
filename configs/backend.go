package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// BackendConfig represents backend configuration.
type BackendConfig struct {
	flagBase

	BindAddress     string
	ExternalAddress string
}

// InitFromViper initializes this configuration from viper.
func (c *BackendConfig) InitFromViper(v *viper.Viper) {
	c.BindAddress = v.GetString("backend-bind-address")
	c.ExternalAddress = v.GetString("backend-external-address")
}

// NewBackendConfig returns a new backend configuration.
func NewBackendConfig() *BackendConfig {
	return &BackendConfig{}
}

// FlagSet returns an instance of the flag set for the configuration.
func (c *BackendConfig) FlagSet() *pflag.FlagSet {
	if c.initFlagSet() {
		c.flagSet.StringVar(&c.BindAddress, "backend-bind-address", "127.0.0.1:8082", "Host-port to bind the backend server on")
		c.flagSet.StringVar(&c.ExternalAddress, "backend-external-address", "http://localhost:8082", "External address this backend server is reachable at")
	}
	return c.flagSet
}
