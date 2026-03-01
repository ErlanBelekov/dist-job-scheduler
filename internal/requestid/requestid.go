package requestid

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey struct{}

// New generates a random UUID v4 request ID.
func New() string {
	return uuid.NewString()
}

// WithRequestID returns a copy of ctx with the request ID attached.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// FromContext extracts the request ID from ctx. Returns "" if absent.
func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}
