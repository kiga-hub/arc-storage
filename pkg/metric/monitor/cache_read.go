package monitor

import (
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// CacheRead 缓存读取
type CacheRead struct {
	counterVec *prometheus.GaugeVec
}

// NewCacheRead .
func NewCacheRead() metric.CacheReadMetric {
	return &CacheRead{
		counterVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      metric.MonitorCacheRead,
			Help:      "record cache read result",
		}, []string{"sensorID", "result"}),
	}
}

// Inc .
func (c *CacheRead) Inc(args ...string) {
	c.counterVec.WithLabelValues(args...).Inc()
}

// Register .
func (c *CacheRead) Register() error {
	if err := prometheus.Register(c.counterVec); err != nil {
		return errors.Wrap(err, "注册缓存读取监控")
	}
	return nil
}
