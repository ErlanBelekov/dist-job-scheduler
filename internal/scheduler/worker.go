package scheduler

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
)

type Worker struct {
	id           string
	repo         repository.JobRepository
	executor     *Executor
	pollInterval time.Duration
	concurrency  int
}

func NewWorker(repo repository.JobRepository, pollInterval time.Duration, concurrency int) *Worker {
	hostname, _ := os.Hostname()
	return &Worker{
		id:           fmt.Sprintf("%s-%d", hostname, os.Getpid()),
		repo:         repo,
		executor:     NewExecutor(),
		pollInterval: pollInterval,
		concurrency:  concurrency,
	}
}

func (w *Worker) Start(ctx context.Context) {
	// every N seconds, process a batch of jobs by claiming them and running(using executor)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	log.Printf("worker %s started (concurrency=%d)", w.id, w.concurrency)

	for {
		select {
		case <-ctx.Done():
			log.Printf("worker %s: shut down", w.id)
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	jobs, err := w.repo.Claim(ctx, w.id, w.concurrency)
	if err != nil {
		log.Printf("worker: claim error: %v", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	log.Printf("worker: claimed %d jobs", len(jobs))

	// for each job we claimed, run/execute it in separate goroutine
	// this worker goroutine will be blocked until they are executed/timed out
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(j *domain.Job) {
			defer wg.Done()
			w.runJob(ctx, j)
		}(job)
	}
	wg.Wait()
}

func (w *Worker) runJob(ctx context.Context, job *domain.Job) {
	// run heartbeat in background while job is being executed
	heartbeatCtx, cancelHeartbeat := context.WithCancel(ctx)
	defer cancelHeartbeat()
	go w.heartbeat(heartbeatCtx, job.ID)

	log.Printf("worker %s: executing job %s -> %s %s", w.id, job.ID, job.Method, job.URL)

	result := w.executor.Run(ctx, job)

	if result.Err == nil && result.StatusCode == http.StatusOK {
		if err := w.repo.Complete(ctx, job.ID); err != nil {
			log.Printf("worker %s: complete job failed %s: %v", w.id, job.ID, err)
		}
		log.Printf("worker %s: job %s completed in %s", w.id, job.ID, result.Duration)
		return
	}

	// build error message
	errMsg := ""
	if result.Err != nil {
		errMsg = result.Err.Error()
	} else {
		errMsg = fmt.Sprintf("unexpected status code: %d", result.StatusCode)
	}

	// try to retry
	if job.RetryCount < job.MaxRetries {
		retryAt := time.Now().Add(retryDelay(job.Backoff, job.RetryCount))
		if err := w.repo.Reschedule(ctx, job.ID, errMsg, retryAt); err != nil {
			log.Printf("worker %s: reschedule job %s: error %v", w.id, job.ID, err)
		}
		log.Printf("worker %s: job %s failed, retry %d/%d at %s", w.id, job.ID, job.RetryCount+1, job.MaxRetries, retryAt)
	} else {
		if err := w.repo.Fail(ctx, job.ID, errMsg); err != nil {
			log.Printf("worker %s: fail job %s: %v", w.id, job.ID, err)
		}
		log.Printf("worker %s: job %s permanently failed %s", w.id, job.ID, errMsg)
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
			log.Printf("worker %s: heartbeat update", w.id)
			if err := w.repo.UpdateHeartbeat(ctx, jobID); err != nil {
				log.Printf("worker %s: heartbeat failed %v", err)
			}
		}
	}
}

func retryDelay(backoff domain.Backoff, retryCount int) time.Duration {
	base := 30 * time.Second
	switch backoff {
	case domain.BackoffExponential:
		delay := time.Duration(float64(base) * math.Pow(2, float64(retryCount)))
		delay = min(delay, time.Hour) // upper bound for a retry is 1 hour
		// jitter: +- 25% to avoid thundering herd
		jitter := time.Duration(rand.Int63n(int64(delay/2))) - delay/4
		return delay + jitter
	case domain.BackoffLinear:
		return base * time.Duration(retryCount+1)
	default:
		return base
	}
}
