package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
	"log/slog"
	"os"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeAuthUsecase implements the unexported authUsecaser interface via method matching.
type fakeAuthUsecase struct {
	requestMagicLink func(ctx context.Context, email string) error
	verifyMagicLink  func(ctx context.Context, rawToken string) (string, error)
}

func (f *fakeAuthUsecase) RequestMagicLink(ctx context.Context, email string) error {
	return f.requestMagicLink(ctx, email)
}

func (f *fakeAuthUsecase) VerifyMagicLink(ctx context.Context, rawToken string) (string, error) {
	return f.verifyMagicLink(ctx, rawToken)
}

func newTestEngine(uc *fakeAuthUsecase) *gin.Engine {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := handler.NewAuthHandler(uc, logger)

	r := gin.New()
	r.POST("/auth/magic-link", h.RequestMagicLink)
	r.GET("/auth/verify", h.Verify)
	return r
}

// ---- RequestMagicLink ----

func TestRequestMagicLink_InvalidJSON_Returns400(t *testing.T) {
	uc := &fakeAuthUsecase{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link", strings.NewReader(`{bad json}`))
	req.Header.Set("Content-Type", "application/json")
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRequestMagicLink_InvalidEmail_Returns400(t *testing.T) {
	uc := &fakeAuthUsecase{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link",
		strings.NewReader(`{"email":"not-an-email"}`))
	req.Header.Set("Content-Type", "application/json")
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRequestMagicLink_UsecaseError_StillReturns200(t *testing.T) {
	uc := &fakeAuthUsecase{
		requestMagicLink: func(_ context.Context, _ string) error {
			return errors.New("internal failure")
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link",
		strings.NewReader(`{"email":"test@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (must not reveal errors)", w.Code)
	}
}

func TestRequestMagicLink_Success_Returns200(t *testing.T) {
	uc := &fakeAuthUsecase{
		requestMagicLink: func(_ context.Context, _ string) error { return nil },
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/auth/magic-link",
		strings.NewReader(`{"email":"test@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ---- Verify ----

func TestVerify_MissingToken_Returns401(t *testing.T) {
	uc := &fakeAuthUsecase{}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestVerify_InvalidToken_Returns401(t *testing.T) {
	uc := &fakeAuthUsecase{
		verifyMagicLink: func(_ context.Context, _ string) (string, error) {
			return "", domain.ErrTokenInvalid
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=bad", nil)
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestVerify_InternalError_Returns401(t *testing.T) {
	uc := &fakeAuthUsecase{
		verifyMagicLink: func(_ context.Context, _ string) (string, error) {
			return "", errors.New("db down")
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=sometoken", nil)
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestVerify_ValidToken_Returns200WithJWT(t *testing.T) {
	const fakeJWT = "header.payload.signature"
	uc := &fakeAuthUsecase{
		verifyMagicLink: func(_ context.Context, _ string) (string, error) {
			return fakeJWT, nil
		},
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/verify?token=validtoken", nil)
	newTestEngine(uc).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	if !strings.Contains(w.Body.String(), fakeJWT) {
		t.Errorf("body %q does not contain JWT %q", w.Body.String(), fakeJWT)
	}
}
