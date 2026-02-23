package repository

import (
	"context"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

// UseCase depends on interface, not concrete implementation.
// This way we get: 1) can swap DB later without touching usecase 2) We can pass a mock implementation of interface in tests
type JobRepository interface {
	Create(ctx context.Context, job *domain.Job) (*domain.Job, error)
	GetByID(ctx context.Context, id string) (*domain.Job, error)
}
