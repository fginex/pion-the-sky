package configs

import (
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// LogConfig represents logging configuration.
type LogConfig struct {
	flagBase

	LogLevel      string
	LogColor      bool
	LogForceColor bool
	LogAsJSON     bool
}

// InitFromViper initializes this configuration from viper.
func (c *LogConfig) InitFromViper(v *viper.Viper) {
	c.LogLevel = v.GetString("log-level")
	c.LogColor = v.GetBool("log-color")
	c.LogForceColor = v.GetBool("log-force-color")
	c.LogAsJSON = v.GetBool("log-as-json")
}

// NewLoggingConfig returns a new logging configuration.
func NewLoggingConfig() *LogConfig {
	return &LogConfig{}
}

// FlagSet returns an instance of the flag set for the configuration.
func (c *LogConfig) FlagSet() *pflag.FlagSet {
	if c.initFlagSet() {
		c.flagSet.StringVar(&c.LogLevel, "log-level", "info", "Log level")
		c.flagSet.BoolVar(&c.LogAsJSON, "log-as-json", false, "Log as JSON")
		c.flagSet.BoolVar(&c.LogColor, "log-color", false, "Log in color")
		c.flagSet.BoolVar(&c.LogForceColor, "log-force-color", false, "Force colored log output")
	}
	return c.flagSet
}

// NewLogger returns a new configured logger.
func (c *LogConfig) NewLogger(name string) hclog.Logger {
	loggerColorOption := hclog.ColorOff
	if c.LogColor {
		loggerColorOption = hclog.AutoColor
	}
	if c.LogForceColor {
		loggerColorOption = hclog.ForceColor
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       name,
		Level:      hclog.LevelFromString(c.LogLevel),
		Color:      loggerColorOption,
		JSONFormat: c.LogAsJSON,
	})
}
