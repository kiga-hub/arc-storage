package arc_volume

import (
	"fmt"

	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ql = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metric.MonitorNamespace,
		Subsystem: metric.MonitorSubsystem,
		Name:      "filecache_queue_len",
	}, []string{"queue_id"})

	tc = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: metric.MonitorNamespace,
		Subsystem: metric.MonitorSubsystem,
		Name:      "filecache_task_cost",
		Objectives: map[float64]float64{
			0.5:  0.05,
			0.9:  0.01,
			0.99: 0.001,
		},
	}, []string{"task_type"})

	ttt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: metric.MonitorNamespace,
		Subsystem: metric.MonitorSubsystem,
		Name:      "filecache_task_timeout_times",
	}, []string{"task_type"})

	qin = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: metric.MonitorNamespace,
		Subsystem: metric.MonitorSubsystem,
		Name:      "filecache_queue_input_num",
	}, []string{"queue_id"})
)

func init() {
	prometheus.MustRegister(ql)
	prometheus.MustRegister(tc)
	prometheus.MustRegister(ttt)
	prometheus.MustRegister(qin)
}

// 队列长度监控
func setQueueLenMetric(queueID int, l float64) {
	ql.WithLabelValues(fmt.Sprintf("%d", queueID)).Set(l)
}

// 读、写任务执行耗时
func addTaskCostMetric(taskType string, seconds float64) {
	tc.WithLabelValues(taskType).Observe(seconds)
}

// 读、写任务超时次数
func addTaskTimeoutTimesMetric(taskType string) {
	ttt.WithLabelValues(taskType).Inc()
}

// 进入队列的任务数
func addQueueInputNumMetric(queueID int) {
	qin.WithLabelValues(fmt.Sprintf("%d", queueID)).Inc()
}
