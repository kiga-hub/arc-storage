package config

import "github.com/spf13/viper"

const (
	configGRPCEnable = "grpc.enable"
	configGRPCServer = "grpc.server"
)

var defaultGRPCConfig = GRPCConfig{
	Enable: false,
	Server: ":8080",
}

// GRPCConfig -
type GRPCConfig struct {
	Enable bool   `toml:"enable"`
	Server string `toml:"server"`
}

// SetDefaultGRPCConfig -
func SetDefaultGRPCConfig() {
	viper.SetDefault(configGRPCEnable, defaultGRPCConfig.Enable)
	viper.SetDefault(configGRPCServer, defaultGRPCConfig.Server)
}

// GetGRPCConfig -
func GetGRPCConfig() *GRPCConfig {
	return &GRPCConfig{
		Enable: viper.GetBool(configGRPCEnable),
		Server: viper.GetString(configGRPCServer),
	}
}
