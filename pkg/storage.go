package pkg

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/api"
	"github.com/kiga-hub/arc-storage/pkg/arc_volume"
	"github.com/kiga-hub/arc-storage/pkg/cache"
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/kiga-hub/arc-storage/pkg/util"
	"github.com/kiga-hub/arc/protocols"
	"github.com/kiga-hub/arc/utils"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

var (
	// AllowFileMap 上传文件格式
	AllowFileMap = map[string]bool{".pcm": true}
	// UploadFileType 上传文件数据类型
	UploadFileType = map[string]string{"audio": "A", "vibrate": "V"}
)

// NumericalTableDataRequest -
type NumericalTableDataRequest struct {
	SensorID string      `json:"sensorid"`
	TS       int64       `json:"ts"`
	SType    byte        `json:"stype"`
	TType    byte        `json:"ttype"`
	Data     interface{} `json:"data"`
}

// getSensorIDsfromStorage -
func (arc *ArcStorage) getSensorIDsfromStorage() ([]string, error) {
	var sensorids []string
	sensoridfolders, err := ioutil.ReadDir(arc.config.Work.DataPath)
	if err != nil {
		arc.logger.Errorw("getSensorIDsfromStorage", "dataPath", arc.config.Work.DataPath, "err", err)
		return []string{}, err
	}
	for _, filename := range sensoridfolders {
		if filename.IsDir() {
			sensorids = append(sensorids, filename.Name())
		}
	}
	return sensorids, nil
}

// getSensorIDs Get Sensor IDs
func (arc *ArcStorage) getSensorIDs(c echo.Context) error {
	sensorids, err := arc.getSensorIDsfromStorage()
	if err != nil {
		arc.logger.Errorw("getSensorIDsfromStorage", "err", err)
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}
	if len(sensorids) < 1 {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	// return SensorIDResponse
	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: api.Success,
		Msg:  "OK",
		Data: sensorids},
	)
}

// getSensorLists metadata from needle & parse data to buffer
func (arc *ArcStorage) getSensorLists(c echo.Context) error {
	var err error
	sensorIDStr := c.QueryParam("sensorid")
	if sensorIDStr == "" {
		arc.logger.Errorw("sensorid is null", "sensorid", sensorIDStr)
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	sensorid := strings.ToUpper(sensorIDStr)
	filetype := c.QueryParam("type")

	var AllowExtMap map[string]bool = map[string]bool{
		"audio":   true,
		"vibrate": true,
	}
	if _, ok := AllowExtMap[filetype]; !ok {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	var t1 time.Time
	var t2 time.Time

	from := c.QueryParam("from")
	to := c.QueryParam("to")
	if util.IsDigit(from) && util.IsDigit(to) { //如果传的是时间戳，则认为是utc时间
		if len(from) == 10 && len(to) == 10 {
			t1 = time.Unix(cast.ToInt64(from), 0)
			t2 = time.Unix(cast.ToInt64(to), 0)
		} else if len(from) == 13 && len(to) == 13 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e6)
			t2 = time.Unix(0, cast.ToInt64(to)*1e6)
		} else if len(from) == 16 && len(to) == 16 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e3)
			t2 = time.Unix(0, cast.ToInt64(to)*1e3)
		} else if len(from) == 19 && len(to) == 19 {
			t1 = time.Unix(0, cast.ToInt64(from))
			t2 = time.Unix(0, cast.ToInt64(to))
		} else {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
	} else { //不是时间戳就认为是字符串 没有自定义时区则默认东八区

		from = strings.ReplaceAll(from, " ", "T")
		to = strings.ReplaceAll(to, " ", "T")

		//长度不够20即认为没有带时区信息
		if len(from) < 20 {
			from += "+08:00"
		}
		if len(to) < 20 {
			to += "+08:00"
		}

		t1, err = time.Parse(time.RFC3339, from)
		if err != nil || t1.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
		t2, err = time.Parse(time.RFC3339, to)
		if err != nil || t2.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
	}

	if t2.Before(t1) || t2.Equal(t1) {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  "time t1 = t2"},
		)
	}

	// 获取以天为单位的时间范围
	daysdiffer, count, err := util.GetDaysDiffer(t1.UTC().Format("2006-01-02 15:04:05"), t2.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  err.Error()},
		)
	}

	start := time.Now()
	defer func() {
		arc.logger.Infof("get data spend %s\n", time.Since(start).String())
	}()

	// 获取大文件列表
	bigfilelists := make(map[string]string, count)

	for _, day := range daysdiffer {
		path := arc.config.Work.DataPath + "/" + sensorid + "/" + day + "/" + filetype
		err := util.GetBigFileLists(path, bigfilelists, "")
		if err != nil {
			arc.logger.Errorw("GetBigFileLists", "dataPath", path, "err", err)
			// 遍历文件夹，没有则跳过
			continue
		}
	}

	var timestamplist []string
	var starttimestamps []string

	for key := range bigfilelists {
		timestamplist = append(timestamplist, key)
	}

	sort.Strings(timestamplist)

	for _, creattime := range timestamplist {
		begin, end, _, _, _, err := util.GetTimeRangeFromFileName(bigfilelists[creattime])
		if err != nil {
			arc.logger.Errorw("GetTimeRangeFromFileName", "fileName", bigfilelists[creattime], "err", err)
			return c.JSON(http.StatusNotFound, utils.ResponseV2{
				Code: http.StatusNotFound,
				Msg:  http.StatusText(http.StatusNotFound)},
			)
		}
		// 开始时间大于文件保存结束时间，忽略
		if end.Before(t1) {
			continue
		}

		// 结束时间小于文件创建时间，退出
		if t2.Before(begin) {
			arc.logger.Debugw("compare timestamp", "t2", t2, "begin", begin)
			continue
		}
		starttimestamps = append(starttimestamps, creattime)
	}

	if len(starttimestamps) < 1 {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	// 排序符合时间段
	sort.Strings(starttimestamps)
	searchlists := []api.SensorItem{}

	for _, v := range starttimestamps {
		fullpath := bigfilelists[v][strings.LastIndex(bigfilelists[v], "/")+1:]
		start, end, _, _, sam, err := util.GetTimeRangeFromFileName(fullpath)
		if err != nil {
			arc.logger.Errorw("GetTimeRangeFromFileName", "path", fullpath, "err", err)
			return c.JSON(http.StatusNotFound, utils.ResponseV2{
				Code: http.StatusNotFound,
				Msg:  http.StatusText(http.StatusNotFound)},
			)
		}

		duration := end.Sub(start).Microseconds()

		item := api.SensorItem{
			SensorID:     sensorid,
			DataType:     filetype,
			SampleRate:   sam,
			Channel:      1,
			TimeFrom:     start.UnixNano() / 1e3, // 返回微妙级别时间戳
			TimeTo:       end.UnixNano() / 1e3,
			TimeDuration: duration,

			DataSize: duration * int64(cast.ToInt(sam)*1*2) / 1e6,
			Query: api.SensorQuery{
				URL:      "http://arc-storage/api/data/v1/history/arc?sensorid=" + sensorid + "&type=" + filetype + "&from=" + cast.ToString(start.UnixNano()/1e3) + "&to=" + cast.ToString(end.UnixNano()/1e3),
				Scheme:   "http",
				Domain:   "arc-storage",
				Port:     80,
				FullPath: "api/data/v1/history/arc?sensorid=" + sensorid + "&type=" + filetype + "&from=" + cast.ToString(start.UnixNano()/1e3) + "&to=" + cast.ToString(end.UnixNano()/1e3),
				Path:     "api/data/v1/history/arc",
				SensorID: sensorid,
				Type:     filetype,
				TimeFrom: start.UnixNano() / 1e3,
				TimeTo:   end.UnixNano() / 1e3,
			},
		}
		searchlists = append(searchlists, item)
	}

	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: api.Success,
		Msg:  "OK",
		Data: searchlists},
	)
}

// getArcStorageData - download multi audio files || return file list
func (arc *ArcStorage) getArcStorageData(c echo.Context) error {
	var err error
	sensorid := c.QueryParam("sensorid")
	if sensorid == "" {
		arc.logger.Errorw("sensorid is empty")
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}
	sensorid = strings.ToUpper(sensorid)

	filetype := c.QueryParam("type")
	var AllowExtMap map[string]bool = map[string]bool{
		"audio":   true,
		"vibrate": true,
	}
	if _, ok := AllowExtMap[filetype]; !ok {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	var t1 time.Time
	var t2 time.Time

	//适配多参数名
	fromString := c.QueryParam("from_string")
	fromTimestamp := c.QueryParam("from")
	toString := c.QueryParam("to_string")
	toTimestamp := c.QueryParam("to")

	var from, to string
	if fromString != "" {
		from = fromString
	}
	if fromTimestamp != "" {
		from = fromTimestamp
	}
	if toString != "" {
		to = toString
	}
	if toTimestamp != "" {
		to = toTimestamp
	}

	//开始多格式适配及校验
	if from == "" || to == "" {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	if util.IsDigit(from) && util.IsDigit(to) { //如果传的是时间戳，则认为是utc时间
		if len(from) == 10 && len(to) == 10 {
			t1 = time.Unix(cast.ToInt64(from), 0)
			t2 = time.Unix(cast.ToInt64(to), 0)
		} else if len(from) == 13 && len(to) == 13 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e6)
			t2 = time.Unix(0, cast.ToInt64(to)*1e6)
		} else if len(from) == 16 && len(to) == 16 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e3)
			t2 = time.Unix(0, cast.ToInt64(to)*1e3)
		} else if len(from) == 19 && len(to) == 19 {
			t1 = time.Unix(0, cast.ToInt64(from))
			t2 = time.Unix(0, cast.ToInt64(to))
		} else {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
	} else { //不是时间戳就认为是字符串 没有自定义时区则默认东八区

		from = strings.ReplaceAll(from, " ", "T")
		to = strings.ReplaceAll(to, " ", "T")

		//长度不够20即认为没有带时区信息
		if len(from) < 20 {
			from += "+08:00"
		}
		if len(to) < 20 {
			to += "+08:00"
		}

		t1, err = time.Parse(time.RFC3339, from)
		if err != nil || t1.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
		t2, err = time.Parse(time.RFC3339, to)
		if err != nil || t2.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
	}

	t1 = t1.UTC()
	t2 = t2.UTC()

	if t2.Before(t1) || t2.Equal(t1) {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	// TODO Tw 缓存开始时间
	duration := time.Duration(arc.config.Cache.ExpireMs) * time.Millisecond
	ms, err := time.ParseDuration("-" + cast.ToString(duration))
	if err != nil {
		arc.logger.Errorw("TimeParseDuration", "sensorid", sensorid, "err", err)
	}

	//系统当前时间
	now := time.Now().UTC()

	//查询时间段的开始时间不能大于等于系统当前时间
	if t1.After(now) || t1.Equal(now) {
		arc.logger.Infow("query time t1 exceeds the current system time",
			"sensorid", sensorid,
			"currentSystemTime", now,
			"start", t1.UTC(),
			"end", t2.UTC(),
		)
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  "The start time for querying data cannot be greater than or equal to the current system time"},
		)
	}

	//缓存有效起始时间
	tw := now.Add(ms)

	// 查询时间 t1,t2早于开始缓存时间. 从落盘文件中查找数据
	if (t1.Before(tw) && t2.Before(tw)) || tw.Unix() < 0 {
		arc.logger.Infow("query time t1, t2 is earlier than the cache start time. find data from the disk", "sensorid", sensorid, "t1", t1, "t2", t2, "tw", tw)
		var handler *arc_volume.ArcVolumeCache
		if filetype == "audio" {
			handler = arc.audioFileStore
		}

		fileType := c.QueryParam("type")

		data, other := handler.ReadDataByQueue(sensorid, fileType, t1, t2)
		if other != nil {
			other.Msg += "|hd"
			return c.JSON(other.Code, other)
		}
		return c.Blob(http.StatusOK, "audio/wav", data)
	}

	// TODO 查询时间 t1早于开始缓存时间,t2在开始缓存时间之后
	if t1.Before(tw) && tw.Before(t2) && tw.Unix() > 0 {
		arc.logger.Infow("T1 is earlier than the cache start time,t2 is after the cache time,get disk data + cage data", "sensorid", sensorid, "t1", t1, "t2", t2, "tw", tw)
		// TODO  落盘数据 + 缓存数据
		var handler *arc_volume.ArcVolumeCache
		if filetype == "audio" {
			handler = arc.audioFileStore
		}

		fileType := c.QueryParam("type")

		data, other := handler.ReadDataByQueue(sensorid, fileType, t1, t2)
		if other != nil {
			other.Msg += "|hd"
			return c.JSON(other.Code, other)
		}
		return c.Blob(http.StatusOK, "audio/wav", data)
	}

	// 查询时间t1,t2 在缓存时间范围内,从缓存中读取数据
	if tw.Before(t1) && tw.Before(t2) && tw.Unix() > 0 {
		return arc.readDataFromCache(c, sensorid, filetype, t1, t2, tw)
	}

	arc.logger.Infow("getARcDataNotFoundData", "sensorid", sensorid, "t1", t1.UTC(), "t2", t2.UTC())
	arc.exportMetrics.SetCacheReadValues(sensorid, metric.MonitorFailed)
	return c.JSON(http.StatusNotFound, utils.ResponseV2{
		Code: http.StatusNotFound,
		Msg:  http.StatusText(http.StatusNotFound) + ": " + sensorid},
	)
}

// uploadPCMData - upload pcm data 2 server
func (arc *ArcStorage) uploadPCMData(c echo.Context) error {
	file, err := c.FormFile("filename")
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	ext := path.Ext(file.Filename)

	if _, ok := AllowFileMap[ext]; !ok {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	params, err := c.FormParams()
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}
	// 参数校验是否为空
	if err = checkUploadParam(params); err != nil {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}
	dataType := params.Get("type")
	// 校验上传数据类型
	_, err = checkUploadType(dataType)
	if err != nil {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	var t1 time.Time
	from := params.Get("from")

	if len(from) == 13 && util.IsDigit(from) {
		t1 = time.Unix(0, cast.ToInt64(from)*int64(time.Millisecond))
	} else {
		from = strings.Replace(from, " ", "T", 1) + "+08:00"

		t1, err = time.Parse(time.RFC3339, from)
		if err != nil || t1.Unix() < 0 {
			return c.JSON(http.StatusNotFound, utils.ResponseV2{
				Code: http.StatusNotFound,
				Msg:  http.StatusText(http.StatusNotFound)},
			)
		}
	}

	// 先打开文件源
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			return
		}
	}(src)

	data, err := ioutil.ReadAll(src)
	if err != nil {
		arc.logger.Errorw("ReadFileFailed", "fileName", file.Filename, "err", err)
	}

	sampleRate := params.Get("sample_rate")
	channel := params.Get("channel")

	// 根据数据量大小修改保存文件保结束时间
	interval := float64(len(data)) / cast.ToFloat64(sampleRate) / float64(2*int(cast.ToInt(channel))) * 1e9
	t2 := t1.Add(time.Nanosecond * time.Duration(interval))

	sensorID := params.Get("sensorid")
	hexID, err := hex.DecodeString(sensorID)
	if err != nil {
		arc.logger.Errorw("hex.DecodeString", "sensorID", sensorID, "hexID", hexID, "err", err)
	}
	sensorID = strings.ToUpper(sensorID)

	var gain uint8
	if params.Get("gain") == "" {
		gain = 1
	} else {
		gainInt := cast.ToInt64(params.Get("gain"))
		if gainInt >= 0 && gainInt <= math.MaxUint8 {
			gain = cast.ToUint8(gainInt)
		} else {
			gain = 1
		}
	}

	var segmentData protocols.ISegment

	id := ByteToUInt64(hexID)

	// 处理id对应的传感器缓存数据，并清空数据
	arc.loadAndStoreTimeOutData(id)
	time.Sleep(time.Millisecond * 100)

	buf, err := ParsedFrameConstructor(segmentData, id, t1.UTC())
	if err != nil {
		arc.logger.Errorw("parsedFrameConstructor", "id", id, "dataType", dataType, "err", err)
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	// 上传文件过一遍数据解析流程,数据由decodeJob进行处理，包括缓存以及fileCache
	arc.decodeJobChans[id&uint64(arc.config.Work.WorkCount-1)] <- buf

	arc.logger.Info("UploadPCMData", "sensorID", sensorID, "bufferSize", len(buf), "gain", gain)

	duration := t2.Sub(t1).Milliseconds()
	item := api.SensorItem{
		SensorID:     sensorID,
		DataType:     dataType,
		SampleRate:   sampleRate,
		Channel:      cast.ToInt(channel),
		TimeFrom:     t1.UnixNano() / 1e3,
		TimeTo:       t2.UnixNano() / 1e3,
		TimeDuration: int64(duration),
		DataSize:     int64(len(data)),
		Query: api.SensorQuery{
			URL: "http://arc-storage/api/data/v1/history/arc?sensorid=" + sensorID + "&type=" +
				dataType + "&from=" + cast.ToString(t1.UnixNano()/1e3) + "&to=" + cast.ToString(t2.UnixNano()/1e3),
			Scheme: "http",
			Domain: "arc-storage",
			Port:   80,
			FullPath: "api/data/v1/history/arc?sensorid=" + sensorID + "&type=" + dataType + "&from=" +
				cast.ToString(t1.UnixNano()/1e3) + "&to=" + cast.ToString(t2.UnixNano()/1e3),
			Path:     "api/data/v1/history/arc",
			SensorID: sensorID,
			Type:     dataType,
			TimeFrom: t1.UnixNano() / 1e3,
			TimeTo:   t2.UnixNano() / 1e3,
		},
	}

	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: api.Success,
		Msg:  "OK",
		Data: item,
	},
	)
}

func (arc *ArcStorage) readDataFromCache(c echo.Context, sensorID, fileType string, t1, t2, tw time.Time) error {
	hexid, err := hex.DecodeString(sensorID)
	if err != nil {
		arc.logger.Errorw("hex.DecodeString", "sensorID", sensorID, "err", err)
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	uint64SensorID := util.BytesToUint64(hexid)
	arc.logger.Debugw("uint64SensorID", "Sensorid", sensorID, "hexid", uint64SensorID)

	var dataCache *cache.DataCacheRepo

	// 数据类型判断使用哪个dataCache
	if fileType == "audio" {
		dataCache = arc.audioCache
	}

	if dataCache == nil {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	// 从缓存中读取音振数据
	data, dataCount, err := dataCache.Search(
		arc.logger,
		uint64SensorID,
		t1,
		t2,
		arc.config.Work.FrameOffset,
	)
	if err != nil {
		arc.logger.Errorw("dataCache.Search", "sensorID", sensorID, "err", err)
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: api.Success,
		Msg:  "OK: " + cast.ToString(dataCount),
		Data: data,
	},
	)
}

func checkUploadParam(params url.Values) error {
	sensorID := params.Get("sensorid")
	if sensorID == "" {
		return fmt.Errorf("empty params sensorid")
	}
	from := params.Get("from")
	if from == "" {
		return fmt.Errorf("empty params from")
	}
	datatype := params.Get("type")
	if datatype == "" {
		return fmt.Errorf("empty params datatype")
	}
	sampleRate := params.Get("sample_rate")
	if sampleRate == "" {
		return fmt.Errorf("empty params samplerate")
	}
	channel := params.Get("channel")
	if channel == "" {
		return fmt.Errorf("empty params channel")
	}
	return nil
}

// checkUploadType 校验上传文件的数据类型,振动或音频
func checkUploadType(dataType string) (string, error) {
	if _, ok := UploadFileType[dataType]; ok {
		return UploadFileType[dataType], nil
	}

	return "", fmt.Errorf("wrong Data Type")
}
