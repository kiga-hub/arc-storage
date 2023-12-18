package arc_volume

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"sync"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/config"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/kiga-hub/arc-storage/pkg/metric/monitor"
	"github.com/kiga-hub/arc-storage/pkg/util"
	"github.com/kiga-hub/arc/logging"
	"github.com/kiga-hub/arc/utils"
)

var (
	// DataTypeMap Data type, description conversion
	DataTypeMap = map[string]string{
		"Arc": "TypeArc",
	}
)

const (
	// FileType
	FileType = ".arc"
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
	SensorID      string
	Buffer        *bytes.Buffer
	Dir           string
	CreateTime    time.Time
	SaveTime      time.Time
	LastTimestamp time.Time
	MinuteStr     string
	Type          string
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

	starttimestamps = append(starttimestamps, timestamplist...)

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

	// 遍历当前文件夹下得目录
	filescount = len(starttimestamps)

	if isCtxTimeout(ctx) {
		return nil, &utils.ResponseV2{
			Code: http.StatusGatewayTimeout,
			Msg:  http.ErrHandlerTimeout.Error(),
		}
	}

	b.logger.Debugw("filepathlist info", "filepathlist", filepathlist)

	// 查询时间大于10分钟，拒绝访问
	if t2.Sub(t1).Minutes() > 10.0 {
		b.exportMetrics.SetFileReadValues(sensorID, metric.MonitorSuccess)
		return nil, &utils.ResponseV2{
			Code: http.StatusMultipleChoices,
			Msg:  http.StatusText(http.StatusMultipleChoices),
		}
	}

	if t2.Sub(t1).Minutes() <= 10.0 && filescount != 0 {
		var Response bytes.Buffer
		// var buffer bytes.Buffer
		for _, v := range filepathlist {
			if isCtxTimeout(ctx) {
				return nil, &utils.ResponseV2{
					Code: http.StatusGatewayTimeout,
					Msg:  http.ErrHandlerTimeout.Error(),
				}
			}

			if err != nil {
				b.logger.Errorw("getTimeRangeFromFileName", "err", err)
				continue
			}

			fi, err := os.Stat(v)
			if os.IsNotExist(err) {
				b.logger.Debugw("file is not exist", "err", err, "filepath", v)
				continue
			}

			seekSize := 1024
			// find data ,write to buffer and return
			data, err := util.SeekBigFileHeader(v, 0, int64(seekSize))
			if err != nil {
				b.logger.Errorw("SeekBigFileHeader", "err", err, "t1", t1, "t2", t2, "startdate", "fileSize", fi.Size())
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

	b.logger.Debugw("PreWriteToFileCache", "type", bf.Type, "buffer_len", bf.Buffer.Len(), "secondHalfSize", secondHalfSize)
	// 保存文件时确定写入文件路径
	dateFolderName := bf.CreateTime.Format("20060102")
	dir := bf.Dir + "/" + bf.SensorID + "/" + dateFolderName + "/" + DataTypeMap[bf.Type]
	data := &ArcVolume{
		CreateTime: bf.CreateTime,
		SaveTime:   t,
		Dir:        dir,
		SensorID:   bf.SensorID,
		Buffer:     buffer,
		MinuteStr:  bf.MinuteStr,
		Type:       bf.Type,
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

	filepath = cc.Dir + "/" + cc.SensorID + "_" + cc.Type + "_" + util.TimeStringReplace(cc.CreateTime) + "_" + util.TimeStringReplace(cc.SaveTime) + FileType
	b.logger.Debugw("no files are generated in the current directory", "filePath", filepath)

	dataToStore := make([]byte, dataSize)
	copy(dataToStore, cc.Buffer.Bytes())

	if isFirst {
		//dump.Dump(cc)
		filepath = cc.Dir + "/" + cc.SensorID + "_" + cc.Type + "_" + util.TimeStringReplace(cc.CreateTime) + "_" + util.TimeStringReplace(cc.CreateTime) + FileType

		b.logger.Debugw("data appending is not required, create a new file", "filePath", filepath, "new data creation time", cc.CreateTime, "dataSize", dataSize)

		if err := createnewFile(ctx, b.logger, filepath, dataToStore); err != nil {
			b.exportMetrics.SetDiskWriteErrorLabelValues(cc.SensorID)
			b.logger.Debugw("CreateNewFile", "filePath", filepath, "err", err)
			return err
		}
	}

	b.exportMetrics.SetConsumingTimeLabelValues(cc.SensorID, startTime)
	b.exportMetrics.SetDataSizeLabelValues(cc.SensorID, float64(dataSize))

	return nil
}

// createnewFile 创建新文件
func createnewFile(ctx context.Context, logger logging.ILogger, filepath string, data []byte) error {

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
