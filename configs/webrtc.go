package configs

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// WebRTCConfig represents WebRTC configuration.
type WebRTCConfig struct {
	flagBase

	ICEServers []string
}

// InitFromViper initializes this configuration from viper.
func (c *WebRTCConfig) InitFromViper(v *viper.Viper) {
	c.ICEServers = v.GetStringSlice("ice-servers")
}

// NewWebRTCConfig returns a new backend configuration.
func NewWebRTCConfig() *WebRTCConfig {
	return &WebRTCConfig{}
}

// FlagSet returns an instance of the flag set for the configuration.
func (c *WebRTCConfig) FlagSet() *pflag.FlagSet {
	if c.initFlagSet() {
		c.flagSet.StringSliceVar(&c.ICEServers, "ice-server", []string{"stun:stun.l.google.com:19302"}, "STUN/TURN server, multiple allowed")
	}
	return c.flagSet
}
