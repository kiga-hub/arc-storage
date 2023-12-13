package monitor

import (
	"fmt"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// FileRead 文件读取
type FileRead struct {
	counterVec *prometheus.GaugeVec
}

// NewFileRead .
func NewFileRead(fileType string) metric.FileReadMetric {
	return &FileRead{
		counterVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      fmt.Sprintf("%s_%s", fileType, metric.MonitorFileRead),
			Help:      "record file read result",
		}, []string{"sensorID", "result"}),
	}
}

// Inc .
func (c *FileRead) Inc(args ...string) {
	c.counterVec.WithLabelValues(args...).Inc()
}

// Register .
func (c *FileRead) Register() error {
	if err := prometheus.Register(c.counterVec); err != nil {
		return errors.Wrap(err, "注册缓存读取监控")
	}
	return nil
}
