package metric

import (
	"time"

	"github.com/pkg/errors"
)

const (
	// MonitorSuccess 统计成功项
	MonitorSuccess = "success"
	// MonitorFailed 统计失败项
	MonitorFailed = "failed"
)

const (
	// MonitorNamespace .
	MonitorNamespace = "Arc"
	// MonitorSubsystem .
	MonitorSubsystem = "Storage"

	// MonitorCacheRead 缓存读取
	MonitorCacheRead = "cache_read"
	// MonitorFileRead 文件读取
	MonitorFileRead = "file_read"
	// MonitorFileCacheConsumingTime channel耗时
	MonitorFileCacheConsumingTime = "fileCache_consuming_time"
	// MonitorDataSize 数据存储大小
	MonitorDataSize = "data_size"
	// MonitorDiskWriteErr 磁盘读写失败
	MonitorDiskWriteErr = "disk_write_err"
	// MonitorGRPCBytes GRPC接受数据量
	MonitorGRPCBytes = "gRPC_bytes"
	// MonitorOutOfOrder 数据包乱序
	MonitorOutOfOrder = "out_of_order"
	// MonitorDataInterrupt 数据中断
	MonitorDataInterrupt = "data_interrupt"
	// MonitorSampleRateChanged 采样率改变
	MonitorSampleRateChanged = "sampleRate_changed"
)

// HandlerMonitor handler监控
type HandlerMonitor struct {
	packageInterruptMetric  PackageInterruptMetric  // 包中断指标
	sampleRateChangedMetric SampleRateChangedMetric // 采样率变化指标
	gRPCMetric              GRPCMetric              // GRPC数据量指标
	cacheMetric             CacheReadMetric         // 缓存读取指标
	outOfOrderMetric        OutOfOrderMetric        // 数据包乱序指标
}

// NewHandlerMonitor .
func NewHandlerMonitor(pi PackageInterruptMetric, src SampleRateChangedMetric, grpc GRPCMetric, order OutOfOrderMetric, cacheRead CacheReadMetric) (*HandlerMonitor, error) {
	h := &HandlerMonitor{
		packageInterruptMetric:  pi,
		sampleRateChangedMetric: src,
		gRPCMetric:              grpc,
		outOfOrderMetric:        order,
		cacheMetric:             cacheRead,
	}
	if err := h.registerHandlerMonitor(); err != nil {
		return nil, errors.Wrap(err, "注册handler监控服务")
	}

	return h, nil
}

// registerHandlerMonitor 注册handler监控
func (h *HandlerMonitor) registerHandlerMonitor() error {
	if err := h.packageInterruptMetric.Register(); err != nil {
		return err
	}
	if err := h.sampleRateChangedMetric.Register(); err != nil {
		return err
	}
	if err := h.outOfOrderMetric.Register(); err != nil {
		return err
	}
	if err := h.cacheMetric.Register(); err != nil {
		return err
	}
	return h.gRPCMetric.Register()
}

// SetGRPCLabelValues 采集gRPC数据传输量
func (h *HandlerMonitor) SetGRPCLabelValues(sensorID string, size float64) {
	h.gRPCMetric.Add(size, sensorID)
}

// SetCacheReadValues api获取缓存统计
func (h *HandlerMonitor) SetCacheReadValues(sensorID, result string) {
	h.cacheMetric.Inc(sensorID, result)
}

// SetInterruptLabelValues 采集数据包中断次数
func (h *HandlerMonitor) SetInterruptLabelValues(sensorID string) {
	h.packageInterruptMetric.Inc(sensorID)
}

// SetSampleRateLabelValues 采集采样率变化次数
func (h *HandlerMonitor) SetSampleRateLabelValues(sensorID string) {
	h.sampleRateChangedMetric.Inc(sensorID)
}

// SetOutOfOrderLabelValues 数据包乱序次数
func (h *HandlerMonitor) SetOutOfOrderLabelValues(sensorID string) {
	h.outOfOrderMetric.Inc(sensorID)
}

// FileCacheMonitor 文件缓存监控
type FileCacheMonitor struct {
	consumingTimeMetric  ConsumingTimeMetric  // 存储消耗指标
	dataSizeMetric       DataSizeMetric       // 存储数据大小指标
	diskWriteErrorMetric DiskWriteErrorMetric // 磁盘写入指标
	fileReadMetric       FileReadMetric       // 文件读取指标
}

// NewFileCacheMonitor .
func NewFileCacheMonitor(ct ConsumingTimeMetric, ds DataSizeMetric, dwe DiskWriteErrorMetric, fc FileReadMetric) (*FileCacheMonitor, error) {
	f := &FileCacheMonitor{
		consumingTimeMetric:  ct,
		dataSizeMetric:       ds,
		diskWriteErrorMetric: dwe,
		fileReadMetric:       fc,
	}
	if err := f.registerFileCacheMonitor(); err != nil {
		return nil, errors.Wrap(err, "注册文件缓存监控服务")
	}
	return f, nil
}

// registerFileCacheMonitor 注册文件缓存监控
func (f *FileCacheMonitor) registerFileCacheMonitor() error {
	if err := f.consumingTimeMetric.Register(); err != nil {
		return err
	}
	if err := f.dataSizeMetric.Register(); err != nil {
		return err
	}
	if err := f.fileReadMetric.Register(); err != nil {
		return err
	}
	return f.diskWriteErrorMetric.Register()
}

// SetConsumingTimeLabelValues 设置channel耗时
func (f *FileCacheMonitor) SetConsumingTimeLabelValues(sensorID string, startTime time.Time) {
	// 记录每个通道处理数据所需时间
	consuming := float64((time.Now().UTC().Sub(startTime)).Milliseconds())
	f.consumingTimeMetric.Set(consuming, sensorID)
}

// SetDataSizeLabelValues  设置数据存储大小
func (f *FileCacheMonitor) SetDataSizeLabelValues(sensorID string, dataSize float64) {
	// 记录每次落盘时数据大小
	f.dataSizeMetric.Set(dataSize, sensorID)
}

// SetDiskWriteErrorLabelValues 设置写入磁盘错误计数
func (f *FileCacheMonitor) SetDiskWriteErrorLabelValues(sensorID string) {
	f.diskWriteErrorMetric.Inc(sensorID)
}

// SetFileReadValues 设置文件读取计数
func (f *FileCacheMonitor) SetFileReadValues(sensorID, result string) {
	f.fileReadMetric.Inc(sensorID, result)
}
