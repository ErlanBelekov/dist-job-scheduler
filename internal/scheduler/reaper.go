package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
)

type Reaper struct {
	repo             repository.JobRepository
	interval         time.Duration
	heartbeatTimeout time.Duration
}

func NewReaper(repo repository.JobRepository, interval time.Duration, heartbeatTimeout time.Duration) *Reaper {
	return &Reaper{
		repo:             repo,
		interval:         interval,
		heartbeatTimeout: heartbeatTimeout,
	}
}

func (r *Reaper) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	log.Printf("reaper started (interval=%s, heartbeat_timeout=%s)", r.interval, r.heartbeatTimeout)

	for {
		select {
		case <-ctx.Done():
			log.Println("reaper: shut down")
			return
		case <-ticker.C:
			r.reap(ctx)
		}
	}
}

func (r *Reaper) reap(ctx context.Context) {
	staleCutoff := time.Now().Add(-r.heartbeatTimeout)

	rescheduled, err := r.repo.RescheduleStale(ctx, staleCutoff, 100)
	if err != nil {
		log.Printf("reaper: reschedule stale: %v", err)
	} else if rescheduled > 0 {
		log.Printf("reaper: rescheduled %d stale jobs", rescheduled)
	}

	failed, err := r.repo.FailStale(ctx, staleCutoff, 100)
	if err != nil {
		log.Printf("reaper: fail stale: %v", err)
	} else if failed > 0 {
		log.Printf("reaper: permanently failed %d stale jobs (max retries exceeded)", failed)
	}
}
