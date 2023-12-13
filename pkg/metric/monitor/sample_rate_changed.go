package monitor

import (
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// SampleRateChanged 采样率变化
type SampleRateChanged struct {
	gaugeVec *prometheus.GaugeVec
}

// NewSampleRateChanged .
func NewSampleRateChanged() metric.SampleRateChangedMetric {
	return &SampleRateChanged{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      metric.MonitorSampleRateChanged,
			Help:      "sampleRate changed",
		}, []string{"sensorID"}),
	}
}

// Inc .
func (s *SampleRateChanged) Inc(args ...string) {
	s.gaugeVec.WithLabelValues(args...).Inc()
}

// Register .
func (s *SampleRateChanged) Register() error {
	if err := prometheus.Register(s.gaugeVec); err != nil {
		return errors.Wrap(err, "注册采样率变化监控")
	}
	return nil
}
