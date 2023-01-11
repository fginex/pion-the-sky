package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// FrontendConfig represents frontend configuration.
type FrontendConfig struct {
	flagBase

	BindAddress     string
	ExternalAddress string

	StaticDirectoryPath         string
	StaticDirectoryRootDocument string
}

// InitFromViper initializes this configuration from viper.
func (c *FrontendConfig) InitFromViper(v *viper.Viper) {
	c.BindAddress = v.GetString("frontend-bind-address")
	c.ExternalAddress = v.GetString("frontend-external-address")
	c.StaticDirectoryPath = v.GetString("static-directory-path")
	c.StaticDirectoryRootDocument = v.GetString("static-directory-root-document")
}

// NewFrontendConfig returns a new frontend configuration.
func NewFrontendConfig() *FrontendConfig {
	return &FrontendConfig{}
}

// FlagSet returns an instance of the flag set for the configuration.
func (c *FrontendConfig) FlagSet() *pflag.FlagSet {
	if c.initFlagSet() {
		c.flagSet.StringVar(&c.BindAddress, "frontend-bind-address", "127.0.0.1:8083", "Host-port to bind the frontend server on")
		c.flagSet.StringVar(&c.ExternalAddress, "frontend-external-address", "http://localhost:8083", "External address this frontend server is available on")
		c.flagSet.StringVar(&c.StaticDirectoryPath, "static-directory-path", "./ui/build", "Frontend static directory path")
		c.flagSet.StringVar(&c.StaticDirectoryRootDocument, "static-directory-root-document", "index.html", "Frontend static directory root document")
	}
	return c.flagSet
}
