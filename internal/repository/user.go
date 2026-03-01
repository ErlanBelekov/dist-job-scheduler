package repository

import (
	"context"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

type UserRepository interface {
	FindOrCreate(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	CreateMagicToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	ClaimMagicToken(ctx context.Context, tokenHash string) (*domain.MagicToken, error)
}
