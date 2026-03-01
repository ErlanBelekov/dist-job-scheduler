package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *UserRepository) FindOrCreate(ctx context.Context, email string) (*domain.User, error) {
	query := `
		INSERT INTO users (email)
		VALUES ($1)
		ON CONFLICT (email) DO UPDATE SET updated_at = NOW()
		RETURNING id, email, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query, email)
	return scanUser(row)
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

func (r *UserRepository) CreateMagicToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO magic_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create magic token: %w", err)
	}
	return nil
}

// ClaimMagicToken atomically marks the token as used and returns it.
// Returns domain.ErrTokenInvalid if the token does not exist, is already used, or is expired.
func (r *UserRepository) ClaimMagicToken(ctx context.Context, tokenHash string) (*domain.MagicToken, error) {
	query := `
		UPDATE magic_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at`

	row := r.pool.QueryRow(ctx, query, tokenHash)
	return scanMagicToken(row)
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

func scanMagicToken(row pgx.Row) (*domain.MagicToken, error) {
	var t domain.MagicToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenInvalid
		}
		return nil, fmt.Errorf("scan magic token: %w", err)
	}
	return &t, nil
}
