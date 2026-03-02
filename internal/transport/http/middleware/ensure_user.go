package middleware

import (
	"log/slog"
	"net/http"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
	"github.com/gin-gonic/gin"
)

// EnsureUser runs after Auth. It upserts the Clerk user ID into the users
// table so that jobs/schedules FK constraints are always satisfied.
func EnsureUser(repo repository.UserRepository, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")
		if err := repo.Upsert(c.Request.Context(), userID); err != nil {
			logger.ErrorContext(c.Request.Context(), "ensure user upsert", "error", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				gin.H{"error": "Internal server error"})
			return
		}
		c.Next()
	}
}
