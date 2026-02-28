package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/email"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultTokenTTL = 15 * time.Minute
	defaultJWTTTL   = 24 * time.Hour
)

type AuthUsecase struct {
	users         repository.UserRepository
	email         email.Sender
	jwtKey        []byte
	tokenTTL      time.Duration
	jwtTTL        time.Duration
	magicLinkBase string
}

func NewAuthUsecase(users repository.UserRepository, emailSender email.Sender, jwtKey []byte, magicLinkBase string) *AuthUsecase {
	return &AuthUsecase{
		users:         users,
		email:         emailSender,
		jwtKey:        jwtKey,
		tokenTTL:      defaultTokenTTL,
		jwtTTL:        defaultJWTTTL,
		magicLinkBase: magicLinkBase,
	}
}

// RequestMagicLink finds or creates the user, generates a secure token,
// stores its hash, and emails the verify link.
func (u *AuthUsecase) RequestMagicLink(ctx context.Context, emailAddr string) error {
	user, err := u.users.FindOrCreate(ctx, emailAddr)
	if err != nil {
		return fmt.Errorf("find or create user: %w", err)
	}

	raw := make([]byte, 32)
	if _, err = io.ReadFull(rand.Reader, raw); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	rawToken := hex.EncodeToString(raw)
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))

	expiresAt := time.Now().Add(u.tokenTTL)
	if err = u.users.CreateMagicToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("store magic token: %w", err)
	}

	link := u.magicLinkBase + "/auth/verify?token=" + rawToken
	subject := "Your sign-in link"
	body := fmt.Sprintf(
		`<p>Click the link below to sign in (expires in 15 minutes):</p><p><a href="%s">%s</a></p>`,
		link, link,
	)
	if err = u.email.Send(ctx, emailAddr, subject, body); err != nil {
		return fmt.Errorf("send magic link: %w", err)
	}
	return nil
}

// VerifyMagicLink hashes the raw token, atomically claims it, and returns a signed JWT.
func (u *AuthUsecase) VerifyMagicLink(ctx context.Context, rawToken string) (string, error) {
	tokenHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))

	mt, err := u.users.ClaimMagicToken(ctx, tokenHash)
	if err != nil {
		return "", domain.ErrTokenInvalid
	}

	user, err := u.users.FindByID(ctx, mt.UserID)
	if err != nil {
		return "", fmt.Errorf("find user: %w", err)
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"iat":   now.Unix(),
		"exp":   now.Add(u.jwtTTL).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := t.SignedString(u.jwtKey)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}
