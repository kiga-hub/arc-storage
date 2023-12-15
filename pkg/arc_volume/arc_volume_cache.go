package arc_volume

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/config"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/kiga-hub/arc-storage/pkg/metric/monitor"
	"github.com/kiga-hub/arc-storage/pkg/util"
	"github.com/kiga-hub/arc/logging"
	"github.com/kiga-hub/arc/utils"

	"github.com/spf13/cast"
)

var (
	// DataTypeMap Data type, description conversion
	DataTypeMap = map[string]string{
		"Arc": "TypeArc",
	}
)

// ArcVolumeCache -
type ArcVolumeCache struct {
	logger        logging.ILogger
	config        *config.ArcConfig
	DataCache     *sync.Map
	queue         *Queue
	exportMetrics *metric.FileCacheMonitor
}

// ArcVolume -
type ArcVolume struct {
	SensorID        string
	Buffer          *bytes.Buffer
	Dir             string
	CreateTime      time.Time
	SaveTime        time.Time
	LastTimestamp   time.Time
	FirmWare        string
	StatusOfStorage string
	MinuteStr       string
	Version         string
	Type            string
	SampleRate      float64
	DynamicRange    byte
	Resolution      byte
	Channel         byte
	Bits            byte
}

// NewArcVolumeCache -
func NewArcVolumeCache(logger logging.ILogger, config *config.ArcConfig, fileType string) (*ArcVolumeCache, error) {
	// Initialize the metric collection module
	ct := monitor.NewConsumingTime(fileType)
	ds := monitor.NewDataSize(fileType)
	dwe := monitor.NewDiskWriteError(fileType)
	fc := monitor.NewFileRead(fileType)
	m, err := metric.NewFileCacheMonitor(ct, ds, dwe, fc)
	if err != nil {
		return nil, err
	}
	return &ArcVolumeCache{
		logger:        logger,
		config:        config,
		DataCache:     &sync.Map{},
		exportMetrics: m,
		queue:         NewQueue(logger, config.Work.ArcVolumeQueueLen, config.Work.ArcVolumeQueueNum),
	}, nil
}

// SafeClose -close Data process channel
func (b *ArcVolumeCache) SafeClose() {
	b.queue.Close()
}

// ReadDataByQueue -
func (b *ArcVolumeCache) ReadDataByQueue(sensorID, fileType string, t1, t2 time.Time) ([]byte, *utils.ResponseV2) {

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(b.config.Work.ArcVolumeQueueReadTimeoutSeconds)*time.Second)
	defer cancel()

	// 走队列的方式查询
	readTask := createReadTask(timeoutCtx, sensorID, b, sensorID, fileType, t1, t2)
	b.queue.DoTask(readTask)

	return readTask.data, readTask.err
}
func (b *ArcVolumeCache) readDataLogic(ctx context.Context, sensorID, fileType string, t1, t2 time.Time) ([]byte, *utils.ResponseV2) {

	channelmulbits := 2
	// 获取以天为单位的时间范围
	daysdiffer, count, err := util.GetDaysDiffer(t1.UTC().Format("2006-01-02 15:04:05"), t2.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		b.logger.Errorw("getDaysDiffer", "err", err)
		return nil, &utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest),
		}
	}

	b.logger.Debugw("util.GetDaysDiffer", "daysdiffer", daysdiffer, "count", count)

	start := time.Now()
	defer func() {
		b.logger.Debugf("get data spend %s\n", time.Since(start).String())
	}()

	if isCtxTimeout(ctx) {
		return nil, &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.ErrHandlerTimeout.Error(),
		}
	}

	// 获取大文件列表
	bigfilelists := make(map[string]string, count) // map[createtime]fullpath

	for _, day := range daysdiffer {
		path := b.config.Work.DataPath + "/" + sensorID + "/" + day + "/" + fileType
		err := util.GetBigFileLists(path, bigfilelists, "")
		if err != nil {
			b.logger.Debugw("GetBigFileList", "path", path, "bigFileList", bigfilelists, "err", err)
			continue
		}
	}
	b.logger.Debugw("bigfilelists", "bigfilelists", bigfilelists)

	var timestamplist []string
	var starttimestamps []string

	for key := range bigfilelists {
		timestamplist = append(timestamplist, key)
	}
	sort.Strings(timestamplist)
	b.logger.Debugw("sort.Strings", "(timestamplist)", timestamplist)

	if isCtxTimeout(ctx) {
		return nil, &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.ErrHandlerTimeout.Error(),
		}
	}

	for index, creattime := range timestamplist {
		begin, end, _, _, _, err := util.GetTimeRangeFromFileName(bigfilelists[creattime])

		if err != nil {
			b.logger.Errorw("GetTimeRangeFromFileName", "fileName", bigfilelists[creattime], "err", err)
			continue
		}
		// 开始时间大于文件保存结束时间，忽略
		if end.Before(t1) {
			continue
		}
		// 匹配到的第一个文件，需要判断时间范围，大文件时间精度为微妙级别，查询时间为到秒级别。
		if t1.Before(begin) && index == 0 {
			b.logger.Debugw("t1.Before(begin)", "index", index)
			continue
		}
		// 结束时间小于文件创建时间，退出
		if t2.Before(begin) {
			b.logger.Debugw("compare timestamp", "t2", t2, "begin", begin)
			continue
		}
		starttimestamps = append(starttimestamps, creattime)
	}

	b.logger.Debugw("sort starttimestamps", "starttimestamps", starttimestamps)
	if len(starttimestamps) < 1 {
		b.logger.Debugw("check timestamp info", "timestamplist", timestamplist, "bigfilelists", bigfilelists)
		return nil, &utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound),
		}
	}

	if isCtxTimeout(ctx) {
		return nil, &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.ErrHandlerTimeout.Error(),
		}
	}

	// 文件列表排序
	sort.Strings(starttimestamps)
	b.logger.Debugw("sort starttimestamps", "starttimestamps", starttimestamps)

	var filepathlist []string
	filescount := 0
	isSampleRateChanged := false

	sampleratestr := ""

	// 遍历当前文件夹下得目录
	for _, v := range starttimestamps {
		fullpath := bigfilelists[v][strings.LastIndex(bigfilelists[v], "/")+1:]
		filepathlist = append(filepathlist, bigfilelists[v])
		_, _, _, _, sam, err := util.GetTimeRangeFromFileName(fullpath)
		if err != nil {
			b.logger.Errorw("getTimeRangeFromFileName", "err", err)
			continue
		}
		if sampleratestr == "" {
			sampleratestr = sam
		}
		if sampleratestr != sam {
			isSampleRateChanged = true
		}
		filescount++
	}

	if isCtxTimeout(ctx) {
		return nil, &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.ErrHandlerTimeout.Error(),
		}
	}

	b.logger.Debugw("filepathlist info", "filepathlist", filepathlist)
	// 查询文件数为多个或没有，客户端需要访问search
	if len(filepathlist) > 0 && isSampleRateChanged {
		b.exportMetrics.SetFileReadValues(sensorID, metric.MonitorFailed)
		return nil, &utils.ResponseV2{
			Code: http.StatusMultipleChoices,
			Msg:  http.StatusText(http.StatusMultipleChoices),
		}
	}
	// 查询时间大于10分钟，拒绝访问
	if t2.Sub(t1).Minutes() > 10.0 {
		b.exportMetrics.SetFileReadValues(sensorID, metric.MonitorSuccess)
		return nil, &utils.ResponseV2{
			Code: http.StatusMultipleChoices,
			Msg:  http.StatusText(http.StatusMultipleChoices),
		}
	}

	// 获取数据文件大小
	finalsize := cast.ToInt(t2.Sub(t1).Seconds()) * cast.ToInt(sampleratestr) * channelmulbits
	b.logger.Debugw("compare t1 and t2", "t1", t1, "t2", t2, "filescount", filescount)

	if t2.Sub(t1).Minutes() <= 10.0 && filescount != 0 && !isSampleRateChanged {
		var Response bytes.Buffer
		// var buffer bytes.Buffer

		for _, v := range filepathlist {

			if isCtxTimeout(ctx) {
				return nil, &utils.ResponseV2{
					Code: http.StatusGatewayTimeout,
					Msg:  http.ErrHandlerTimeout.Error(),
				}
			}

			filename := v[strings.LastIndex(v, "/")+1:]
			startdate, enddate, id, _, _, err := util.GetTimeRangeFromFileName(filename)

			if err != nil {
				b.logger.Errorw("getTimeRangeFromFileName", "err", err)
				continue
			}

			fi, err := os.Stat(v)
			if os.IsNotExist(err) {
				b.logger.Debugw("file is not exist", "err", err, "filepath", v)
				continue
			}

			b.logger.Debugw("date info", "fullpath", v, "startdate", startdate, "enddate", enddate, "sensorid", id)
			// 只有一个文件符合条件，且查询结束时间早于文件保存时间
			if len(filepathlist) == 1 && !t2.After(enddate) {

				// 从开始时间进行切分
				cutduration := t1.Sub(startdate).Seconds()
				// 切分时间长度
				cutindex := cutduration * float64(cast.ToInt(sampleratestr))
				b.logger.Infow("cutInfo", "cutduration", cutduration, "cut", cast.ToInt64(cutindex)*cast.ToInt64(channelmulbits)+44)

				if isCtxTimeout(ctx) {
					return nil, &utils.ResponseV2{
						Code: http.StatusGatewayTimeout,
						Msg:  http.ErrHandlerTimeout.Error(),
					}
				}

				// 查找数据, 写入Bufffer并返回
				data, err := util.SeekBigFileHeader(v, cast.ToInt64(cutindex)*cast.ToInt64(channelmulbits)+44, cast.ToInt64(finalsize))
				if err != nil {
					b.logger.Errorw("SeekBigFileHeader", "err", err,
						"t1", t1, "t2", t2, "startdate", startdate, "enddate", enddate, "sensorid", id,
						"finalSize", cast.ToInt64(finalsize), "fileSize", fi.Size())
					continue
				}

				if isCtxTimeout(ctx) {
					return nil, &utils.ResponseV2{
						Code: http.StatusGatewayTimeout,
						Msg:  http.ErrHandlerTimeout.Error(),
					}
				}

				Response.Write(data)

				b.logger.Debugw("Response info", "sensorid", sensorID, "Response size", Response.Len())
				b.exportMetrics.SetFileReadValues(sensorID, metric.MonitorSuccess)

				return Response.Bytes(), nil
			}

			// 返回300, 需要先调用search确定查询时间范围
			return nil, &utils.ResponseV2{
				Code: http.StatusMultipleChoices,
				Msg:  http.StatusText(http.StatusMultipleChoices),
			}
		}
	}

	return nil, &utils.ResponseV2{
		Code: http.StatusNotFound,
		Msg:  http.StatusText(http.StatusNotFound),
	}
}

// Update 更新文件存储信息
func (bf *ArcVolume) Update(t time.Time) {
	// reset buffer
	buffer := bytes.NewBuffer([]byte{})
	buffer.Grow(2048)

	// 日期变更，更新存储路径
	minuteStr := t.UTC().Format("200601021504")
	bf.Buffer.Reset()

	bf.MinuteStr = minuteStr
	bf.Buffer = buffer
	bf.Channel = 0
	bf.SampleRate = 0
}

// PreWriteToFileCache 深拷贝数据,准备写入FileCache
func (b *ArcVolumeCache) PreWriteToFileCache(bf *ArcVolume, t time.Time, secondHalfSize int) error {
	if bf.Buffer.Len()-secondHalfSize <= 0 {
		b.logger.Debug("PreWriteToFileCache", "bufferSize", bf.Buffer.Len(), "secondHalfSize", secondHalfSize, "sensorID", bf.SensorID)
	}
	// deep copy the data
	buffer := bytes.NewBuffer([]byte{})
	if bf.Buffer.Len() > secondHalfSize {
		buffer.Grow(bf.Buffer.Len() - secondHalfSize)
		buffer.Write(bf.Buffer.Bytes()[:bf.Buffer.Len()-secondHalfSize])
	} else {
		buffer.Grow(bf.Buffer.Len())
		buffer.Write(bf.Buffer.Bytes())
	}

	b.logger.Debugw("PreWriteToFileCache", "type", bf.Type, "buffer_len", bf.Buffer.Len(), "sampleRate", bf.SampleRate, "secondHalfSize", secondHalfSize)
	// 保存文件时确定写入文件路径
	dateFolderName := bf.CreateTime.Format("20060102")
	dir := bf.Dir + "/" + bf.SensorID + "/" + dateFolderName + "/" + DataTypeMap[bf.Type]
	data := &ArcVolume{
		CreateTime:      bf.CreateTime,
		SaveTime:        t,
		Dir:             dir,
		FirmWare:        bf.FirmWare,
		SensorID:        bf.SensorID,
		Buffer:          buffer,
		Channel:         bf.Channel,
		DynamicRange:    bf.DynamicRange,
		Resolution:      bf.Resolution,
		SampleRate:      bf.SampleRate,
		Bits:            bf.Bits,
		MinuteStr:       bf.MinuteStr,
		Version:         bf.Version,
		Type:            bf.Type,
		StatusOfStorage: bf.StatusOfStorage,
	}
	if err := b.WriteDataByQueue(data); err != nil {
		b.logger.Errorw("SetRowDataByTask tmp fail", "type", bf.Type, "buffer_len", data.Buffer.Len(), "err", err)
		return err
	}

	return nil
}

// WriteDataByQueue -
func (b *ArcVolumeCache) WriteDataByQueue(data *ArcVolume) error {

	// 走队列的方式查询
	writeTask := createWriteTask(context.Background(), data.SensorID, b, data)
	b.queue.DoTask(writeTask)

	return writeTask.err
}
func (b *ArcVolumeCache) writeDataLogic(ctx context.Context, cc *ArcVolume) error {
	dataSize := cc.Buffer.Len()

	if dataSize < 1 {
		return nil
	}

	startTime := time.Now().UTC()

	filepath := ""
	isFirst := false

	// 获取最近一次创建文件名
	lastcreatefilename := util.GetLastCreateFile(cc.Dir)
	if lastcreatefilename == "" {
		// 当前目录下未生成文件
		filepath = cc.Dir + "/" + cc.SensorID + "_" + cc.Type + "_" + util.TimeStringReplace(cc.CreateTime) + "_" + util.TimeStringReplace(cc.SaveTime) +
			"_" + cc.StatusOfStorage + "_" + cast.ToString(cc.SampleRate) + "_" + cc.FirmWare + "_" + cc.Version + ".wav"

		b.logger.Debugw("no files are generated in the current directory", "filePath", filepath)
		isFirst = true
	} else {
		// 比较是否是创建新文件
		filepath = cc.Dir + "/" + lastcreatefilename
		b.logger.Debugw("determine to create a new file or not", "filepath", filepath)
		isFirst = isFirstCreateFile(filepath, cc.StatusOfStorage, cc.SensorID, cast.ToString(cc.SampleRate), cc.CreateTime)
	}

	dataToStore := make([]byte, dataSize)
	copy(dataToStore, cc.Buffer.Bytes())

	if isFirst {
		// 修改保存时间
		duration := float64(dataSize) / cc.SampleRate / float64(2*int(cc.Channel)) * 1e9
		//dump.Dump(cc)
		filepath = cc.Dir + "/" + cc.SensorID + "_" + cc.Type + "_" + util.TimeStringReplace(cc.CreateTime) + "_" +
			util.TimeStringReplace(cc.CreateTime.Add(time.Nanosecond*time.Duration(duration))) + "_" + cc.StatusOfStorage +
			"_" + cast.ToString(cc.SampleRate) + "_" + cc.FirmWare + "_" + cc.Version + ".wav"

		b.logger.Debugw("data appending is not required, create a new file", "filePath", filepath, "the file name was created last time",
			lastcreatefilename, "new data creation time", cc.CreateTime, "channel", int(cc.Channel), "sampleRate", cast.ToInt(cc.SampleRate), "dataSize", dataSize)

		if err := createnewFile(ctx, b.logger, filepath, dataToStore, int(cc.Channel), cast.ToInt(cc.SampleRate)); err != nil {
			b.exportMetrics.SetDiskWriteErrorLabelValues(cc.SensorID)
			b.logger.Debugw("CreateNewFile", "filePath", filepath, "err", err)
			return err
		}
	} else {
		// 拼接
		b.logger.Debugw("append", "filePath", filepath)
		if err := append2ExistFile(ctx, b.logger, filepath, dataToStore, cc.CreateTime, int(cc.Channel), int(cc.SampleRate)); err != nil {
			b.exportMetrics.SetDiskWriteErrorLabelValues(cc.SensorID)
			b.logger.Debugw("Append2ExistFile", "filePath", filepath, "channel", cc.Channel, "SampleRate", cc.SampleRate, "err", err)
			return err
		}
	}

	b.exportMetrics.SetConsumingTimeLabelValues(cc.SensorID, startTime)
	b.exportMetrics.SetDataSizeLabelValues(cc.SensorID, float64(dataSize))

	return nil
}

// renameBigFile -
func renameBigFile(ctx context.Context, logger logging.ILogger, samplerate float64, filepath string, pointsize int) error {

	start, end, _, _, _, err := util.GetTimeRangeFromFileName(filepath)
	if err != nil {
		return fmt.Errorf("getTimeRangeFromFileName: %v", err)
	}

	fi, err := os.Stat(filepath)
	if err != nil {
		return err
	}

	duration := float64(fi.Size()-44) / samplerate / float64(pointsize) * 1e9
	new := strings.Replace(filepath, util.TimeStringReplace(end), util.TimeStringReplace(start.Add(time.Nanosecond*time.Duration(duration))), -1)

	if err := os.Rename(filepath, new); err != nil {
		return err
	}

	return nil
}

// isFirstCreateFile 判断是否是第一次创建文件
func isFirstCreateFile(filepath, storagestatus, sensorid, samplerate string, createtime time.Time) bool {
	start, _, id, _, samp, err := util.GetTimeRangeFromFileName(filepath)
	if err != nil {
		return true
	}

	// 创建时间，sensorid,采样率是否一致
	// 此处使用UnixNano进行时间比较，createtime 是由time.Unix(0, f.Timestamp*1e3) 转换，而start是由 time.Parse()进行转换，两个time.Time类型进行比较会不一致
	if createtime.UnixNano() == start.UnixNano() && sensorid == id && samplerate == samp {
		return false
	}

	// createtime = {%!t(uint64=763726000) %!t(int64=63795626414) %!t(*time.Location=&{Local [{UTC 0 false}] [{-576460752303423488 0 false false}] UTC0 9223372036854775807 9223372036854775807 0xc000137bc0})}
	// start      = {%!t(uint64=763726000) %!t(int64=63795626414) %!t(*time.Location=<nil>)}
	return true
}

// createnewFile 创建新文件
func createnewFile(ctx context.Context, logger logging.ILogger, filepath string, data []byte, channel, samplerte int) error {

	err := os.MkdirAll(path.Dir(filepath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("MKdirAll: %v", err)
	}

	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("file create error: %v", err)
	}
	defer file.Close()
	return nil
}

// append2ExistFile 文件追加
func append2ExistFile(ctx context.Context, logger logging.ILogger, filepath string, data []byte, createtime time.Time, channel, samplerate int) error {

	filename, err := util.GetFileListNamebyCreateTime(path.Dir(filepath), util.TimeStringReplace(createtime))
	if err != nil {
		return fmt.Errorf("GetFileListNamebyCreateTime: %v", err)
	}

	if util.CheckFileExists(filename) {

		if err := appendBigFileData(ctx, logger, filename, data); err != nil {
			return fmt.Errorf("appendBigFileData: %v", err)
		}

		// 修改保存时间，文件重命名
		if err := renameBigFile(ctx, logger, float64(samplerate), filename, 2*channel); err != nil {
			return fmt.Errorf("renameBigFile: %v", err)
		}
	} else {

		// 文件不存在，新建文件
		if err := createnewFile(ctx, logger, filepath, data, channel, samplerate); err != nil {
			return fmt.Errorf("createnewFile: %v", err)
		}
	}

	return nil
}

func appendBigFileData(ctx context.Context, logger logging.ILogger, filename string, data []byte) error {

	// 文件存在，追加写入
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil { //os.O_TRUNC
		return fmt.Errorf("OpenFile: %v", err)
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("file write error: %v", err)
	}

	return nil
}

func isCtxTimeout(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
