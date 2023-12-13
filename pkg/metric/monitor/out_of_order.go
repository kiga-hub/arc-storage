package monitor

import (
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// OutOfOrder 数据包乱序
type OutOfOrder struct {
	gaugeVec *prometheus.GaugeVec
}

// NewOutOfOrder .
func NewOutOfOrder() metric.OutOfOrderMetric {
	return &OutOfOrder{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      metric.MonitorOutOfOrder,
			Help:      "package is out of order",
		}, []string{"sensorID"}),
	}
}

// Inc .
func (s *OutOfOrder) Inc(args ...string) {
	s.gaugeVec.WithLabelValues(args...).Inc()
}

// Register .
func (s *OutOfOrder) Register() error {
	if err := prometheus.Register(s.gaugeVec); err != nil {
		return errors.Wrap(err, "注册数据包乱序监控")
	}
	return nil
}
