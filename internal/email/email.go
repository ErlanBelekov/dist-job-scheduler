package email

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/resend/resend-go/v2"
)

type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogSender logs emails instead of sending them — used in ENV=local.
type LogSender struct {
	logger *slog.Logger
}

func (s *LogSender) Send(_ context.Context, to, subject, body string) error {
	s.logger.Info("magic link email (local dev)", "to", to, "subject", subject, "body", body)
	return nil
}

// ResendSender sends emails via the Resend API — used in staging/production.
type ResendSender struct {
	client *resend.Client
	from   string
}

func (s *ResendSender) Send(ctx context.Context, to, subject, body string) error {
	params := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Html:    body,
	}
	_, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// NewSender returns a LogSender for ENV=local, ResendSender otherwise.
func NewSender(env, apiKey, from string, logger *slog.Logger) Sender {
	if env == "local" {
		return &LogSender{logger: logger}
	}
	return &ResendSender{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}
