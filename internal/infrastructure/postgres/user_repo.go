package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Upsert(ctx context.Context, clerkID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING`,
		clerkID,
	)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	query := `SELECT id, email, created_at, updated_at FROM users WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
