package monitor

import (
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// PackageInterrupt 包数据中断
type PackageInterrupt struct {
	gaugeVec *prometheus.GaugeVec
}

// NewPackageInterrupt .
func NewPackageInterrupt() metric.PackageInterruptMetric {
	return &PackageInterrupt{
		gaugeVec: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      metric.MonitorDataInterrupt,
			Help:      "interrupt of package",
		}, []string{"sensorID"}),
	}
}

// Inc .
func (p *PackageInterrupt) Inc(args ...string) {
	p.gaugeVec.WithLabelValues(args...).Inc()
}

// Register .
func (p *PackageInterrupt) Register() error {
	if err := prometheus.Register(p.gaugeVec); err != nil {
		return errors.Wrap(err, "注册包中断统计监控")
	}
	return nil
}
