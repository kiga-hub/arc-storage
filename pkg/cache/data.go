package cache

import (
	"fmt"
	"math"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/config"
	"github.com/kiga-hub/arc/cache"
	"github.com/kiga-hub/arc/logging"
	"github.com/spf13/cast"
)

// DataCacheRepo 音频实时缓存
type DataCacheRepo struct {
	Container *cache.DataCacheContainer
}

// NewCacheRepo -
func NewCacheRepo(logger logging.ILogger) *DataCacheRepo {
	d := &DataCacheRepo{
		Container: cache.NewDataCacheContainer(
			config.GetCacheConfig().TimeoutMin,
			int64(config.GetCacheConfig().ExpireMs*1000),
			config.GetCacheConfig().Search,
			logger),
	}
	logger.Infow("start audio cache service...")
	d.Container.Start(config.GetConfig().Basic.IsDevMode)

	return d
}

// Close -
func (d *DataCacheRepo) Close() {
	d.Container.Stop()
}

// Stat -
func (d *DataCacheRepo) Stat(id uint64) *cache.DataCacheStat {
	stats := d.Container.GetStat()
	if st, ok := stats[id]; ok {
		return st
	}
	return nil
}

// Input -
func (d *DataCacheRepo) Input(data *cache.DataPoint) {
	d.Container.Input(data)
}

// Search -
func (d *DataCacheRepo) Search(logger logging.ILogger, id uint64, from, to time.Time, frameOffset int) ([]byte, int, error) {

	sampleRate := float64(8000) // 音频最低采样率
	channel := 1
	// 两帧时间差,单位ms
	gap := (float64(frameOffset) / sampleRate) * 1e6
	// previousTimestamp开始时间多取1帧时间, previousTimestamp < t1 < 缓存中的开始时间戳
	previousTimestamp := from.Add(-time.Duration(gap) * time.Millisecond)
	// 结束时间多取1帧数据
	nextTimestamp := to.Add(time.Duration(gap) * time.Millisecond)

	logger.Info("Search", id, previousTimestamp.UTC().UnixMicro(), nextTimestamp.UTC().UnixMicro())
	dataPoint, err := d.Container.Search(&cache.SearchRequest{
		ID:       id,
		TimeFrom: previousTimestamp.UTC().UnixMicro(), //us
		TimeTo:   nextTimestamp.UTC().UnixMicro(),     //us
	})
	if err != nil {
		return nil, 0, err
	}

	// 查询到的数据分段的个数,缓存数据不连续会返回多段数据
	dataCount := len(dataPoint)
	logger.Info("search", "dataCount", dataCount, "id", id)

	if dataCount != 1 {
		logger.Warnw("dataCount!=1", "dataCount", dataCount)
		return nil, dataCount, fmt.Errorf("multi data or empty")
	}

	data := dataPoint[0].(*cache.DataPoint)

	// 切分查询到的数据
	sub := int64(math.Abs(float64(from.Sub(previousTimestamp).Seconds()) * sampleRate * 2 * float64(channel)))
	sub = sub - sub%2

	// 获取数据预期大小
	expectSize := cast.ToInt64(to.Sub(from).Seconds()) * 2 * int64(channel)

	if len(data.Data) < 1 {
		stat := d.Container.GetStat()
		if stat != nil {
			logger.Warnw("The query time is within the cache time range |dataLen<1|stat!=nil",
				"sensorID-int", id,
				"currentSystemTime", time.Now().UTC(),
				"from", from.UTC(),
				"to", to.UTC(),
				"stat size", stat[id].Size,
				"stat from", stat[id].From,
				"stat to", stat[id].To,
				"stat expire", stat[id].Expire,
				"stat count", stat[id].Count,
			)
		} else {
			logger.Warnw("The query time is within the cache time range |dataLen<1|stat!=nil",
				"sensorID-int", id,
				"currentSystemTime", time.Now().UTC(),
				"from", from.UTC(),
				"to", to.UTC(),
			)
		}
		return nil, 0, fmt.Errorf("not found data %s", err.Error())
	}

	var wavData []byte
	// 越界判断
	if int64(len(data.Data)) >= sub+expectSize && sub < int64(len(data.Data)) {
		wavData = data.Data[sub : sub+expectSize]
	} else {
		wavData = data.Data
	}

	logger.Infow("searchAudio", "from", from, "previousTimestamp", previousTimestamp, "to", to, "nextTimestamp", nextTimestamp,
		"expectSize", expectSize, "wavDataLen", len(wavData), "sub", sub, "data.Data", len(data.Data))

	return wavData, 1, err
}
