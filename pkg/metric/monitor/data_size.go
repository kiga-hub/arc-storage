package monitor

import (
	"fmt"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// DataSize 存储数据大小
type DataSize struct {
	gaugeVec *prometheus.GaugeVec
}

// NewDataSize .
func NewDataSize(fileType string) metric.DataSizeMetric {
	return &DataSize{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      fmt.Sprintf("%s_%s", fileType, metric.MonitorDataSize),
			Help:      "data size of perform goroutines",
		}, []string{"sensorID"}),
	}
}

// Set .
func (d *DataSize) Set(val float64, args ...string) {
	d.gaugeVec.WithLabelValues(args...).Set(val)
}

// Register .
func (d *DataSize) Register() error {
	if err := prometheus.Register(d.gaugeVec); err != nil {
		return errors.Wrap(err, "注册存储数据大小监控")
	}
	return nil
}
