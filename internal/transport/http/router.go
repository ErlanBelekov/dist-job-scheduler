package httptransport

import (
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func NewRouter(jobHandler *handler.JobHandler, authHandler *handler.AuthHandler, jwtKey []byte) *gin.Engine {
	r := gin.Default()

	// Public auth routes
	r.POST("/auth/magic-link", authHandler.RequestMagicLink)
	r.GET("/auth/verify", authHandler.Verify)

	// Protected job routes
	jobs := r.Group("/jobs", middleware.Auth(jwtKey))
	jobs.POST("", jobHandler.Create)
	jobs.GET("/:id", jobHandler.GetByID)

	return r
}
