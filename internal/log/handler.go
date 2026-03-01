package log

import (
	"context"
	"log/slog"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/requestid"
)

// ContextHandler wraps an slog.Handler and automatically extracts
// request_id from the context of each log record.
type ContextHandler struct {
	inner slog.Handler
}

// NewContextHandler returns a handler that enriches every record with
// context values (currently request_id) before delegating to inner.
func NewContextHandler(inner slog.Handler) *ContextHandler {
	return &ContextHandler{inner: inner}
}

func (h *ContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := requestid.FromContext(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.inner.Handle(ctx, r)
}

func (h *ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ContextHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *ContextHandler) WithGroup(name string) slog.Handler {
	return &ContextHandler{inner: h.inner.WithGroup(name)}
}
