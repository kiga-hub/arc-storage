package monitor

import (
	"fmt"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// ConsumingTime 消耗时间
type ConsumingTime struct {
	gaugeVec *prometheus.GaugeVec
}

// NewConsumingTime .
func NewConsumingTime(fileType string) metric.ConsumingTimeMetric {
	return &ConsumingTime{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      fmt.Sprintf("%s_%s", fileType, metric.MonitorFileCacheConsumingTime),
			Help:      "consuming time of perform goroutines",
		}, []string{"sensorID"}),
	}
}

// Set .
func (c *ConsumingTime) Set(val float64, args ...string) {
	c.gaugeVec.WithLabelValues(args...).Set(val)
}

// Register .
func (c *ConsumingTime) Register() error {
	if err := prometheus.Register(c.gaugeVec); err != nil {
		return errors.Wrap(err, "注册消耗时间监控")
	}
	return nil
}
