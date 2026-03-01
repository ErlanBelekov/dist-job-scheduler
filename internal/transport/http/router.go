package httptransport

import (
	"log/slog"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"

	sloggin "github.com/samber/slog-gin"
)

func NewRouter(logger *slog.Logger, jobHandler *handler.JobHandler, authHandler *handler.AuthHandler, jwtKey []byte) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Security())
	r.Use(sloggin.New(logger))
	r.Use(middleware.Metrics())

	// Public auth routes
	r.POST("/auth/magic-link", authHandler.RequestMagicLink)
	r.GET("/auth/verify", authHandler.Verify)

	// Protected job routes
	jobs := r.Group("/jobs", middleware.Auth(jwtKey))
	jobs.GET("", jobHandler.List)
	jobs.POST("", jobHandler.Create)
	jobs.GET("/:id", jobHandler.GetByID)
	jobs.DELETE("/:id", jobHandler.Cancel)
	jobs.GET("/:id/attempts", jobHandler.ListAttempts)

	return r
}
