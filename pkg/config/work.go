package config

import (
	"github.com/spf13/viper"
)

const (
	configDataPath                    = "arc.dataPath"
	configSaveType                    = "arc.saveType"
	configDebugMod                    = "arc.debugMod"
	configSaveDuration                = "arc.saveDuration"
	configSaveNum                     = "arc.saveNum"
	configWorkCount                   = "arc.workCount"
	configChanCapacity                = "arc.chanCapacity"
	configFileQueueLen                = "arc.arcVolumeQueueLen"
	configFileQueueReadTimeoutSeconds = "arc.arcVolumeQueueReadTimeoutSeconds"
	configFileQueueNum                = "arc.arcVolumeQueueNum"
	configAutomaticallySaveFile       = "arc.allowAutomaticallySaveFile"
	configFrameOffset                 = "arc.frameOffset"
	configTimeOut                     = "arc.timeout"
)

var defaultWorkConfig = WorkConfig{
	DataPath:                         "/data",
	SaveType:                         0,
	DebugMod:                         0,
	SaveDuration:                     "hour",
	SaveNum:                          12,
	WorkCount:                        15,
	ChanCapacity:                     1024,
	ArcVolumeQueueLen:                2048,
	ArcVolumeQueueReadTimeoutSeconds: 10,
	ArcVolumeQueueNum:                2,
	AllowAutomaticallySaveFile:       true,
	FrameOffset:                      5,
	TimeOut:                          300,
}

// WorkConfig 配置
type WorkConfig struct {
	DataPath                         string `toml:"dataPath"`
	SaveType                         int    `toml:"saveType"`
	DebugMod                         int    `toml:"debugMod"`
	SaveDuration                     string `toml:"saveduration"`
	SaveNum                          int    `toml:"saveNum"`
	WorkCount                        int    `toml:"workCount"`
	ChanCapacity                     int    `toml:"chanCapacity"`
	ArcVolumeQueueLen                int    `toml:"arcVolumeQueueLen"`
	ArcVolumeQueueReadTimeoutSeconds int    `toml:"arcVolumeQueueReadTimeoutSeconds"`
	ArcVolumeQueueNum                int    `toml:"arcVolumeQueueNum"`
	AllowAutomaticallySaveFile       bool   `toml:"allowAutomaticallySaveFile"` // 允许每分钟自动保存到文件
	FrameOffset                      int    `toml:"frameOffset"`                // 从缓存查询数据, 多查询的帧数
	TimeOut                          int    `toml:"timeOut"`                    // 超时落盘，单位:s
}

// SetDefaultWorkConfig -
func SetDefaultWorkConfig() {
	viper.SetDefault(configDataPath, defaultWorkConfig.DataPath)
	viper.SetDefault(configSaveType, defaultWorkConfig.SaveType)
	viper.SetDefault(configDebugMod, defaultWorkConfig.DebugMod)
	viper.SetDefault(configSaveDuration, defaultWorkConfig.SaveDuration)
	viper.SetDefault(configSaveNum, defaultWorkConfig.SaveNum)
	viper.SetDefault(configWorkCount, defaultWorkConfig.WorkCount)
	viper.SetDefault(configChanCapacity, defaultWorkConfig.ChanCapacity)

	viper.SetDefault(configFileQueueLen, defaultWorkConfig.ArcVolumeQueueLen)
	viper.SetDefault(configFileQueueReadTimeoutSeconds, defaultWorkConfig.ArcVolumeQueueReadTimeoutSeconds)
	viper.SetDefault(configFileQueueNum, defaultWorkConfig.ArcVolumeQueueNum)

	viper.SetDefault(configAutomaticallySaveFile, defaultWorkConfig.AllowAutomaticallySaveFile)
	viper.SetDefault(configFrameOffset, defaultWorkConfig.FrameOffset)
	viper.SetDefault(configTimeOut, defaultWorkConfig.TimeOut)
}

// GetWorkConfig Get默认配置参数
func GetWorkConfig() *WorkConfig {
	return &WorkConfig{
		DataPath:                         viper.GetString(configDataPath),
		SaveType:                         viper.GetInt(configSaveType),
		DebugMod:                         viper.GetInt(configDebugMod),
		SaveDuration:                     viper.GetString(configSaveDuration),
		SaveNum:                          viper.GetInt(configSaveNum),
		WorkCount:                        viper.GetInt(configWorkCount),
		ChanCapacity:                     viper.GetInt(configChanCapacity),
		ArcVolumeQueueLen:                viper.GetInt(configFileQueueLen),
		ArcVolumeQueueReadTimeoutSeconds: viper.GetInt(configFileQueueReadTimeoutSeconds),
		ArcVolumeQueueNum:                viper.GetInt(configFileQueueNum),
		AllowAutomaticallySaveFile:       viper.GetBool(configAutomaticallySaveFile),
		FrameOffset:                      viper.GetInt(configFrameOffset),
		TimeOut:                          viper.GetInt(configTimeOut),
	}
}
