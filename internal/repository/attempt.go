package repository

import (
	"context"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

type AttemptRepository interface {
	// CreateAttempt opens an attempt record at the moment execution starts.
	// Returns the persisted attempt (with its DB-generated ID) so the caller
	// can close it with CompleteAttempt once the job finishes.
	CreateAttempt(ctx context.Context, attempt *domain.JobAttempt) (*domain.JobAttempt, error)

	// CompleteAttempt closes an open attempt record with the execution outcome.
	// statusCode is nil when the HTTP request never received a response.
	// errMsg is nil on success.
	CompleteAttempt(ctx context.Context, id string, statusCode *int, errMsg *string, durationMS int64) error

	// ListByJobID returns all attempts for a job, ordered by started_at ASC.
	// Ownership is assumed to have been verified by the caller.
	ListByJobID(ctx context.Context, jobID string) ([]*domain.JobAttempt, error)
}
