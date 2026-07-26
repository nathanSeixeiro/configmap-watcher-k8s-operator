package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	EventSentTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "configmapwatcher_events_sent_total",
			Help: "Total number of events sent to the webhook",
		},
		[]string{"watcher", "namespace"},
	)

	EventsFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "configmapwatcher_events_failed_total",
			Help: "Total number of events failed to send to the webhook",
		},
		[]string{"watcher", "namespace"},
	)

	SendDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "configmapwatcher_send_duration_seconds",
			Help:    "Duration of sending events to the webhook",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"watcher", "namespace"},
	)
)

func init() {
	metrics.Registry.MustRegister(EventSentTotal, EventsFailedTotal, SendDurationSeconds)
}
