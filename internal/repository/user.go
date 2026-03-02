package repository

import (
	"context"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

type UserRepository interface {
	Upsert(ctx context.Context, clerkID string) error
	FindByID(ctx context.Context, id string) (*domain.User, error)
}
