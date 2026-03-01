package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/metrics"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
)

type Reaper struct {
	repo             repository.JobRepository
	logger           *slog.Logger
	interval         time.Duration
	heartbeatTimeout time.Duration
}

func NewReaper(repo repository.JobRepository, logger *slog.Logger, interval time.Duration, heartbeatTimeout time.Duration) *Reaper {
	return &Reaper{
		repo:             repo,
		logger:           logger,
		interval:         interval,
		heartbeatTimeout: heartbeatTimeout,
	}
}

func (r *Reaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	r.logger.InfoContext(ctx, "reaper started", "interval", r.interval, "heartbeat_timeout", r.heartbeatTimeout)

	for {
		select {
		case <-ctx.Done():
			r.logger.InfoContext(ctx, "reaper shut down")
			return
		case <-ticker.C:
			r.reap(ctx)
		}
	}
}

func (r *Reaper) reap(ctx context.Context) {
	start := time.Now()
	defer func() {
		metrics.ReaperCycleDuration.Observe(time.Since(start).Seconds())
	}()

	staleCutoff := time.Now().Add(-r.heartbeatTimeout)

	rescheduled, err := r.repo.RescheduleStale(ctx, staleCutoff, 100)
	if err != nil {
		r.logger.ErrorContext(ctx, "reschedule stale jobs", "error", err)
	} else if rescheduled > 0 {
		metrics.ReaperRescuedTotal.WithLabelValues("rescheduled").Add(float64(rescheduled))
		r.logger.InfoContext(ctx, "rescheduled stale jobs", "count", rescheduled)
	}

	failed, err := r.repo.FailStale(ctx, staleCutoff, 100)
	if err != nil {
		r.logger.ErrorContext(ctx, "fail stale jobs", "error", err)
	} else if failed > 0 {
		metrics.ReaperRescuedTotal.WithLabelValues("failed").Add(float64(failed))
		r.logger.InfoContext(ctx, "permanently failed stale jobs", "count", failed)
	}
}
