package httptransport

import (
	"github.com/ErlanBelekov/dist-job-scheduler/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func NewRouter(jobHandler *handler.JobHandler) *gin.Engine {
	r := gin.Default()

	// Schedules a job to be executed later, calls UseCase/Service
	r.POST("/jobs", jobHandler.Create)

	return r
}
