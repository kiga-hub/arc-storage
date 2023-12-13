package config

import (
	logging "github.com/kiga-hub/arc/logging/conf"
	basic "github.com/kiga-hub/arc/micro/conf"
	tracing "github.com/kiga-hub/arc/tracing"
)

// ArcConfig 配置
type ArcConfig struct {
	Basic *basic.BasicConfig

	Work  *WorkConfig          `toml:"-"`
	Log   *logging.LogConfig   `toml:"-"`
	Trace *tracing.TraceConfig `toml:"-"`
	Cache *CacheConfig         `toml:"-"`
	Kafka *KafkaConfig         `toml:"-"`
	Taos  *TaosConfig          `toml:"-"`
	Grpc  *GRPCConfig          `toml:"-"`
	Pprof *PprofConfig         `toml:"-"`
}

// SetDefaultArcConfig -
func SetDefaultArcConfig() {
	basic.SetDefaultBasicConfig()
	SetDefaultWorkConfig()
	SetDefaultCacheConfig()
	SetDefaultKafkaConfig()
	SetDefaultTaosConfig()
	SetDefaultGRPCConfig()
	SetDefaultPprofConfig()
}

// GetConfig Get默认配置参数
func GetConfig() *ArcConfig {
	return &ArcConfig{
		Basic: basic.GetBasicConfig(),

		Work:  GetWorkConfig(),
		Cache: GetCacheConfig(),
		Kafka: GetKafkaConfig(),
		Taos:  GetTaosConfig(),
		Grpc:  GetGRPCConfig(),
		Pprof: GetPprofConfig(),

		Log:   logging.GetLogConfig(),
		Trace: tracing.GetTraceConfig(),
	}
}
