package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FrontEndConfig represents frontend configuration.
type FrontEndConfig struct {
	flagBase

	BindAddress     string
	ExternalAddress string
}

// InitFromViper initializes this configuration from viper.
func (c *FrontEndConfig) InitFromViper(v *viper.Viper) {
	c.BindAddress = v.GetString("frontend-bind-address")
	c.ExternalAddress = v.GetString("frontend-external-address")
}

// NewFrontEndConfig returns a new frontend configuration.
func NewFrontEndConfig() *FrontEndConfig {
	return &FrontEndConfig{}
}

// FlagSet returns an instance of the flag set for the configuration.
func (c *FrontEndConfig) FlagSet() *pflag.FlagSet {
	if c.initFlagSet() {
		c.flagSet.StringVar(&c.BindAddress, "frontend-bind-address", "127.0.0.1:8083", "Host-port to bind the frontend server on")
		c.flagSet.StringVar(&c.ExternalAddress, "frontend-external-address", "http://localhost:8083", "External address this frontend server is available on")
	}
	return c.flagSet
}
