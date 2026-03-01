package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/metrics"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
)

type Worker struct {
	id           string
	repo         repository.JobRepository
	attempts     repository.AttemptRepository
	executor     *Executor
	logger       *slog.Logger
	pollInterval time.Duration
	concurrency  int
	sem          chan struct{}
}

func NewWorker(
	repo repository.JobRepository,
	attempts repository.AttemptRepository,
	logger *slog.Logger,
	pollInterval time.Duration,
	concurrency int,
) *Worker {
	hostname, _ := os.Hostname()
	id := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	return &Worker{
		id:           id,
		repo:         repo,
		attempts:     attempts,
		executor:     NewExecutor(),
		logger:       logger.With("worker_id", id),
		pollInterval: pollInterval,
		concurrency:  concurrency,
		sem:          make(chan struct{}, concurrency),
	}
}

func (w *Worker) Start(ctx context.Context) {
	metrics.WorkerStartTime.SetToCurrentTime()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	w.logger.Info("worker started", "concurrency", w.concurrency)

	for {
		select {
		case <-ctx.Done():
			metrics.WorkerShutdownsTotal.Inc()
			w.logger.Info("worker shut down")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	available := cap(w.sem) - len(w.sem)
	if available == 0 {
		return
	}

	jobs, err := w.repo.Claim(ctx, w.id, available)
	if err != nil {
		w.logger.Error("claim jobs", "error", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	w.logger.Info("claimed jobs", "count", len(jobs), "slots_used", len(w.sem)+len(jobs), "slots_total", cap(w.sem))

	for _, job := range jobs {
		w.sem <- struct{}{}
		go func(j *domain.Job) {
			metrics.JobsInFlight.Inc()
			defer metrics.JobsInFlight.Dec()
			defer func() { <-w.sem }()
			w.runJob(ctx, j)
		}(job)
	}
}

func (w *Worker) runJob(ctx context.Context, job *domain.Job) {
	metrics.JobPickupLatency.Observe(time.Since(job.CreatedAt).Seconds())

	startedAt := time.Now()

	// Open the attempt record before executing so a worker crash leaves a
	// visible incomplete entry (completed_at = NULL) in the history.
	attempt, err := w.attempts.CreateAttempt(ctx, &domain.JobAttempt{
		JobID:      job.ID,
		AttemptNum: job.RetryCount + 1,
		WorkerID:   w.id,
		StartedAt:  startedAt,
	})
	if err != nil {
		// Fatal: if the DB is unhealthy enough to reject this write, all subsequent
		// writes (Complete/Reschedule/Fail) will fail too. Return now — the job
		// stays in "running" status, the heartbeat stops, and the reaper will
		// reschedule it to "pending" after the stale cutoff.
		w.logger.Error("create attempt record, aborting run — reaper will reschedule", "job_id", job.ID, "error", err)
		return
	}

	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go w.heartbeat(heartbeatCtx, job.ID)

	w.logger.Info("executing job", "job_id", job.ID, "method", job.Method, "url", job.URL)

	result := w.executor.Run(ctx, job)
	durationMS := time.Since(startedAt).Milliseconds()

	if result.Err == nil && result.StatusCode == http.StatusOK {
		metrics.JobExecutionDuration.WithLabelValues("success").Observe(result.Duration.Seconds())
		metrics.JobsCompletedTotal.WithLabelValues("success").Inc()
		w.closeAttempt(ctx, attempt, &result.StatusCode, nil, durationMS)
		if err := w.repo.Complete(ctx, job.ID); err != nil {
			w.logger.Error("mark job complete", "job_id", job.ID, "error", err)
		}
		w.logger.Info("job completed", "job_id", job.ID, "duration", result.Duration)
		return
	}

	errMsg := ""
	if result.Err != nil {
		errMsg = result.Err.Error()
	} else {
		errMsg = fmt.Sprintf("unexpected status code: %d", result.StatusCode)
	}

	var statusCode *int
	if result.StatusCode != 0 {
		statusCode = &result.StatusCode
	}
	metrics.JobExecutionDuration.WithLabelValues("failure").Observe(result.Duration.Seconds())
	w.closeAttempt(ctx, attempt, statusCode, &errMsg, durationMS)

	if job.RetryCount < job.MaxRetries {
		retryAt := time.Now().Add(retryDelay(job.Backoff, job.RetryCount))
		if err := w.repo.Reschedule(ctx, job.ID, errMsg, retryAt); err != nil {
			w.logger.Error("reschedule job", "job_id", job.ID, "error", err)
		}
		metrics.JobsCompletedTotal.WithLabelValues("retry").Inc()
		w.logger.Warn("job failed, will retry",
			"job_id", job.ID,
			"error", errMsg,
			"attempt", job.RetryCount+1,
			"max_retries", job.MaxRetries,
			"retry_at", retryAt,
		)
	} else {
		if err := w.repo.Fail(ctx, job.ID, errMsg); err != nil {
			w.logger.Error("mark job failed", "job_id", job.ID, "error", err)
		}
		metrics.JobsCompletedTotal.WithLabelValues("failed").Inc()
		w.logger.Warn("job permanently failed", "job_id", job.ID, "error", errMsg)
	}
}

// closeAttempt writes the execution outcome to the attempt record.
func (w *Worker) closeAttempt(ctx context.Context, attempt *domain.JobAttempt, statusCode *int, errMsg *string, durationMS int64) {
	if err := w.attempts.CompleteAttempt(ctx, attempt.ID, statusCode, errMsg, durationMS); err != nil {
		w.logger.Error("complete attempt record", "job_id", attempt.JobID, "error", err)
	}
}

func (w *Worker) heartbeat(ctx context.Context, jobID string) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.repo.UpdateHeartbeat(ctx, jobID); err != nil {
				w.logger.Warn("heartbeat failed", "job_id", jobID, "error", err)
			}
		}
	}
}

func retryDelay(backoff domain.Backoff, retryCount int) time.Duration {
	base := 30 * time.Second
	switch backoff {
	case domain.BackoffExponential:
		delay := time.Duration(float64(base) * math.Pow(2, float64(retryCount)))
		delay = min(delay, time.Hour)
		jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
		return delay + jitter
	case domain.BackoffLinear:
		return base * time.Duration(retryCount+1)
	default:
		return base
	}
}
