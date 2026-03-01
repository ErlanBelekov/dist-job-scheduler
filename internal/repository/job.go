package repository

import (
	"context"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

// UseCase depends on interface, not concrete implementation.
// This way we get: 1) can swap DB later without touching usecase 2) We can pass a mock implementation of interface in tests
type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) (*domain.Job, error)
	GetByID(ctx context.Context, jobID, userID string) (*domain.Job, error)

	// what does the scheduler worker need? Worker to poll, then claim and process the batch
	// Reaper process to find all failed jobs and re-schedule them for another attempt if a retry is possible
	Claim(ctx context.Context, workerID string, limit int) ([]*domain.Job, error)
	UpdateHeartbeat(ctx context.Context, jobID string) error
	Complete(ctx context.Context, jobID string) error
	Fail(ctx context.Context, jobID string, lastError string) error
	Reschedule(ctx context.Context, jobID string, lastError string, retryAt time.Time) error

	// Reaper methods — recover jobs from crashed workers
	RescheduleStale(ctx context.Context, staleCutoff time.Time, limit int) (int, error)
	FailStale(ctx context.Context, staleCutoff time.Time, limit int) (int, error)
}
