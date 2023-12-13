package monitor

import (
	"fmt"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// DiskWriteError 磁盘写入错误
type DiskWriteError struct {
	gaugeVec *prometheus.GaugeVec
}

// NewDiskWriteError .
func NewDiskWriteError(fileType string) metric.DiskWriteErrorMetric {
	return &DiskWriteError{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      fmt.Sprintf("%s_%s", fileType, metric.MonitorDiskWriteErr),
			Help:      "error count of disk write",
		}, []string{"sensorID"}),
	}
}

// Inc .
func (d *DiskWriteError) Inc(args ...string) {
	d.gaugeVec.WithLabelValues(args...).Inc()
}

// Register .
func (d *DiskWriteError) Register() error {
	if err := prometheus.Register(d.gaugeVec); err != nil {
		return errors.Wrap(err, "注册磁盘写入错误统计监控")
	}
	return nil
}
