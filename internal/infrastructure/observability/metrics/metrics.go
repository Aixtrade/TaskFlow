package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TasksEnqueued = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "taskflow",
			Name:      "tasks_enqueued_total",
			Help:      "Total number of tasks enqueued",
		},
		[]string{"type", "queue"},
	)

	TasksProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "taskflow",
			Name:      "tasks_processed_total",
			Help:      "Total number of tasks processed",
		},
		[]string{"type", "status"},
	)

	TaskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "taskflow",
			Name:      "task_duration_seconds",
			Help:      "Task processing duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 15),
		},
		[]string{"type"},
	)

	TaskRetries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "taskflow",
			Name:      "task_retries_total",
			Help:      "Total number of task retries",
		},
		[]string{"type"},
	)

	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "taskflow",
			Name:      "queue_size",
			Help:      "Current queue size",
		},
		[]string{"queue", "state"},
	)

	ActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "taskflow",
			Name:      "active_workers",
			Help:      "Number of active workers",
		},
	)

	RedisConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "taskflow",
			Name:      "redis_connections",
			Help:      "Number of Redis connections",
		},
	)
)

func RecordTaskEnqueued(taskType, queue string) {
	TasksEnqueued.WithLabelValues(taskType, queue).Inc()
}

func RecordTaskProcessed(taskType, status string) {
	TasksProcessed.WithLabelValues(taskType, status).Inc()
}

func RecordTaskDuration(taskType string, duration float64) {
	TaskDuration.WithLabelValues(taskType).Observe(duration)
}

func RecordTaskRetry(taskType string) {
	TaskRetries.WithLabelValues(taskType).Inc()
}

func SetQueueSize(queue, state string, size float64) {
	QueueSize.WithLabelValues(queue, state).Set(size)
}

func SetActiveWorkers(count float64) {
	ActiveWorkers.Set(count)
}

func SetRedisConnections(count float64) {
	RedisConnections.Set(count)
}
