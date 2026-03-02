package domain

import (
	"errors"
	"time"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUnauthorized = errors.New("unauthorized")
)

type User struct {
	ID        string
	Email     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
