package config

import "github.com/spf13/viper"

const (
	configTaosEnable = "taos.enable"
	configTaosHost   = "taos.host"
	confgInterval    = "taos.interval"
	configTagName    = "taos.tag"
)

var defaultTaosConfig = TaosConfig{
	Host:     "taos",
	Enable:   false,
	Interval: 1,
	TagName:  "arc",
}

// TaosConfig -
type TaosConfig struct {
	Host     string `toml:"host"`
	Enable   bool   `toml:"enable"`
	Interval int    `toml:"interval"`
	TagName  string `toml:"tag_name"`
}

// SetDefaultTaosConfig -
func SetDefaultTaosConfig() {
	viper.SetDefault(configTaosEnable, defaultTaosConfig.Enable)
	viper.SetDefault(configTaosHost, defaultTaosConfig.Host)
	viper.SetDefault(confgInterval, defaultTaosConfig.Interval)
	viper.SetDefault(configTagName, defaultTaosConfig.TagName)
}

// GetTaosConfig -
func GetTaosConfig() *TaosConfig {
	return &TaosConfig{
		Enable:   viper.GetBool(configTaosEnable),
		Host:     viper.GetString(configTaosHost),
		Interval: viper.GetInt(confgInterval),
		TagName:  viper.GetString(configTagName),
	}
}
