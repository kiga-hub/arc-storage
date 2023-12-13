package config

import "github.com/spf13/viper"

const (
	cacheenable     = "cache.enable"
	cachetimeoutmin = "cache.timeoutmin"
	cacheexpire     = "cache.expirems"
	cachesearch     = "cache.search"
)

var defaultCacheConfig = CacheConfig{
	Enable:     true,
	TimeoutMin: 3,
	ExpireMs:   120000,
	Search:     true,
}

//CacheConfig struct
type CacheConfig struct {
	Enable     bool `toml:"enable"`
	TimeoutMin int  `toml:"timeoutmin"`
	ExpireMs   int  `toml:"expirems"`
	Search     bool `toml:"search"`
}

//SetDefaultCacheConfig -
func SetDefaultCacheConfig() {
	viper.SetDefault(cacheenable, defaultCacheConfig.Enable)
	viper.SetDefault(cachetimeoutmin, defaultCacheConfig.TimeoutMin)
	viper.SetDefault(cacheexpire, defaultCacheConfig.ExpireMs)
	viper.SetDefault(cachesearch, defaultCacheConfig.Search)
}

//GetCacheConfig -
func GetCacheConfig() *CacheConfig {
	return &CacheConfig{
		Enable:     viper.GetBool(cacheenable),
		TimeoutMin: viper.GetInt(cachetimeoutmin),
		ExpireMs:   viper.GetInt(cacheexpire),
		Search:     viper.GetBool(cachesearch),
	}
}
