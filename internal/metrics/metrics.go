package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Worker metrics

	JobPickupLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "scheduler",
		Name:      "job_pickup_latency_seconds",
		Help:      "Time from job creation to worker claiming it.",
		Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 120, 300},
	})

	JobExecutionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "scheduler",
		Name:      "job_execution_duration_seconds",
		Help:      "Duration of job HTTP execution.",
		Buckets:   []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"status"})

	JobsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "scheduler",
		Name:      "worker_jobs_in_flight",
		Help:      "Number of jobs currently being executed by the worker.",
	})

	JobsCompletedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scheduler",
		Name:      "jobs_completed_total",
		Help:      "Total jobs finished, by outcome.",
	}, []string{"outcome"})

	// Reaper metrics

	ReaperRescuedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scheduler",
		Name:      "reaper_rescued_total",
		Help:      "Total stale jobs handled by the reaper.",
	}, []string{"action"})

	ReaperCycleDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "scheduler",
		Name:      "reaper_cycle_duration_seconds",
		Help:      "Time taken for one reaper cycle.",
		Buckets:   prometheus.DefBuckets,
	})

	// Worker lifecycle

	WorkerStartTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "scheduler",
		Name:      "worker_start_time_seconds",
		Help:      "Unix timestamp when the worker started.",
	})

	WorkerShutdownsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "scheduler",
		Name:      "worker_shutdowns_total",
		Help:      "Number of times the worker has shut down.",
	})

	// HTTP metrics

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "scheduler",
		Name:      "http_request_duration_seconds",
		Help:      "HTTP request latency.",
		Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"method", "path", "status"})

	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scheduler",
		Name:      "http_requests_total",
		Help:      "Total HTTP requests.",
	}, []string{"method", "path", "status"})
)

func Register() {
	prometheus.MustRegister(
		JobPickupLatency,
		JobExecutionDuration,
		JobsInFlight,
		JobsCompletedTotal,
		ReaperRescuedTotal,
		ReaperCycleDuration,
		WorkerStartTime,
		WorkerShutdownsTotal,
		HTTPRequestDuration,
		HTTPRequestsTotal,
	)
}

func NewServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return &http.Server{Addr: addr, Handler: mux}
}
