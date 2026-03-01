package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/gin-gonic/gin"
)

// authUsecaser is the subset of AuthUsecase the handler needs.
// Defined here (point of use) so tests can inject a fake.
type authUsecaser interface {
	RequestMagicLink(ctx context.Context, email string) error
	VerifyMagicLink(ctx context.Context, rawToken string) (string, error)
}

type AuthHandler struct {
	authUsecase authUsecaser
	logger      *slog.Logger
}

func NewAuthHandler(authUsecase authUsecaser, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		logger:      logger.With("component", "auth_handler"),
	}
}

type magicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// POST /auth/magic-link
// Always returns 200 to avoid revealing whether the email exists.
func (h *AuthHandler) RequestMagicLink(c *gin.Context) {
	var req magicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.authUsecase.RequestMagicLink(c.Request.Context(), req.Email); err != nil {
		h.logger.ErrorContext(c.Request.Context(), "request magic link", "error", err)
	}

	c.Status(http.StatusOK)
}

// GET /auth/verify?token=<raw>
// Returns {"token": "<jwt>"} on success, 401 on invalid/expired token.
func (h *AuthHandler) Verify(c *gin.Context) {
	rawToken := c.Query("token")
	if rawToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": errTokenInvalid})
		return
	}

	jwtToken, err := h.authUsecase.VerifyMagicLink(c.Request.Context(), rawToken)
	if err != nil {
		if !errors.Is(err, domain.ErrTokenInvalid) {
			h.logger.ErrorContext(c.Request.Context(), "verify magic link", "error", err)
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": errTokenInvalid})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": jwtToken})
}
