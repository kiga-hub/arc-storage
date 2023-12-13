package config

import "github.com/spf13/viper"

const (
	configPprofEnable = "pprof.enable"
)

var defaultPprofConfig = PprofConfig{
	Enable: false,
}

// PprofConfig -
type PprofConfig struct {
	Enable bool `toml:"enable"`
}

// SetDefaultPprofConfig -
func SetDefaultPprofConfig() {
	viper.SetDefault(configPprofEnable, defaultPprofConfig.Enable)
}

// GetPprofConfig -
func GetPprofConfig() *PprofConfig {
	return &PprofConfig{
		Enable: viper.GetBool(configPprofEnable),
	}
}
