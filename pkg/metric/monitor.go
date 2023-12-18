package metric

// GRPCMetric GRPC数据量度量指标
type GRPCMetric interface {
	Add(val float64, args ...string) // 自增给定值
	Register() error                 // 注册
}

// CacheReadMetric 读取缓存度量指标
type CacheReadMetric interface {
	Inc(args ...string) // 自增读取缓存
	Register() error    // 注册
}

// FileReadMetric 读取文件度量指标
type FileReadMetric interface {
	Inc(args ...string) // 自增读取文件
	Register() error    // 注册
}

// ConsumingTimeMetric 储耗时度量指标
type ConsumingTimeMetric interface {
	Set(val float64, args ...string) // 设置给定值
	Register() error                 // 注册
}

// DataSizeMetric 存储数据大小度量指标
type DataSizeMetric interface {
	Set(val float64, args ...string) // 设置给定值
	Register() error                 // 注册
}

// DiskWriteErrorMetric 磁盘写入错误指标
type DiskWriteErrorMetric interface {
	Inc(args ...string) // 自增1
	Register() error    // 注册
}
