package domain

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrTokenInvalid  = errors.New("token is invalid or expired")
	ErrUnauthorized  = errors.New("unauthorized")
)

type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MagicToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
