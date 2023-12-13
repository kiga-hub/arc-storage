package monitor

import (
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

// GRPC GRPC数据量
type GRPC struct {
	counterVec *prometheus.CounterVec
}

// NewGRPC .
func NewGRPC() metric.GRPCMetric {
	return &GRPC{
		counterVec: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: metric.MonitorNamespace,
			Subsystem: metric.MonitorSubsystem,
			Name:      metric.MonitorGRPCBytes,
			Help:      "record gRPC received data size",
		}, []string{"sensorID"}),
	}
}

// Add .
func (g *GRPC) Add(val float64, args ...string) {
	g.counterVec.WithLabelValues(args...).Add(val)
}

// Register .
func (g *GRPC) Register() error {
	if err := prometheus.Register(g.counterVec); err != nil {
		return errors.Wrap(err, "注册GRPC数据量监控")
	}
	return nil
}
