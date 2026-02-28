package usecase_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
)

// ---- fakes ----

type fakeUserRepo struct {
	findOrCreate     func(ctx context.Context, email string) (*domain.User, error)
	findByID         func(ctx context.Context, id string) (*domain.User, error)
	createMagicToken func(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	claimMagicToken  func(ctx context.Context, tokenHash string) (*domain.MagicToken, error)
}

func (r *fakeUserRepo) FindOrCreate(ctx context.Context, email string) (*domain.User, error) {
	return r.findOrCreate(ctx, email)
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return r.findByID(ctx, id)
}

func (r *fakeUserRepo) CreateMagicToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	return r.createMagicToken(ctx, userID, tokenHash, expiresAt)
}

func (r *fakeUserRepo) ClaimMagicToken(ctx context.Context, tokenHash string) (*domain.MagicToken, error) {
	return r.claimMagicToken(ctx, tokenHash)
}

type fakeEmailSender struct {
	send func(ctx context.Context, to, subject, body string) error
}

func (s *fakeEmailSender) Send(ctx context.Context, to, subject, body string) error {
	return s.send(ctx, to, subject, body)
}

// ---- helpers ----

const (
	testJWTKey        = "test-jwt-secret-at-least-32-chars!!"
	testMagicLinkBase = "http://localhost:8080"
)

func newUsecase(repo *fakeUserRepo, sender *fakeEmailSender) *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(repo, sender, []byte(testJWTKey), testMagicLinkBase)
}

var testUser = &domain.User{ID: "user-1", Email: "test@example.com"}

// ---- RequestMagicLink ----

func TestRequestMagicLink_StoresHashOfEmailedToken(t *testing.T) {
	var capturedHash string
	var capturedBody string

	repo := &fakeUserRepo{
		findOrCreate: func(_ context.Context, _ string) (*domain.User, error) {
			return testUser, nil
		},
		createMagicToken: func(_ context.Context, _, tokenHash string, _ time.Time) error {
			capturedHash = tokenHash
			return nil
		},
	}
	sender := &fakeEmailSender{
		send: func(_ context.Context, _, _, body string) error {
			capturedBody = body
			return nil
		},
	}

	if err := newUsecase(repo, sender).RequestMagicLink(context.Background(), testUser.Email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract the raw token from the link embedded in the email body.
	idx := strings.Index(capturedBody, "?token=")
	if idx == -1 {
		t.Fatal("email body does not contain ?token=")
	}
	rawToken := strings.SplitN(capturedBody[idx+len("?token="):], `"`, 2)[0]

	wantHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))
	if capturedHash != wantHash {
		t.Errorf("stored hash %q != SHA-256 of emailed token %q", capturedHash, wantHash)
	}
}

func TestRequestMagicLink_TokenExpiresInFuture(t *testing.T) {
	var capturedExpiry time.Time

	repo := &fakeUserRepo{
		findOrCreate: func(_ context.Context, _ string) (*domain.User, error) {
			return testUser, nil
		},
		createMagicToken: func(_ context.Context, _, _ string, expiresAt time.Time) error {
			capturedExpiry = expiresAt
			return nil
		},
	}
	sender := &fakeEmailSender{
		send: func(_ context.Context, _, _, _ string) error { return nil },
	}

	before := time.Now()
	if err := newUsecase(repo, sender).RequestMagicLink(context.Background(), testUser.Email); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedExpiry.After(before) {
		t.Errorf("expiry %v is not after request time %v", capturedExpiry, before)
	}
}

func TestRequestMagicLink_RepoError_Propagates(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeUserRepo{
		findOrCreate: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, repoErr
		},
	}
	sender := &fakeEmailSender{}

	err := newUsecase(repo, sender).RequestMagicLink(context.Background(), testUser.Email)
	if !errors.Is(err, repoErr) {
		t.Errorf("want wrapped repoErr, got %v", err)
	}
}

func TestRequestMagicLink_EmailError_Propagates(t *testing.T) {
	sendErr := errors.New("smtp unavailable")
	repo := &fakeUserRepo{
		findOrCreate: func(_ context.Context, _ string) (*domain.User, error) {
			return testUser, nil
		},
		createMagicToken: func(_ context.Context, _, _ string, _ time.Time) error {
			return nil
		},
	}
	sender := &fakeEmailSender{
		send: func(_ context.Context, _, _, _ string) error { return sendErr },
	}

	err := newUsecase(repo, sender).RequestMagicLink(context.Background(), testUser.Email)
	if !errors.Is(err, sendErr) {
		t.Errorf("want wrapped sendErr, got %v", err)
	}
}

// ---- VerifyMagicLink ----

func TestVerifyMagicLink_ReturnsSignedJWT(t *testing.T) {
	const rawToken = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	expectedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawToken)))

	mt := &domain.MagicToken{ID: "mt-1", UserID: testUser.ID, TokenHash: expectedHash}
	repo := &fakeUserRepo{
		claimMagicToken: func(_ context.Context, tokenHash string) (*domain.MagicToken, error) {
			if tokenHash != expectedHash {
				return nil, domain.ErrTokenInvalid
			}
			return mt, nil
		},
		findByID: func(_ context.Context, _ string) (*domain.User, error) {
			return testUser, nil
		},
	}
	sender := &fakeEmailSender{}

	signed, err := newUsecase(repo, sender).VerifyMagicLink(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	token, parseErr := jwt.Parse(signed, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected method")
		}
		return []byte(testJWTKey), nil
	})
	if parseErr != nil || !token.Valid {
		t.Fatalf("returned JWT is invalid: %v", parseErr)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("could not cast claims")
	}
	if claims["sub"] != testUser.ID {
		t.Errorf("sub = %v, want %q", claims["sub"], testUser.ID)
	}
	if claims["email"] != testUser.Email {
		t.Errorf("email = %v, want %q", claims["email"], testUser.Email)
	}
}

func TestVerifyMagicLink_InvalidToken_ReturnsErrTokenInvalid(t *testing.T) {
	repo := &fakeUserRepo{
		claimMagicToken: func(_ context.Context, _ string) (*domain.MagicToken, error) {
			return nil, domain.ErrTokenInvalid
		},
	}
	sender := &fakeEmailSender{}

	_, err := newUsecase(repo, sender).VerifyMagicLink(context.Background(), "bad-token")
	if !errors.Is(err, domain.ErrTokenInvalid) {
		t.Errorf("want ErrTokenInvalid, got %v", err)
	}
}
