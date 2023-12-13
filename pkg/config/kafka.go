package config

import "github.com/spf13/viper"

const (
	configKafkaEnable          = "kafka.enable"
	configKafkaServer          = "kafka.bootstrapServeres"
	configGroupID              = "kafka.groupID"
	configKafkaMessageMaxBytes = "kafka.messageMaxBytes"
	configKafkaTopic           = "kafka.topic"
	configKafkaInterval        = "kafka.interval"
)

var defaultKafkaConfig = KafkaConfig{
	Enable:          false,
	Server:          "kafka-1:9092",
	GroupID:         "arc_id",
	MessageMaxBytes: 67108864,
	Topic:           "arc",
	Interval:        2,
}

// KafkaConfig -
type KafkaConfig struct {
	Enable          bool   `toml:"enable"`
	Server          string `toml:"server"`
	GroupID         string `toml:"groupID"`
	MessageMaxBytes int    `toml:"messageMaxBytes"`
	Topic           string `toml:"topic"`
	Interval        int    `toml:"interval"`
}

// SetDefaultKafkaConfig -
func SetDefaultKafkaConfig() {
	viper.SetDefault(configKafkaEnable, defaultKafkaConfig.Enable)
	viper.SetDefault(configKafkaServer, defaultKafkaConfig.Server)
	viper.SetDefault(configGroupID, defaultKafkaConfig.GroupID)
	viper.SetDefault(configKafkaMessageMaxBytes, defaultKafkaConfig.MessageMaxBytes)
	viper.SetDefault(configKafkaTopic, defaultKafkaConfig.Topic)
	viper.SetDefault(configKafkaInterval, defaultKafkaConfig.Interval)
}

// GetKafkaConfig -
func GetKafkaConfig() *KafkaConfig {
	return &KafkaConfig{
		Enable:          viper.GetBool(configKafkaEnable),
		Server:          viper.GetString(configKafkaServer),
		GroupID:         viper.GetString(configGroupID),
		MessageMaxBytes: viper.GetInt(configKafkaMessageMaxBytes),
		Topic:           viper.GetString(configKafkaTopic),
		Interval:        viper.GetInt(configKafkaInterval),
	}
}
