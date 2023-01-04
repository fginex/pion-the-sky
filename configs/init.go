package configs

import (
	"strings"
	"sync"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type flagBase struct {
	sync.Mutex
	flagSet *pflag.FlagSet
}

func (fb *flagBase) initFlagSet() bool {
	fb.Lock()
	defer fb.Unlock()
	if fb.flagSet == nil {
		fb.flagSet = &pflag.FlagSet{}
		return true
	}
	return false
}

// InitViper initializes Viper config
func InitViper(configName string) *viper.Viper {
	v := viper.GetViper()
	v.SetConfigName(configName)
	v.AddConfigPath("/etc/boos/")
	v.AddConfigPath("$HOME/.boos")
	v.AddConfigPath("./conf/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AutomaticEnv()
	return v
}

// GetSettingsByPrefixFromViper fetches a map of settings with a given prefix.
func GetSettingsByPrefixFromViper(v *viper.Viper, prefix string) map[string]interface{} {
	allKeys := v.AllKeys()
	outSettings := map[string]interface{}{}
	for _, k := range allKeys {
		if strings.HasPrefix(k, prefix) {
			outSettings[k] = v.Get(k)
		}
	}
	return outSettings
}
